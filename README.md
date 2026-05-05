<p align="center">
  <img src="frontend/public/logo.svg" alt="Oiwest Core" width="160" />
</p>

<h1 align="center">Oiwest Core</h1>
<p align="center">
  <strong>高性能 · 多平台 · 模块化 代理协议内核</strong><br>
  基于 DCCP (RFC 4340) 传输协议 · 兼容 v2ray-core · Xray-core · sing-box 生态
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-2.0.1-blue" />
  <img src="https://img.shields.io/badge/go-1.22.0%2B-00ADD8" />
  <img src="https://img.shields.io/badge/license-MIT-green" />
  <img src="https://img.shields.io/badge/platforms-18%2B-orange" />
</p>

---

## 项目概述

Oiwest Core 是一个高性能、模块化的代理协议内核。最初以 **DCCP（Datagram Congestion Control Protocol，数据报拥塞控制协议，RFC 4340）** 为核心传输协议，现已扩展为支持 **11 种传输协议**、**18 种平台架构**的通用代理引擎。

项目包含两个主要组件：

- **核心引擎** — 两种运行模式：`oiwest-core` CLI 命令行模式 + `oiwest-daemon` 无头守护进程模式
- **图形界面 GUI** — 基于 Wails v2 + React 的桌面程序，支持节点管理、流量监控、系统代理配置

---

## 🚀 特性

### 代理协议

| 类型 | 协议 |
|---|---|
| **入站** | VMess · VLESS · Trojan · Shadowsocks · SOCKS5 · HTTP · Dokodemo-door · Loopback · DCCP |
| **出站** | Freedom · Blackhole · Direct · DNS · VMess · VLESS · Trojan · Shadowsocks · SOCKS · DCCP |

### 底层传输

| 协议 | 说明 |
|---|---|
| **TCP** | 标准 TCP，支持 HTTP 伪装头 |
| **mKCP** | 基于 KCP 的可靠 UDP 传输（流模式/数据报模式） |
| **WebSocket** | WebSocket / WebSocket v2，支持 TLS |
| **HTTP/2** | HTTP/2 多路复用传输 |
| **QUIC** | 基于 quic-go 的低延迟传输 |
| **gRPC** | gRPC 流式传输，支持多服务模式 |
| **XHTTP** | 自定义帧协议 + 多路复用 HTTP 流 |
| **DCCP** | 数据报拥塞控制协议 (RFC 4340)，支持 CCID2/CCID3/CCID4 |

### 混淆与隐匿

| 模块 | 能力 |
|---|---|
| **XTLS** | XTLS Vision/Reality 流控混淆 + uTLS 指纹模拟 |
| **Random Padding** | 随机填充混淆，规避流量特征检测 |
| **XOR Obfuscation** | 基于异或的数据混淆 |
| **UDP Obfuscation** | UDP 数据包混淆 |
| **DTLS Obfuscation** | DTLS 伪装 |
| **WireGuard Obfuscation** | WireGuard 协议伪装 |
| **Fingerprint** | Chrome / Firefox / Safari / iOS / Android / Edge / 360 / QQ / Random |

### 高级特性

| 模块 | 能力 |
|---|---|
| **BBR 拥塞控制** | BBR / BBRv2 / BBRv3 / BBRPlus / BBR-ECN / BBR-Adaptive / BBR-ProbeRTT |
| **双栈网络** | IPv4/IPv6 双栈 + 多线路并行 + latency/random/roundrobin/multiline 策略 |
| **WorkerPool** | 多线程任务池，可配置 worker 数量/队列/重试，动态扩缩容 |
| **TLS 证书** | RSA/ECDSA/Ed25519 自动生成，CA 签发，到期自动续期 |
| **路由引擎** | 域名（全匹配/关键词/正则）/ IP（CIDR/精确）/ 端口 / 协议 / 入站标签 + 负载均衡 |
| **多路复用** | 帧级多路复用 MuxSession，128+ 并发流，KeepAlive 保活 |
| **加密算法** | AES-128-GCM · AES-256-GCM · ChaCha20-Poly1305 · XChaCha20-Poly1305 |

---

## 📦 预编译包

所有包位于 `build/zip/`，每个 ZIP 包含 `oiwest-core` (CLI) 和 `oiwest-daemon` (守护进程)。

### CLI + Daemon（无头模式，14 个平台）

