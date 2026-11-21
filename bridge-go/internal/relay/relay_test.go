package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

type testLogger struct{}

func (testLogger) Infof(string, ...any)    {}
func (testLogger) Warningf(string, ...any) {}
func (testLogger) Debugf(string, ...any)   {}

func Test_req_AGT_12_HandshakeSkipsStartupOutputAndPreservesTunnelBytes(t *testing.T) {
	var output bytes.Buffer
	if _, err := output.WriteString("WSL startup message\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := WriteHandshake(&output); err != nil {
		t.Fatal(err)
	}
	const firstTunnelBytes = "\x00\x01\x00\x00"
	if _, err := output.WriteString(firstTunnelBytes); err != nil {
		t.Fatal(err)
	}

	reader, skipped, err := ReadHandshake(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != len("WSL startup message\r\n") {
		t.Fatalf("skipped %d bytes, want %d", skipped, len("WSL startup message\r\n"))
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != firstTunnelBytes {
		t.Fatalf("tunnel bytes = %q, want %q", got, firstTunnelBytes)
	}
}

func Test_req_AGT_12_HandshakeRejectsMissingMarker(t *testing.T) {
	_, _, err := ReadHandshake(bytes.NewReader([]byte("not a relay")))
	if err == nil {
		t.Fatal("ReadHandshake() accepted output without a relay marker")
	}
}

func Test_req_AGT_12_ConnectionsTunnelThroughYamux(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "relay-ok")
	}))
	defer target.Close()
	targetAddress := strings.TrimPrefix(target.URL, "http://")

	left, right := net.Pipe()
	preambleWritten := make(chan error, 1)
	go func() {
		if _, err := io.WriteString(left, "WSL startup message\r\n"); err != nil {
			preambleWritten <- err
			return
		}
		preambleWritten <- WriteHandshake(left)
	}()
	reader, skipped, err := ReadHandshake(right)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-preambleWritten; err != nil {
		t.Fatal(err)
	}
	if skipped == 0 {
		t.Fatal("expected to discard WSL startup output")
	}
	clientSession, err := yamux.Client(left, quietYamuxConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- RunServer(
			ctx,
			bufferedPipe{reader: reader, connection: right},
			targetAddress,
			testLogger{},
		)
	}()

	address := startTestListener(t, ctx, clientSession)
	const concurrentConnections = 2
	results := make(chan error, concurrentConnections)
	for range concurrentConnections {
		go func() {
			connection, err := net.Dial("tcp", address)
			if err != nil {
				results <- err
				return
			}
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(2 * time.Second))
			if _, err := io.WriteString(connection, "GET / HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n"); err != nil {
				results <- err
				return
			}
			response, err := io.ReadAll(connection)
			if err != nil {
				results <- err
				return
			}
			if !strings.Contains(string(response), "relay-ok") {
				results <- fmt.Errorf("tunnel response = %q", response)
				return
			}
			results <- nil
		}()
	}
	for range concurrentConnections {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	if err := <-serverResult; err != nil {
		t.Fatal(err)
	}
}

type bufferedPipe struct {
	reader     io.Reader
	connection net.Conn
}

func (p bufferedPipe) Read(data []byte) (int, error)  { return p.reader.Read(data) }
func (p bufferedPipe) Write(data []byte) (int, error) { return p.connection.Write(data) }
func (p bufferedPipe) Close() error                   { return p.connection.Close() }

func Test_req_BRG_08_OnlyLoopbackAndDockerNetworkClientsAreAllowed(t *testing.T) {
	for _, test := range []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"172.32.0.1", false},
		{"172.15.255.254", false},
		{"192.168.1.1", false},
	} {
		address := &net.TCPAddr{IP: net.ParseIP(test.ip)}
		if got := allowsRemote(address); got != test.want {
			t.Errorf("allowsRemote(%s) = %v, want %v", test.ip, got, test.want)
		}
	}
}

func Test_req_BRG_08_DockerRelaySelectsFirstAvailablePort(t *testing.T) {
	var attempts []string
	listen := func(network, address string) (net.Listener, error) {
		attempts = append(attempts, address)
		if len(attempts) < 3 {
			return nil, syscall.EADDRINUSE
		}
		return net.Listen(network, "127.0.0.1:0")
	}

	listener, address, err := listenFirstAvailable(
		dockerHost,
		dockerPortFirst,
		dockerPortLast,
		listen,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if address != "172.17.0.1:8768" {
		t.Fatalf("selected address = %q, want 172.17.0.1:8768", address)
	}
	if len(attempts) != 3 {
		t.Fatalf("listen attempts = %v, want first three candidates", attempts)
	}
}

func Test_req_BRG_08_DockerRelayReportsWhenAllCandidatePortsAreOccupied(t *testing.T) {
	_, _, err := listenFirstAvailable(
		dockerHost,
		dockerPortFirst,
		dockerPortLast,
		func(string, string) (net.Listener, error) {
			return nil, syscall.EADDRINUSE
		},
	)
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("listenFirstAvailable() error = %v, want address-in-use error", err)
	}
}

func startTestListener(t *testing.T, ctx context.Context, session *yamux.Session) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go acceptConnections(ctx, listener, session, testLogger{})
	return listener.Addr().String()
}
