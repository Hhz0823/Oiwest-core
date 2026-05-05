package services

import (
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type DeviceInfo struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	GoVersion   string `json:"goVersion"`
	CPUCores    int    `json:"cpuCores"`
	Hostname    string `json:"hostname"`
	TotalMemory uint64 `json:"totalMemory"`
	AppVersion  string `json:"appVersion"`
}

type SystemUsage struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsed    uint64  `json:"memoryUsed"`
	MemoryTotal   uint64  `json:"memoryTotal"`
	MemoryPercent float64 `json:"memoryPercent"`
	PublicIP      string  `json:"publicIp"`
	PublicIPv6    string  `json:"publicIpv6"`
}

type cpuState struct {
	lastIdle   syscall.Filetime
	lastKernel syscall.Filetime
	lastUser   syscall.Filetime
}

type memStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

type SysInfoManager struct {
	cpuState    *cpuState
	publicIP    string
	publicIPv6  string
	ipCheckTime time.Time
}

var sysInfoMgr *SysInfoManager

func GetSysInfoManager() *SysInfoManager {
	if sysInfoMgr == nil {
		sysInfoMgr = &SysInfoManager{cpuState: &cpuState{}}
	}
	return sysInfoMgr
}

func (sm *SysInfoManager) GetDeviceInfo() DeviceInfo {
	hostname, _ := os.Hostname()
	var totalMem uint64
	var ms memStatusEx
	ms.length = uint32(unsafe.Sizeof(ms))
	ret, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&ms)))
	if ret != 0 {
		totalMem = ms.totalPhys
	}

	return DeviceInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
		CPUCores:    runtime.NumCPU(),
		Hostname:    hostname,
		TotalMemory: totalMem,
		AppVersion:  "1.1.0",
	}
}

func (sm *SysInfoManager) GetSystemUsage() SystemUsage {
	usage := SystemUsage{}

	var ms memStatusEx
	ms.length = uint32(unsafe.Sizeof(ms))
	ret, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&ms)))
	if ret != 0 {
		usage.MemoryTotal = ms.totalPhys
		usage.MemoryUsed = ms.totalPhys - ms.availPhys
		if ms.totalPhys > 0 {
			usage.MemoryPercent = float64(usage.MemoryUsed) / float64(ms.totalPhys) * 100
		}
	}

	usage.CPUPercent = sm.cpuPercent()

	if time.Since(sm.ipCheckTime) > 60*time.Second || sm.publicIP == "" {
		sm.publicIP = sm.fetchPublicIP()
		sm.publicIPv6 = sm.fetchPublicIPv6()
		sm.ipCheckTime = time.Now()
	}
	usage.PublicIP = sm.publicIP
	usage.PublicIPv6 = sm.publicIPv6

	return usage
}

func (sm *SysInfoManager) cpuPercent() float64 {
	var idle, kernel, user syscall.Filetime

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetSystemTimes")
	r, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return 0
	}

	if sm.cpuState.lastIdle.Nanoseconds() == 0 {
		sm.cpuState.lastIdle = idle
		sm.cpuState.lastKernel = kernel
		sm.cpuState.lastUser = user
		return 0
	}

	idleDiff := ftDiff(sm.cpuState.lastIdle, idle)
	kernelDiff := ftDiff(sm.cpuState.lastKernel, kernel)
	userDiff := ftDiff(sm.cpuState.lastUser, user)
	totalDiff := kernelDiff + userDiff

	sm.cpuState.lastIdle = idle
	sm.cpuState.lastKernel = kernel
	sm.cpuState.lastUser = user

	if totalDiff == 0 {
		return 0
	}
	return float64(totalDiff-idleDiff) / float64(totalDiff) * 100
}

func ftDiff(old, new syscall.Filetime) int64 {
	o := uint64(old.HighDateTime)<<32 | uint64(old.LowDateTime)
	n := uint64(new.HighDateTime)<<32 | uint64(new.LowDateTime)
	return int64(n - o)
}

func (sm *SysInfoManager) GetPublicIP() string {
	if time.Since(sm.ipCheckTime) > 60*time.Second || sm.publicIP == "" {
		sm.publicIP = sm.fetchPublicIP()
		sm.publicIPv6 = sm.fetchPublicIPv6()
		sm.ipCheckTime = time.Now()
	}
	return sm.publicIP
}

func (sm *SysInfoManager) RefreshPublicIP() string {
	sm.publicIP = sm.fetchPublicIP()
	sm.publicIPv6 = sm.fetchPublicIPv6()
	sm.ipCheckTime = time.Now()
	return sm.publicIP
}

func (sm *SysInfoManager) fetchPublicIP() string {
	for _, url := range []string{"https://api4.ipify.org", "https://ipv4.icanhazip.com"} {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if ip != "" && !strings.Contains(ip, ":") && !strings.Contains(ip, "<") {
			return ip
		}
	}
	return "N/A"
}

func (sm *SysInfoManager) fetchPublicIPv6() string {
	for _, url := range []string{"https://api6.ipify.org", "https://ipv6.icanhazip.com"} {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if ip != "" && !strings.Contains(ip, "<") && strings.Contains(ip, ":") {
			return ip
		}
	}
	return ""
}