| 平台 | 架构 | 适用系统 |
|---|---|---|
| `windows-amd64` | x64 | Windows 10/11 |
| `windows-arm64` | ARM64 | Windows on ARM (Surface Pro X 等) |
| `windows-386` | x86 | Windows 32 位 |
| `linux-amd64` | x64 | Ubuntu / Debian / Fedora / CentOS / Arch |
| `linux-arm64` | ARM64 | 树莓派 4/5 · ARM 服务器 |
| `linux-armv7` | ARMv7 | 树莓派 2/3 · ARM 设备 |
| `linux-386` | x86 | 老旧 Linux 设备 |
| `linux-mips` | MIPS BE | OpenWrt ar71xx/ath79 |
| `linux-mipsle` | MIPS LE | OpenWrt ramips/mt7621 (新路由3/小米路由等) |
| `linux-mips64` | MIPS64 | 高端 MIPS 路由器 |
| `linux-mips64le` | MIPS64LE | 高端 MIPS 路由器 (LE) |
| `darwin-amd64` | x64 | macOS Intel |
| `darwin-arm64` | ARM64 | macOS Apple Silicon (M1/M2/M3/M4) |
| `android-arm64` | ARM64 | Android 设备 (Termux) |

### GUI 桌面程序（4 个包）

| 包名 | 架构 | 说明 |
|---|---|---|
| `gui-windows-amd64` | x64 | Windows 图形界面，带节点管理/流量监控/系统代理 |
| `gui-windows-arm64` | ARM64 | Windows on ARM 图形界面 |
| `gui-darwin-amd64` | x64 | macOS Intel 图形界面 |
| `gui-darwin-arm64` | ARM64 | macOS Apple Silicon 图形界面 |

---

## 🏗️ 从源码构建

### 前置依赖

- **Go** 1.22.0+
- **Node.js** 18+（仅 GUI 构建需要）
- **CGO 交叉编译器**（如需跨平台编译 GUI）

### 快速开始

```bash
# 克隆项目
git clone https://github.com/Hhz0823/oiwest-core.git
cd oiwest-core

# 安装依赖
go mod download

# 编译当前平台 CLI
make build
# 或
go build -o build/oiwest-core ./cmd/oiwest-core

# 编译当前平台守护进程
make build-daemon
# 或
go build -o build/oiwest-daemon ./cmd/oiwest-daemon
```

### 构建 GUI 程序

