package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	grpcFrameSize    = 5
	grpcMaxDataSize  = 64 * 1024
)

var (
	ErrGRPCClosed = errors.New("gRPC: connection closed")
)

type GRPCConfig struct {
	ServiceName string `json:"serviceName"`
	MultiMode   bool   `json:"multiMode"`
	IdleTimeout time.Duration `json:"idleTimeout"`
}

type GRPCConn struct {
	conn     net.Conn
	config   *GRPCConfig
	readBuf  []byte
	readPos  int
	readLen  int
	mu       sync.Mutex
	closed   bool
}

func NewGRPCConn(ctx context.Context, addr string, port int, settings *StreamSettings) (*GRPCConn, error) {
	config := &GRPCConfig{
		ServiceName: "TunService",
		MultiMode:   false,
		IdleTimeout: 60 * time.Second,
	}
	if settings.GRPCSettings != nil {
		config.ServiceName = settings.GRPCSettings.ServiceName
		config.MultiMode = settings.GRPCSettings.MultiMode
	}

	remoteAddr := fmt.Sprintf("%s:%d", addr, port)
	tcpConn, err := net.DialTimeout("tcp", remoteAddr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial: %w", err)
	}

	conn := &GRPCConn{
		conn:    tcpConn,
		config:  config,
		readBuf: make([]byte, 64*1024),
	}

	return conn, nil
}

func (c *GRPCConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, ErrGRPCClosed
	}

	if c.readLen > 0 {
		n := copy(p, c.readBuf[c.readPos:c.readPos+c.readLen])
		c.readPos += n
		c.readLen -= n
		if c.readLen == 0 {
			c.readPos = 0
		}
		return n, nil
	}

	frameHeader := make([]byte, 5)
	if _, err := io.ReadFull(c.conn, frameHeader); err != nil {
		return 0, err
	}

	dataLen := int(binary.BigEndian.Uint32(frameHeader[0:4]))
	isCompressed := frameHeader[4] != 0

	if dataLen > len(c.readBuf) {
		c.readBuf = make([]byte, dataLen+64*1024)
	}

	if _, err := io.ReadFull(c.conn, c.readBuf[:dataLen]); err != nil {
		return 0, err
	}

	c.readPos = 0
	c.readLen = dataLen
	_ = isCompressed

	n := copy(p, c.readBuf[:dataLen])
	c.readPos = n
	c.readLen = dataLen - n
	if c.readLen == 0 {
		c.readPos = 0
	}

	return n, nil
}

func (c *GRPCConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, ErrGRPCClosed
	}

	offset := 0
	written := 0

	for offset < len(p) {
		chunkSize := len(p) - offset
		if chunkSize > grpcMaxDataSize {
			chunkSize = grpcMaxDataSize
		}

		frameHeader := make([]byte, 5)
		binary.BigEndian.PutUint32(frameHeader[0:4], uint32(chunkSize))
		frameHeader[4] = 0

		if _, err := c.conn.Write(frameHeader); err != nil {
			return written, err
		}
		if _, err := c.conn.Write(p[offset : offset+chunkSize]); err != nil {
			return written, err
		}

		offset += chunkSize
		written += chunkSize
	}

	return written, nil
}

func (c *GRPCConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return c.conn.Close()
}

func (c *GRPCConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *GRPCConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *GRPCConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *GRPCConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *GRPCConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type GRPCListener struct {
	listener net.Listener
	config   *GRPCConfig
	mu       sync.Mutex
}

func NewGRPCListener(addr string, settings *StreamSettings) (*GRPCListener, error) {
	config := &GRPCConfig{
		ServiceName: "TunService",
		MultiMode:   false,
		IdleTimeout: 60 * time.Second,
	}
	if settings.GRPCSettings != nil {
		config.ServiceName = settings.GRPCSettings.ServiceName
		config.MultiMode = settings.GRPCSettings.MultiMode
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &GRPCListener{
		listener: listener,
		config:   config,
	}, nil
}

func (l *GRPCListener) Accept() (net.Conn, error) {
	tcpConn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	conn := &GRPCConn{
		conn: tcpConn,
		config: l.config,
	}
	return conn, nil
}

func (l *GRPCListener) Close() error {
	return l.listener.Close()
}

func (l *GRPCListener) Addr() net.Addr {
	return l.listener.Addr()
}
