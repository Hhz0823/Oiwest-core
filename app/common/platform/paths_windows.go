//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

func appDataDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_DATA_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".oiwest")
	}
	exePath, _ := os.Executable()
	return filepath.Join(filepath.Dir(exePath), "data")
}

func configDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_CONFIG_DIR"); d != "" {
		return d
	}
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return filepath.Join(appData, "oiwest-core")
	}
	return filepath.Join(appDataDir(p), "config")
}

func cacheDir(p *PlatformInfo) string {
	appData := os.Getenv("LOCALAPPDATA")
	if appData != "" {
		return filepath.Join(appData, "oiwest-core", "cache")
	}
	return filepath.Join(appDataDir(p), "cache")
}

func logDir(p *PlatformInfo) string {
	return filepath.Join(appDataDir(p), "logs")
}
