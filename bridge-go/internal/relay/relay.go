package relay

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
)

const (
	LoopbackAddress = "127.0.0.1:8765"
	dockerHost      = "172.17.0.1"
	dockerPortFirst = 8766
	dockerPortLast  = 8770
	dockerPrefix    = "172.16.0.0/12"
	retryInterval   = 5 * time.Second
	handshakeLimit  = 64 * 1024
)

var handshakeMarker = []byte{0, 'C', 'O', 'D', 'Y', '-', 'R', 'E', 'L', 'A', 'Y', 1, '\n'}

type Logger interface {
	Infof(string, ...any)
	Warningf(string, ...any)
	Debugf(string, ...any)
}

func WriteHandshake(writer io.Writer) error {
	written, err := writer.Write(handshakeMarker)
	if err != nil {
		return fmt.Errorf("write relay handshake: %w", err)
	}
	if written != len(handshakeMarker) {
		return io.ErrShortWrite
	}
	return nil
}

func ReadHandshake(reader io.Reader) (io.Reader, int, error) {
	buffered := bufio.NewReader(reader)
	matched := 0
	for consumed := 1; consumed <= handshakeLimit; consumed++ {
		value, err := buffered.ReadByte()
		if err != nil {
			return nil, consumed - 1, fmt.Errorf("read relay handshake: %w", err)
		}
		if value == handshakeMarker[matched] {
			matched++
			if matched == len(handshakeMarker) {
				return buffered, consumed - len(handshakeMarker), nil
			}
			continue
		}
		if value == handshakeMarker[0] {
			matched = 1
		} else {
			matched = 0
		}
	}
	return nil, handshakeLimit, errors.New("relay handshake marker not found")
}

func RunClient(ctx context.Context, transport io.ReadWriteCloser, logger Logger) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	session, err := yamux.Client(transport, quietYamuxConfig())
	if err != nil {
		return fmt.Errorf("create relay session: %w", err)
	}
	defer session.Close()

	listenLoopback(runCtx, session, logger)
	go retryDockerListener(runCtx, session, logger)

	select {
	case <-ctx.Done():
		return nil
	case <-session.CloseChan():
		if ctx.Err() != nil {
			return nil
		}
		return errors.New("relay transport closed")
	}
}

func RunServer(
	ctx context.Context,
	transport io.ReadWriteCloser,
	bridgeAddress string,
	logger Logger,
) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	session, err := yamux.Server(transport, quietYamuxConfig())
	if err != nil {
		return fmt.Errorf("create relay session: %w", err)
	}
	defer session.Close()

	go func() {
		<-runCtx.Done()
		_ = session.Close()
	}()

	for {
		stream, err := session.Accept()
		if err != nil {
			if runCtx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept relay stream: %w", err)
		}
		go forwardToBridge(runCtx, stream, bridgeAddress, logger)
	}
}

func quietYamuxConfig() *yamux.Config {
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	return config
}

func listenLoopback(ctx context.Context, session *yamux.Session, logger Logger) {
	listener, err := net.Listen("tcp", LoopbackAddress)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			logger.Infof("relay loopback listener unavailable; using Docker listener if available: %v", err)
			return
		}
		logger.Warningf("relay listen %s: %v", LoopbackAddress, err)
		return
	}
	logger.Infof("relay listening on %s", LoopbackAddress)
	go acceptConnections(ctx, listener, session, logger)
}

func retryDockerListener(ctx context.Context, session *yamux.Session, logger Logger) {
	waitingForDocker := false
	for {
		listener, address, err := listenFirstAvailable(
			dockerHost,
			dockerPortFirst,
			dockerPortLast,
			net.Listen,
		)
		if err == nil {
			waitingForDocker = false
			logger.Infof("relay listening on %s", address)
			acceptConnections(ctx, listener, session, logger)
			if ctx.Err() != nil || session.IsClosed() {
				return
			}
		} else if errors.Is(err, syscall.EADDRNOTAVAIL) {
			if !waitingForDocker {
				logger.Infof("relay waiting for Docker bridge address %s", dockerHost)
				waitingForDocker = true
			}
		} else {
			logger.Warningf(
				"relay listen %s:%d-%d: %v",
				dockerHost,
				dockerPortFirst,
				dockerPortLast,
				err,
			)
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-session.CloseChan():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func listenFirstAvailable(
	host string,
	firstPort, lastPort int,
	listen func(string, string) (net.Listener, error),
) (net.Listener, string, error) {
	var conflicts []error
	for port := firstPort; port <= lastPort; port++ {
		address := net.JoinHostPort(host, strconv.Itoa(port))
		listener, err := listen("tcp", address)
		if err == nil {
			return listener, address, nil
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			return nil, "", fmt.Errorf("listen %s: %w", address, err)
		}
		conflicts = append(conflicts, fmt.Errorf("%s: %w", address, err))
	}
	return nil, "", fmt.Errorf(
		"no available relay port on %s:%d-%d: %w",
		host,
		firstPort,
		lastPort,
		errors.Join(conflicts...),
	)
}

func acceptConnections(
	ctx context.Context,
	listener net.Listener,
	session *yamux.Session,
	logger Logger,
) {
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)
	defer listener.Close()
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || session.IsClosed() {
				return
			}
			logger.Warningf("relay accept on %s: %v", listener.Addr(), err)
			return
		}
		if !allowsRemote(connection.RemoteAddr()) {
			logger.Warningf("relay rejected connection from %s", connection.RemoteAddr())
			_ = connection.Close()
			continue
		}
		go forwardToRelay(ctx, connection, session, logger)
	}
}

func allowsRemote(address net.Addr) bool {
	tcpAddress, ok := address.(*net.TCPAddr)
	if !ok {
		return false
	}
	ip, ok := netip.AddrFromSlice(tcpAddress.IP)
	if !ok {
		return false
	}
	ip = ip.Unmap()
	dockerNetwork := netip.MustParsePrefix(dockerPrefix)
	return ip.IsLoopback() || dockerNetwork.Contains(ip)
}

func forwardToRelay(
	ctx context.Context,
	connection net.Conn,
	session *yamux.Session,
	logger Logger,
) {
	defer connection.Close()
	if err := ctx.Err(); err != nil {
		return
	}
	stream, err := session.Open()
	if err != nil {
		logger.Warningf("relay open stream: %v", err)
		return
	}
	defer stream.Close()
	logger.Debugf("relay connection %s", connection.RemoteAddr())
	copyBidirectional(connection, stream)
}

func forwardToBridge(ctx context.Context, stream net.Conn, bridgeAddress string, logger Logger) {
	defer stream.Close()
	if err := ctx.Err(); err != nil {
		return
	}
	connection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(
		ctx,
		"tcp",
		bridgeAddress,
	)
	if err != nil {
		logger.Warningf("relay connect to bridge: %v", err)
		return
	}
	defer connection.Close()
	logger.Debugf("relay connection to bridge")
	copyBidirectional(stream, connection)
}

func copyBidirectional(first, second net.Conn) {
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(first, second)
		_ = first.Close()
		close(done)
	}()
	_, _ = io.Copy(second, first)
	_ = second.Close()
	<-done
}
