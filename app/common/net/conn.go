package net

import (
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Connection interface {
	io.ReadWriteCloser
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type connWrapper struct {
	net.Conn
}

func WrapConn(c net.Conn) Connection {
	return &connWrapper{Conn: c}
}

type PacketConnWrapper struct {
	net.PacketConn
}

func (p *PacketConnWrapper) ReadFrom(b []byte) (int, net.Addr, error) {
	return p.PacketConn.ReadFrom(b)
}

func (p *PacketConnWrapper) WriteTo(b []byte, addr net.Addr) (int, error) {
	return p.PacketConn.WriteTo(b, addr)
}

// StatCounter uses atomic operations for thread-safe byte counting
type StatCounter struct {
	ReadBytes    int64
	WrittenBytes int64
}

func (s *StatCounter) AddRead(n int64)  { atomic.AddInt64(&s.ReadBytes, n) }
func (s *StatCounter) AddWritten(n int64) { atomic.AddInt64(&s.WrittenBytes, n) }
func (s *StatCounter) GetRead() int64    { return atomic.LoadInt64(&s.ReadBytes) }
func (s *StatCounter) GetWritten() int64 { return atomic.LoadInt64(&s.WrittenBytes) }

type StatConnection struct {
	Connection
	Stat *StatCounter
}

func (c *StatConnection) Read(b []byte) (int, error) {
	n, err := c.Connection.Read(b)
	c.Stat.AddRead(int64(n))
	return n, err
}

func (c *StatConnection) Write(b []byte) (int, error) {
	n, err := c.Connection.Write(b)
	c.Stat.AddWritten(int64(n))
	return n, err
}
