package net

import (
	"io"
	"net"
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

type StatCounter struct {
	ReadBytes    int64
	WrittenBytes int64
}

type StatConnection struct {
	Connection
	Stat *StatCounter
}

func (c *StatConnection) Read(b []byte) (int, error) {
	n, err := c.Connection.Read(b)
	c.Stat.ReadBytes += int64(n)
	return n, err
}

func (c *StatConnection) Write(b []byte) (int, error) {
	n, err := c.Connection.Write(b)
	c.Stat.WrittenBytes += int64(n)
	return n, err
}
