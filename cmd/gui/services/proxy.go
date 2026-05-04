package services

import (
	"fmt"
	"os/exec"
	"strings"
)

type ProxyMode string

const (
	ProxyModeNone      ProxyMode = "none"
	ProxyModeGlobal    ProxyMode = "global"
	ProxyModePAC       ProxyMode = "pac"
)

type ProxySettings struct {
	Mode           ProxyMode `json:"mode"`
	ProxyHost      string    `json:"proxyHost"`
	ProxyPort      int       `json:"proxyPort"`
	BypassLocal    bool      `json:"bypassLocal"`
	PACURL         string    `json:"pacUrl,omitempty"`
	Enabled        bool      `json:"enabled"`
}

type ProxyManager struct {
	settings ProxySettings
}

var proxyManager *ProxyManager

func GetProxyManager() *ProxyManager {
	if proxyManager == nil {
		proxyManager = &ProxyManager{
			settings: ProxySettings{
				Mode:        ProxyModeGlobal,
				ProxyHost:   "127.0.0.1",
				ProxyPort:   10808,
				BypassLocal: true,
				Enabled:     false,
			},
		}
	}
	return proxyManager
}

func (pm *ProxyManager) GetSettings() ProxySettings {
	return pm.settings
}

func (pm *ProxyManager) UpdateSettings(settings ProxySettings) {
	pm.settings = settings
	if settings.Enabled {
		pm.Enable()
	} else {
		pm.Disable()
	}
}

func (pm *ProxyManager) Enable() error {
	if pm.settings.Mode == ProxyModeNone {
		return nil
	}

	proxyAddr := fmt.Sprintf("%s:%d", pm.settings.ProxyHost, pm.settings.ProxyPort)

	if err := pm.setInternetOptions(proxyAddr); err != nil {
		return fmt.Errorf("启用系统代理失败: %w", err)
	}

	pm.settings.Enabled = true
	return nil
}

func (pm *ProxyManager) Disable() error {
	if err := pm.clearInternetOptions(); err != nil {
		return fmt.Errorf("关闭系统代理失败: %w", err)
	}
	pm.settings.Enabled = false
	return nil
}

func (pm *ProxyManager) Toggle() error {
	if pm.settings.Enabled {
		return pm.Disable()
	}
	return pm.Enable()
}

func (pm *ProxyManager) setInternetOptions(proxyAddr string) error {
	commands := [][]string{
		{"reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f"},
		{"reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f"},
		{"reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyOverride", "/t", "REG_SZ",
			"/d", fmt.Sprintf("<local>;%s", bypassLocalAddr()), "/f"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("注册表操作失败: %v", err)
		}
	}
	return nil
}

func (pm *ProxyManager) clearInternetOptions() error {
	commands := [][]string{
		{"reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("注册表操作失败: %v", err)
		}
	}
	return nil
}

func (pm *ProxyManager) IsEnabled() bool {
	return pm.settings.Enabled
}

func (pm *ProxyManager) IsSystemProxyEnabled() bool {
	cmd := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "0x1")
}

func bypassLocalAddr() string {
	return "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*"
}
