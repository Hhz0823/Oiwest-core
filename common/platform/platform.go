package platform

import (
	"runtime"
)

type OSType string

const (
	Windows OSType = "windows"
	Linux   OSType = "linux"
	Darwin  OSType = "darwin"
	Android OSType = "android"
	Unknown OSType = "unknown"
)

type ArchType string

const (
	AMD64 ArchType = "amd64"
	ARM64 ArchType = "arm64"
	ARMv7 ArchType = "arm"
	X86   ArchType = "386"
)

type DistroType string

const (
	DistroUnknown  DistroType = "unknown"
	DistroUbuntu   DistroType = "ubuntu"
	DistroDebian   DistroType = "debian"
	DistroFedora   DistroType = "fedora"
	DistroOpenWrt  DistroType = "openwrt"
	DistroAndroid  DistroType = "android"
	DistroAlpine   DistroType = "alpine"
	DistroArch     DistroType = "arch"
	DistroCentOS   DistroType = "centos"
)

type PlatformInfo struct {
	OS       OSType
	Arch     ArchType
	Distro   DistroType
	IsHeadless  bool
	IsContainer bool
	AppDir      string
	ConfigDir   string
	CacheDir    string
	LogDir      string
}

var info *PlatformInfo

func init() {
	info = detectPlatform()
}

func Get() *PlatformInfo {
	return info
}

func detectPlatform() *PlatformInfo {
	p := &PlatformInfo{
		OS:         detectOS(),
		Arch:       detectArch(),
		Distro:     detectDistro(),
		IsHeadless:  !hasGUI(),
		IsContainer: detectContainer(),
	}
	p.AppDir = appDataDir(p)
	p.ConfigDir = configDir(p)
	p.CacheDir = cacheDir(p)
	p.LogDir = logDir(p)
	return p
}

func detectOS() OSType {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "linux":
		if isAndroid() {
			return Android
		}
		return Linux
	case "darwin":
		return Darwin
	default:
		return Unknown
	}
}

func detectArch() ArchType {
	switch runtime.GOARCH {
	case "amd64":
		return AMD64
	case "arm64":
		return ARM64
	case "arm":
		return ARMv7
	case "386":
		return X86
	default:
		return ArchType(runtime.GOARCH)
	}
}

func (p *PlatformInfo) IsWindows() bool  { return p.OS == Windows }
func (p *PlatformInfo) IsLinux() bool    { return p.OS == Linux }
func (p *PlatformInfo) IsDarwin() bool   { return p.OS == Darwin }
func (p *PlatformInfo) IsAndroid() bool  { return p.OS == Android }
func (p *PlatformInfo) IsAMD64() bool    { return p.Arch == AMD64 }
func (p *PlatformInfo) IsARM64() bool    { return p.Arch == ARM64 }
func (p *PlatformInfo) IsARM() bool      { return p.Arch == ARMv7 }
func (p *PlatformInfo) IsOpenWrt() bool  { return p.Distro == DistroOpenWrt }

func (p *PlatformInfo) ExeName() string {
	name := "oiwest-core"
	if p.OS == Windows {
		return name + ".exe"
	}
	return name
}

func (p *PlatformInfo) IsSupported() bool {
	supportedOS := map[OSType]bool{
		Windows: true, Linux: true, Android: true, Darwin: true,
	}
	supportedArch := map[ArchType]bool{
		AMD64: true, ARM64: true, ARMv7: true, X86: true,
	}
	return supportedOS[p.OS] && supportedArch[p.Arch]
}
