//go:build darwin

package platform

import (
	"net"
	"syscall"
)

func SetReuseAddr(network string, address string, c syscall.RawConn) error {
	var soReuseAddrErr error
	err := c.Control(func(fd uintptr) {
		soReuseAddrErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
	if err != nil {
		return err
	}
	return soReuseAddrErr
}

func SetReusePort(network string, address string, c syscall.RawConn) error {
	var soReusePortErr error
	err := c.Control(func(fd uintptr) {
		soReusePortErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0x0200, 1)
	})
	if err != nil {
		return err
	}
	return soReusePortErr
}

func EnableTCPFastOpen(listener *net.TCPListener) error {
	return nil
}

func EnableTCPKeepAlive(conn *net.TCPConn, idle, interval, count int) error {
	return conn.SetKeepAlive(true)
}

func SetSocketBuffer(conn *net.TCPConn, readSize, writeSize int) error {
	conn.SetReadBuffer(readSize)
	conn.SetWriteBuffer(writeSize)
	return nil
}

func SetTCPNoDelay(conn *net.TCPConn, noDelay bool) error {
	return conn.SetNoDelay(noDelay)
}
