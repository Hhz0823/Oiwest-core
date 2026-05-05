//go:build windows

package platform

import (
	"os"
	"os/signal"
)

func NotifySignal(callback func(os.Signal)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		sig := <-sigCh
		if callback != nil {
			callback(sig)
		}
	}()
}

func WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	return <-sigCh
}

func SendSignal(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

func GracefulShutdown() os.Signal {
	return os.Interrupt
}
