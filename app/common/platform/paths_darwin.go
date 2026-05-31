//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

func appDataDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_DATA_DIR"); d != "" {
		return d
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Library", "Application Support", "oiwest-core")
	}
	return "/var/lib/oiwest-core"
}

func configDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_CONFIG_DIR"); d != "" {
		return d
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Library", "Preferences", "oiwest-core")
	}
	return "/etc/oiwest-core"
}

func cacheDir(p *PlatformInfo) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Library", "Caches", "oiwest-core")
	}
	return filepath.Join(appDataDir(p), "cache")
}

func logDir(p *PlatformInfo) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Library", "Logs", "oiwest-core")
	}
	return filepath.Join(appDataDir(p), "logs")
}
