package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/sb-panel/dccp-kernel/config"
	"github.com/sb-panel/dccp-kernel/core"
)

var (
	configFile = flag.String("config", "config.json", "path to config file")
	version    = flag.Bool("version", false, "show version")
	testMode   = flag.Bool("test", false, "run in test mode")
	debugMode  = flag.Bool("debug", false, "enable debug logging")
)

const (
	AppVersion = "1.0.0"
	AppName    = "DCCP-Kernel"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("DCCP Protocol Support: RFC 4340\n")
		fmt.Printf("Compatibility: v2ray-core, Xray-core, sing-box\n")
		os.Exit(0)
	}

	if *debugMode {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}

	log.Printf("============================================")
	log.Printf("  %s v%s - DCCP Transport Protocol Kernel", AppName, AppVersion)
	log.Printf("  RFC 4340 | v2ray-core | Xray-core | sing-box")
	log.Printf("============================================")

	var cfg *config.Config

	if *testMode {
		log.Println("[Test Mode] Using default configuration")
		cfg = config.DefaultConfig()
	} else {
		data, err := os.ReadFile(*configFile)
		if err != nil {
			log.Printf("Config file not found: %s, using defaults", *configFile)
			cfg = config.DefaultConfig()

			defaultJSON, _ := cfg.ToJSON()
			os.WriteFile(*configFile, defaultJSON, 0644)
			log.Printf("Default config saved to: %s", *configFile)
		} else {
			cfg, err = config.ParseConfig(data)
			if err != nil {
				log.Fatalf("Failed to parse config: %v", err)
			}
			log.Printf("Config loaded from: %s", *configFile)
		}
	}

	kernel := core.NewCore(cfg)

	if err := kernel.Start(); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}

	log.Printf("[%s] Kernel is running (PID: %d)", AppName, os.Getpid())
	log.Printf("[%s] Transports: DCCP, TCP, WebSocket, QUIC, gRPC", AppName)
	log.Printf("[%s] Protocols: VMess, VLESS, Trojan, Shadowsocks, SOCKS, HTTP", AppName)
	log.Printf("[%s] Stealth: XTLS, Vision, Reality, Random Padding", AppName)

	kernel.WaitForSignal()

	log.Printf("[%s] Goodbye!", AppName)
}
