package relay

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

type testLogger struct{}

func (testLogger) Infof(string, ...any)    {}
func (testLogger) Warningf(string, ...any) {}
func (testLogger) Debugf(string, ...any)   {}

func Test_req_AGT_12_ConnectionsTunnelThroughYamux(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "relay-ok")
	}))
	defer target.Close()
	targetAddress := strings.TrimPrefix(target.URL, "http://")

	left, right := net.Pipe()
	clientSession, err := yamux.Client(left, quietYamuxConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- RunServer(ctx, right, targetAddress, testLogger{})
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

func startTestListener(t *testing.T, ctx context.Context, session *yamux.Session) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go acceptConnections(ctx, listener, session, testLogger{})
	return listener.Addr().String()
}
