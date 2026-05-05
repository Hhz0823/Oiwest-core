//go:build !windows && !darwin

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
		soReusePortErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0x0F, 1)
	})
	if err != nil {
		return err
	}
	return soReusePortErr
}

func EnableTCPFastOpen(listener *net.TCPListener) error {
	f, err := listener.File()
	if err != nil {
		return err
	}
	defer f.Close()
	return syscall.SetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, 23, 256)
}

func EnableTCPKeepAlive(conn *net.TCPConn, idle, interval, count int) error {
	f, err := conn.File()
	if err != nil {
		return err
	}
	defer f.Close()
	fd := int(f.Fd())
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, idle)
	syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, interval)
	syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, count)
	return nil
}

func SetSocketBuffer(conn *net.TCPConn, readSize, writeSize int) error {
	f, err := conn.File()
	if err != nil {
		return err
	}
	defer f.Close()
	fd := int(f.Fd())
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, readSize)
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, writeSize)
	return nil
}

func SetTCPNoDelay(conn *net.TCPConn, noDelay bool) error {
	var v int
	if noDelay {
		v = 1
	}
	f, err := conn.File()
	if err != nil {
		return err
	}
	defer f.Close()
	return syscall.SetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, v)
}
