//go:build windows

package platform

import (
	"net"
	"syscall"
)

var (
	kernel32 = syscall.NewLazyDLL("ws2_32.dll")
	setsockopt = kernel32.NewProc("setsockopt")
)

const (
	SOL_SOCKET   = 0xFFFF
	SO_REUSEADDR = 0x0004
	IPPROTO_TCP  = 6
	TCP_NODELAY  = 0x0001
	SO_RCVBUF    = 0x1002
	SO_SNDBUF    = 0x1001
	SO_KEEPALIVE = 0x0008
)

func SetReuseAddr(network string, address string, c syscall.RawConn) error {
	return nil
}

func SetReusePort(network string, address string, c syscall.RawConn) error {
	return nil
}

func EnableTCPFastOpen(listener *net.TCPListener) error {
	return nil
}

func EnableTCPKeepAlive(conn *net.TCPConn, idle, interval, count int) error {
	conn.SetKeepAlive(true)
	return nil
}

func SetSocketBuffer(conn *net.TCPConn, readSize, writeSize int) error {
	conn.SetReadBuffer(readSize)
	conn.SetWriteBuffer(writeSize)
	return nil
}

func SetTCPNoDelay(conn *net.TCPConn, noDelay bool) error {
	conn.SetNoDelay(noDelay)
	return nil
}
