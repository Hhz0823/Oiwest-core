//go:build android

package platform

import (
	"os"
	"path/filepath"
)

func appDataDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_DATA_DIR"); d != "" {
		return d
	}
	if internal := os.Getenv("ANDROID_DATA"); internal != "" {
		return filepath.Join(internal, "oiwest-core")
	}
	return "/data/local/tmp/oiwest-core"
}

func configDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_CONFIG_DIR"); d != "" {
		return d
	}
	return filepath.Join(appDataDir(p), "config")
}

func cacheDir(p *PlatformInfo) string {
	return filepath.Join(appDataDir(p), "cache")
}

func logDir(p *PlatformInfo) string {
	return filepath.Join(appDataDir(p), "logs")
}
