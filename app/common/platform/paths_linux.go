//go:build linux && !android

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
		if p.Distro == DistroOpenWrt {
			return filepath.Join("/etc", "oiwest-core")
		}
		return filepath.Join(home, ".local", "share", "oiwest-core")
	}
	return "/var/lib/oiwest-core"
}

func configDir(p *PlatformInfo) string {
	if d := os.Getenv("OIWEST_CONFIG_DIR"); d != "" {
		return d
	}
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "oiwest-core")
	}
	if p.Distro == DistroOpenWrt {
		return filepath.Join("/etc", "oiwest-core")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config", "oiwest-core")
	}
	return "/etc/oiwest-core"
}

func cacheDir(p *PlatformInfo) string {
	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache != "" {
		return filepath.Join(xdgCache, "oiwest-core")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p.Distro == DistroOpenWrt {
			return filepath.Join(appDataDir(p), "cache")
		}
		return filepath.Join(home, ".cache", "oiwest-core")
	}
	return filepath.Join(appDataDir(p), "cache")
}

func logDir(p *PlatformInfo) string {
	if p.Distro == DistroOpenWrt {
		return "/var/log"
	}
	return filepath.Join(appDataDir(p), "logs")
}
