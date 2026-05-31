//go:build !windows && !android

package platform

import (
	"os"
	"os/signal"
	"syscall"
)

func NotifySignal(callback func(os.Signal)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-sigCh
		if callback != nil {
			callback(sig)
		}
	}()
}

func WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return <-sigCh
}

func SendSignal(pid int, sig os.Signal) error {
	return syscall.Kill(pid, syscall.Signal(sig.(syscall.Signal)))
}

func GracefulShutdown() syscall.Signal {
	return syscall.SIGTERM
}
