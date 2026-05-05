//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hhz0823/oiwest-core/common/platform"
	"github.com/Hhz0823/oiwest-core/config"
	"github.com/Hhz0823/oiwest-core/core"
)

func main() {
	p := platform.Get()
	fmt.Printf("Oiwest Core Daemon v%s\n", core.Version)
	fmt.Printf("Platform: %s/%s | Distro: %s\n", p.OS, p.Arch, p.Distro)
	fmt.Printf("Headless: %v | Container: %v\n", p.IsHeadless, p.IsContainer)
	fmt.Printf("Data Dir: %s\n", p.AppDir)
	fmt.Printf("Config Dir: %s\n", p.ConfigDir)
	fmt.Printf("PID: %d\n", os.Getpid())

	cfg, err := loadConfig(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	kernel := core.NewCore(cfg)
	if err := kernel.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start kernel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[%s] Daemon started successfully\n", core.CoreName)
	fmt.Printf("[%s] Transports: TCP, mKCP, WebSocket, HTTP/2, QUIC, gRPC, XHTTP, DCCP\n", core.CoreName)
	fmt.Printf("[%s] Protocols: VMess, VLESS, Trojan, Shadowsocks, SOCKS, HTTP, Dokodemo, Loopback, DNS\n", core.CoreName)
	fmt.Printf("[%s] Features: WorkerPool, BBR, DualStack, TLS\n", core.CoreName)

	writePIDFile(p)
	defer removePIDFile(p)

	kernel.WaitForSignal()
}

func loadConfig(p *platform.PlatformInfo) (*config.Config, error) {
	configPath := filepath.Join(p.ConfigDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.MkdirAll(p.ConfigDir, 0700)
		cfg := config.DefaultConfig()
		data, _ := cfg.ToJSON()
		os.WriteFile(configPath, data, 0644)
		return cfg, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return config.ParseConfig(data)
}

func writePIDFile(p *platform.PlatformInfo) {
	os.MkdirAll(p.AppDir, 0700)
	pidPath := filepath.Join(p.AppDir, "oiwest-core.pid")
	pid := fmt.Sprintf("%d", os.Getpid())
	os.WriteFile(pidPath, []byte(pid), 0644)
}

func removePIDFile(p *platform.PlatformInfo) {
	pidPath := filepath.Join(p.AppDir, "oiwest-core.pid")
	os.Remove(pidPath)
}