GUI 基于 [Wails v2](https://wails.io/) 框架，将前端（React + TypeScript）和后端（Go）编译为单个可执行文件。

```bash
# 安装 Wails CLI（首次）
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 构建 GUI 程序（Windows）
make build-gui-windows

# 构建 GUI 程序（macOS - 需在 macOS 上）
make build-gui-darwin

# 构建 GUI 程序（Linux - 需在 Linux 上）
make build-gui-linux

# 开发模式（前端热重载）
wails dev
```

### 构建全部平台

```bash
# 编译所有 CLI + Daemon（14 个平台）
make build-all

# 编译所有 OpenWrt 架构
make build-openwrt-all

# 编译所有 GUI 程序
make build-gui-all

# 查看完整目标列表
make info
```

### 构建特定目标

```bash
# Windows
make build-windows-amd64
make build-windows-arm64

# Linux
make build-linux-amd64
make build-linux-arm64
make build-linux-arm       # ARMv7

# OpenWrt
make build-openwrt-mips    # ar71xx/ath79
make build-openwrt-mipsel  # mt7621/ramips
make build-openwrt-arm     # ipq40xx/bcm27xx
make build-openwrt-arm64   # rockchip/ipq807x

# macOS
make build-darwin-amd64
make build-darwin-arm64

# Android
make build-android-arm64
```

---

## 🖥️ 运行说明

### CLI 模式（全平台）

```bash
./oiwest-core -config config.json   # 指定配置文件运行
./oiwest-core -test                 # 使用内置默认配置测试
./oiwest-core -debug                # 调试模式启动
./oiwest-core -version              # 查看版本信息
```

### Daemon 守护进程模式（全平台无头）

```bash
./oiwest-daemon
```

守护进程会自动检测当前平台并选择适配的配置目录：

| 平台 | 配置目录 | 数据目录 |
|---|---|---|
| Windows | `%APPDATA%/oiwest-core` | `%USERPROFILE%/.oiwest` |
| Linux | `~/.config/oiwest-core` | `~/.local/share/oiwest-core` |
| macOS | `~/Library/Preferences/oiwest-core` | `~/Library/Application Support/oiwest-core` |
| Android | `/data/local/tmp/oiwest-core/config` | `/data/local/tmp/oiwest-core` |
| OpenWrt | `/etc/oiwest-core` | `/etc/oiwest-core` |

守护进程会自动生成 PID 文件到数据目录，支持优雅关闭。

### GUI 桌面模式（Windows / macOS）

双击运行 `oiwest-core-gui.exe`（Windows）或 `oiwest-core-gui`（macOS）即可启动图形界面。

---

## 🎨 GUI 使用说明

### 仪表盘
- 实时显示上行/下行速度和总流量
- 活跃连接数和核心运行时长
- 一键启动/停止/重启核心引擎
- 系统代理开关控制
- 当前选中节点信息展示

### 服务器管理
- 节点列表展示，支持分组过滤和关键词搜索
- **添加节点**：填写协议、地址、端口、UUID/密码等
- **导入节点**：粘贴 vmess:// 分享链接自动解析
- 编辑/删除节点
- 单击设为当前使用节点

### 传输配置

节点支持以下传输层配置：

- **TCP** — 标准 TCP
- **WebSocket** — 需配置 Path 和 Host
- **gRPC** — 需配置 ServiceName
- **QUIC** — 基于 QUIC 传输
- **mKCP** — KCP 可靠 UDP 传输
- **XHTTP** — 自定义 HTTP 流传输
- **DCCP** — DCCP 协议（核心特色）
- **TLS** — 支持 SNI / Fingerprint / AllowInsecure
- **Reality** — 支持 PublicKey / ShortID / SpiderX

### 日志查看
- 实时显示核心运行日志
- 彩色标记：绿色（信息）、黄色（警告）、红色（错误）
- 自动滚动到最新日志
- 一键清空

### 系统代理设置
- 全局模式：所有流量通过代理
- PAC 模式：使用 PAC 脚本自动判断
- 直连模式：关闭系统代理
- 自定义代理地址和端口
- 可配置是否绕过局域网地址

---

## ⚙️ 配置示例

### 完整配置（全功能展示）

```json
{
  "log": { "loglevel": "warning" },
  "workerPool": {
    "numWorkers": 16,
    "queueSize": 2000,
    "maxRetries": 3
  },
  "bbr": {
    "enabled": true,
    "algorithm": "bbrv2",
    "settings": { "mss": 1460 }
  },
  "dualStack": {
    "enabled": true,
    "preference": "dual",
    "multiLine": false,
    "failover": true,
    "strategy": "latency"
  },
  "certificate": {
    "enabled": true,
    "autoGenerate": true,
    "config": {
      "commonName": "example.com",
      "keyType": "ecdsa",
      "keySize": 256,
      "validFor": 365
    }
  },
  "inbounds": [
    {
      "tag": "socks-in",
      "port": 10808,
      "listen": "0.0.0.0",
      "protocol": "socks"
    },
    {
      "tag": "http-in",
      "port": 10809,
      "listen": "0.0.0.0",
      "protocol": "http"
    },
    {
      "tag": "vless-in",
      "port": 443,
      "listen": "0.0.0.0",
      "protocol": "vless",
      "streamSettings": {
        "network": "ws",
        "security": "tls",
        "wsSettings": { "path": "/ws" }
      }
    }
  ],
  "outbounds": [
    {
      "tag": "proxy",
      "protocol": "vless",
      "streamSettings": {
        "network": "ws",
        "security": "tls",
        "wsSettings": { "path": "/ws", "host": "example.com" }
      }
    },
    { "tag": "direct", "protocol": "freedom" },
    { "tag": "block", "protocol": "blackhole" }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      { "type": "field", "domain": ["geosite:cn"], "outboundTag": "direct" },
      { "type": "field", "domain": ["geosite:category-ads-all"], "outboundTag": "block" }
    ]
  }
}
```

### DCCP 默认配置

```json
{
  "inbounds": [{
    "tag": "dccp-in",
    "port": 33445,
    "listen": "0.0.0.0",
    "protocol": "dccp",
    "streamSettings": {
      "network": "dccp",
      "security": "none",
      "dccpSettings": {
        "ccid": 4,
        "serviceCode": "V2RY",
        "maxPacketSize": 1500,
        "handshakeTimeout": 15000000000,
        "maxRetries": 3,
        "enableObfuscation": true,
        "obfuscationType": "random"
      }
    }
  }],
  "outbounds": [{ "tag": "direct", "protocol": "freedom" }]
}
```

### GUI 配置

GUI 的节点数据保存在 `~/.oiwest/servers.json`，核心配置文件保存在 `~/.oiwest/config.json`，由 GUI 程序自动管理。

---

## 🌐 DCCP 协议说明

**DCCP（Datagram Congestion Control Protocol）** 是 IETF RFC 4340 定义的传输层协议，它提供了：

- **不可靠数据传输** — 类似 UDP，无重传机制
- **拥塞控制** — 类似 TCP，避免网络拥塞崩溃
- **连接建立与拆除** — 具有三次握手和四次挥手
- **多种拥塞控制算法** — CCID2（TCP-like）、CCID3（TFRC）、CCID4（TFRC-SP）

本项目将 DCCP 引入代理通信领域，利用其在高延迟/高丢包网络环境下的传输性能优势。

---

## 📁 项目结构

```
oiwest-core/
├── cmd/
│   ├── oiwest-core/          # CLI 核心入口
│   ├── oiwest-daemon/        # 无头守护进程入口
│   └── gui/services/         # GUI 后端服务层
│       ├── node.go            # 节点管理器（CRUD / 分组 / 持久化）
│       ├── core.go            # 核心进程管理（启动/停止/重启/日志）
│       ├── proxy.go           # 系统代理（注册表配置）
│       ├── stats.go           # 流量统计（速度计算 / 格式化）
│       ├── config.go          # 配置生成器（多协议配置构建）
│       ├── latency.go         # 延迟测试
│       ├── network.go         # 网络配置管理
│       ├── sysinfo.go         # 系统信息（CPU/内存/公网IP）
│       └── tlscert.go         # TLS 证书管理
├── core/                     # 核心引擎
│   ├── core.go                # Core 生命周期 + 协议注册
│   └── worker.go              # WorkerPool / 任务调度器 / 并行执行器
├── proxy/                    # 代理协议实现
│   ├── proxy.go               # Inbound / Outbound 接口定义
│   ├── registry.go            # 协议注册框架 + Dokodemo/Loopback/DNS
│   ├── vmess/                 # VMess 协议 (AEAD 加密)
│   ├── vless/                 # VLESS 协议 (XTLS Vision)
│   ├── trojan/                # Trojan 协议
│   └── shadowsocks/           # Shadowsocks 协议 (AES/ChaCha20)
├── transport/                # 底层传输协议
│   ├── config.go              # StreamSettings 配置定义
│   ├── transport.go           # DCCP 传输实现
│   ├── mkcp.go                # mKCP 传输 (KCP 状态机)
│   ├── websocket.go           # WebSocket 传输
│   ├── quic.go                # QUIC 传输
│   ├── grpc.go                # gRPC 传输
│   ├── xhttp.go               # XHTTP 传输
│   ├── dccp/                  # DCCP 协议实现
│   │   ├── congestion.go      # CCID2/CCID3/CCID4 拥塞控制
│   │   ├── dccp.go            # DCCP 协议核心
│   │   ├── handshake.go       # 三次握手
│   │   └── packet.go          # 数据包编解码
│   └── bbr/                   # BBR 拥塞控制算法族
├── common/
│   ├── buf/                   # 高效内存缓冲区 (sync.Pool)
│   ├── crypto/                # AES-GCM / ChaCha20-Poly1305 / HKDF
│   ├── net/                   # 地址抽象 + 双栈拨号器 + 多线路管理
│   ├── tls/                   # 证书生成管理器
│   └── platform/              # 平台抽象层
│       ├── platform.go         # 检测核心 (OS/Arch/Distro)
│       ├── detect_*.go         # 平台检测 (Windows/Linux/Android/macOS)
│       ├── paths_*.go          # 路径策略 (XDG/APPDATA/Library)
│       ├── signal_*.go         # 信号处理 (SIGINT/SIGTERM)
│       └── net_*.go            # 套接字选项 (SO_REUSEADDR/FASTOPEN)
├── features/
│   ├── multiplex/             # 帧级多路复用 (MuxSession)
│   └── stealth/               # XTLS · Vision · Reality · 混淆
├── config/config.go           # 配置解析与验证
├── router/router.go           # 路由引擎 (域名/IP/端口/协议)
├── frontend/                  # Wails GUI 前端 (React + Vite + Tailwind)
│   ├── src/
│   │   ├── components/        # 组件
│   │   ├── pages/             # 页面 (Dashboard/ServerList/Settings/Logs)
│   │   ├── stores/            # Zustand 状态管理
│   │   └── types/             # TypeScript 类型定义
│   └── package.json           # 前端依赖
├── app.go                     # Wails 主应用 (Go ⇄ 前端桥接)
├── wails.json                 # Wails 构建配置
├── go.mod / go.sum            # Go 模块定义
├── Makefile                   # 跨平台交叉编译脚本
├── build/                     # 编译输出
│   └── zip/                   # ZIP 分发包
└── README.md
```

---

## 🔧 Makefile 目标速查

| 目标 | 说明 |
|---|---|
| `make build` | 当前平台 CLI 编译 |
| `make build-daemon` | 当前平台守护进程编译 |
| `make build-all` | 编译全部 14 个平台 CLI + Daemon |
| `make build-openwrt-all` | OpenWrt MIPS/MIPSEL/ARM/ARM64/x86 |
| `make build-android-all` | Android ARM64/ARMv7 |
| `make build-gui-all` | GUI Windows/macOS/Linux |
| `make build-gui-windows` | GUI Windows AMD64 编译 |
| `make build-gui-darwin` | GUI macOS Universal 编译 |
| `make clean` | 清理构建产物 |
| `make test` / `make vet` | 运行测试 / 代码静态检查 |
| `make fmt` | 代码格式化 |
| `make lint` | 代码规范检查 |
| `make mod` | 依赖整理与验证 |
| `make deps` | 下载依赖 |
| `make run` / `make run-debug` | 测试/调试模式运行 CLI |
| `make run-daemon` | 运行守护进程 |
| `make info` | 查看所有构建目标 |

---

## 🌐 兼容性

| 生态 | 兼容情况 |
|---|---|
| **v2ray-core** | 配置格式兼容 · 协议兼容 · 传输兼容 · 路由规则兼容 |
| **Xray-core** | 完全兼容 VLESS/Trojan/XTLS/Vision/Reality 规范 |
| **sing-box** | 传输层 + 路由规则兼容 |
| **DCCP RFC 4340** | 原生 DCCP 协议支持 |
| **Wails v2** | GUI 桌面程序框架 (React + Go) |

---

## 📊 性能设计

- **零 GC 压力缓冲区**：`common/buf` 基于 `sync.Pool` 的对象复用，减少内存分配
- **流复制优化**：`io.Copy` + goroutine 双向管道，高效数据转发
- **BBR 拥塞控制**：自适应带宽探测 + pacing rate 控制，优化高延迟链路
- **多路复用**：单连接承载 128+ 并发流，减少连接建立开销
- **WorkerPool**：默认 `2×CPU` 核心数，支持动态扩缩容，任务队列缓冲

---

## 🛠️ 技术栈

### CLI 核心 / 守护进程
- **语言**: Go 1.22+
- **传输**: DCCP (RFC 4340), TCP, mKCP, WebSocket, HTTP/2, QUIC, gRPC, XHTTP
- **安全**: AES-128-GCM, AES-256-GCM, ChaCha20-Poly1305, XChaCha20-Poly1305
- **隐匿**: XTLS, Vision, Reality, uTLS 指纹仿真, 随机填充

### GUI 桌面程序
- **框架**: Wails v2 (Go ⇄ 前端桥接)
- **前端**: React 18 + TypeScript
- **状态管理**: Zustand
- **样式**: Tailwind CSS 3 + Lucide Icons
- **构建**: Vite 5
- **打包**: 单个独立可执行文件（不依赖 WebView 运行时）

---

## 📄 License

MIT License. Copyright (c) 2025 Oiwest.

---

<p align="center">
  <sub>Built with ❤️ by Oiwest</sub>
</p>
<p align="center">
  <sub>⚠️ 本项目仅供学习和研究目的使用，请遵守当地法律法规。</sub>
</p>
