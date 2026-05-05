//go:build linux && !android

package platform

import (
	"os"
	"strings"
)

func isAndroid() bool {
	return false
}

func hasGUI() bool {
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("XDG_SESSION_TYPE") == "x11" {
		return true
	}
	return false
}

func detectDistro() DistroType {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		data, err = os.ReadFile("/usr/lib/os-release")
		if err != nil {
			return DistroUnknown
		}
	}

	content := strings.ToLower(string(data))
	if strings.Contains(content, "openwrt") {
		return DistroOpenWrt
	}
	if strings.Contains(content, "ubuntu") {
		return DistroUbuntu
	}
	if strings.Contains(content, "debian") {
		return DistroDebian
	}
	if strings.Contains(content, "fedora") {
		return DistroFedora
	}
	if strings.Contains(content, "alpine") {
		return DistroAlpine
	}
	if strings.Contains(content, "arch") {
		return DistroArch
	}
	if strings.Contains(content, "centos") {
		return DistroCentOS
	}
	return DistroUnknown
}

func detectContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil && (strings.Contains(string(data), "docker") ||
		strings.Contains(string(data), "lxc") ||
		strings.Contains(string(data), "kubepods")) {
		return true
	}
	return false
}
