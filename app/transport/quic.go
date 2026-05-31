package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

type QUICConn struct {
	conn   quic.Connection
	stream quic.Stream
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewQUICConn(ctx context.Context, addr string, port int, settings *StreamSettings) (*QUICConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
		MinVersion:         tls.VersionTLS12,
	}
	if settings.TLSSettings != nil {
		tlsConfig.ServerName = settings.TLSSettings.ServerName
		tlsConfig.InsecureSkipVerify = settings.TLSSettings.AllowInsecure
		if len(settings.TLSSettings.ALPN) > 0 {
			tlsConfig.NextProtos = settings.TLSSettings.ALPN
		}
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
		InitialStreamReceiveWindow:  8 * 1024 * 1024,
		InitialConnectionReceiveWindow: 12 * 1024 * 1024,
	}

	remoteAddr := fmt.Sprintf("%s:%d", addr, port)
	conn, err := quic.DialAddr(ctx, remoteAddr, tlsConfig, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("quic dial: %w", err)
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(0, "")
		return nil, fmt.Errorf("quic open stream: %w", err)
	}

	qCtx, cancel := context.WithCancel(ctx)
	return &QUICConn{
		conn:   conn,
		stream: stream,
		ctx:    qCtx,
		cancel: cancel,
	}, nil
}

func (c *QUICConn) Read(p []byte) (int, error) {
	return c.stream.Read(p)
}

func (c *QUICConn) Write(p []byte) (int, error) {
	return c.stream.Write(p)
}

func (c *QUICConn) Close() error {
	c.cancel()
	c.stream.Close()
	c.conn.CloseWithError(0, "")
	return nil
}

func (c *QUICConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *QUICConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *QUICConn) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

func (c *QUICConn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *QUICConn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}

type QUICListener struct {
	listener  *quic.Listener
	settings  *StreamSettings
	acceptCh  chan *QUICConn
	done      chan struct{}
	mu        sync.Mutex
}

func NewQUICListener(addr string, settings *StreamSettings) (*QUICListener, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
		MinVersion:         tls.VersionTLS12,
	}
	if settings.TLSSettings != nil {
		if len(settings.TLSSettings.Certificates) > 0 {
			cert := settings.TLSSettings.Certificates[0]
			if cert.CertificateFile != "" && cert.KeyFile != "" {
				tlsCert, err := tls.LoadX509KeyPair(cert.CertificateFile, cert.KeyFile)
				if err != nil {
					return nil, err
				}
				tlsConfig.Certificates = []tls.Certificate{tlsCert}
			}
		}
		if len(settings.TLSSettings.ALPN) > 0 {
			tlsConfig.NextProtos = settings.TLSSettings.ALPN
		}
		tlsConfig.ServerName = settings.TLSSettings.ServerName
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
		Allow0RTT:       true,
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	ql, err := quic.Listen(udpConn, tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	l := &QUICListener{
		listener: ql,
		settings: settings,
		acceptCh: make(chan *QUICConn, 16),
		done:     make(chan struct{}),
	}

	go l.acceptLoop()
	return l, nil
}

func (l *QUICListener) acceptLoop() {
	for {
		conn, err := l.listener.Accept(context.Background())
		if err != nil {
			select {
			case <-l.done:
				return
			default:
			}
			continue
		}
		go l.handleConn(conn)
	}
}

func (l *QUICListener) handleConn(conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		qc := &QUICConn{
			conn:   conn,
			stream: stream,
			ctx:    context.Background(),
			cancel: func() {},
		}
		select {
		case l.acceptCh <- qc:
		default:
			stream.Close()
		}
	}
}

func (l *QUICListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.acceptCh:
		return conn, nil
	case <-l.done:
		return nil, errors.New("quic listener closed")
	}
}

func (l *QUICListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return l.listener.Close()
}

func (l *QUICListener) Addr() net.Addr {
	return l.listener.Addr()
}

