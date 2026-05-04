# Oiwest Core

基于 RFC 4340 的 DCCP（Datagram Congestion Control Protocol，数据报拥塞控制协议）传输协议内核，使用 Go 语言编写，兼容 v2ray-core、Xray-core、sing-box。  
附带完整的图形用户界面 (GUI) 程序，支持节点管理、流量监控、系统代理配置等功能。

## 项目概述

Oiwest Core 是一个高性能、模块化的 DCCP 协议实现，将 DCCP 作为传输层协议引入代理通信领域。通过拥塞控制、多路复用和多种混淆技术，提供安全、高效的数据传输能力。

项目包含两个主要组件：

- **核心引擎 (Core)** — DCCP 协议内核，CLI 命令行程序
- **图形界面 (GUI)** — 基于 Wails v2 + React 的 Windows 桌面程序，用于管理核心引擎

## 功能特性

### 传输协议
- **DCCP** (RFC 4340) — 核心传输协议，支持 CCID2/CCID3/CCID4 拥塞控制
- **TCP** — 标准 TCP 传输
- **WebSocket** — 基于 WebSocket 的流式传输
- **QUIC** — 基于 QUIC 的高性能传输
- **gRPC** — 基于 gRPC 的传输支持

### 代理协议
- VMess — v2ray 标准协议
- VLESS — 轻量级无状态协议（支持 XTLS Vision Flow）
- Trojan — Trojan 代理协议
- Shadowsocks — Shadowsocks 协议（AES-GCM / ChaCha20-Poly1305）
- SOCKS5 — 标准 SOCKS5 代理
- HTTP — HTTP 代理

### 混淆与隐匿
- **XTLS** — XTLS Vision/Reality 流控混淆
- **Random Padding** — 随机填充混淆，规避流量特征检测
- **XOR Obfuscation** — 基于异或的数据混淆
- **UDP Obfuscation** — UDP 数据包混淆
- **DTLS Obfuscation** — DTLS 伪装
- **WireGuard Obfuscation** — WireGuard 协议伪装
- **Fingerprint** — 支持 Chrome / Firefox / Safari / iOS / Android / Edge / Random 指纹模拟

### 加密算法
- AES-256-GCM
- ChaCha20-Poly1305
- XChaCha20-Poly1305

### 路由功能
- 域名匹配（全匹配、关键词、正则表达式）
- IP 地址匹配（CIDR / 精确匹配）
- 端口范围匹配
- 入站/出站标签匹配
- 协议类型匹配
- 负载均衡（Balancer）

### GUI 功能
- **服务器节点管理** — 添加、编辑、删除、分组管理节点
- **多协议支持** — VMess / VLESS / Trojan / Shadowsocks / SOCKS / HTTP
- **传输配置** — TCP / WebSocket / gRPC / QUIC / DCCP
- **链接导入/导出** — 支持 vmess:// 格式分享链接
- **核心进程管理** — 一键启动、停止、重启核心引擎
- **系统代理** — Windows 系统代理自动配置（全局 / PAC / 直连）
- **流量监控** — 实时上下行速度、总流量统计、数据包统计
- **运行状态** — 核心运行状态展示、运行时长监控
- **实时日志** — 彩色日志查看器，自动滚动

### 其他
- 流量统计（上行/下行、活跃连接数）
- JSON 配置文件格式，兼容主流配置结构
- 跨平台支持：Linux / Windows / macOS（amd64 / arm64）
- 信号监听，优雅退出

## 快速开始

### 环境要求
- Go 1.22+
- Node.js 18+（构建 GUI 时需要）
- Make（可选）

### 构建 CLI 核心

```bash
# 构建当前平台
make build

# 构建所有平台
make build-all

# 构建指定平台
make build-linux-amd64
make build-linux-arm64
make build-windows-amd64
make build-darwin-amd64
make build-darwin-arm64
```

直接使用 go build：

```bash
go build -o build/oiwest-core ./cmd/oiwest-core
```

### 构建 GUI 程序

GUI 基于 [Wails v2](https://wails.io/) 框架，将前端（React + TypeScript）和后端（Go）编译为单个 Windows 可执行文件。

```bash
# 安装 Wails CLI（首次）
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 构建 GUI 程序
wails build

# 开发模式（前端热重载）
wails dev
```

构建产物位于 `build/bin/oiwest-core-gui.exe`。

### 运行 CLI 核心

```bash
# 使用配置文件运行
./build/oiwest-core -config config.json

# 测试模式（使用默认配置）
./build/oiwest-core -test

# 调试模式
./build/oiwest-core -debug -config config.json

# 查看版本信息
./build/oiwest-core -version
```

### 运行 GUI 程序

直接双击 `build/bin/oiwest-core-gui.exe` 即可启动图形界面程序。GUI 会自动管理核心引擎进程。

## GUI 使用说明

### 仪表盘
- 实时显示上行/下行速度和总流量
- 活跃连接数和核心运行时长
- 一键启动/停止/重启核心引擎
- 系统代理开关控制
- 当前选中节点信息展示

### 服务器管理
- 节点列表展示，支持分组过滤和关键词搜索
- 添加节点：填写协议、地址、端口、UUID/密码等
- 导入节点：粘贴 vmess:// 分享链接自动解析
- 编辑/删除节点
- 单击设为当前使用节点

### 传输配置
节点支持以下传输层配置：
- **TCP** — 标准 TCP
- **WebSocket** — 需配置 Path 和 Host
- **gRPC** — 需配置 ServiceName
- **QUIC** — 基于 QUIC 传输
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

## 项目结构

```
Oiwest Core/
├── cmd/
│   └── oiwest-core/main.go       # CLI 核心程序入口
├── common/                        # 公共组件
│   ├── buf/                       # 缓冲区管理
│   ├── crypto/                    # 加密算法 (AES-GCM, ChaCha20-Poly1305)
│   ├── net/                       # 网络地址与连接抽象
│   └── protocol/                  # 协议编解码
├── config/                        # 配置解析与管理
├── core/                          # 核心引擎（启动、停止、生命周期）
├── features/
│   ├── multiplex/                 # 多路复用
│   └── stealth/                   # 隐匿与混淆（XTLS, Vision, Reality, Padding）
├── proxy/                         # 代理管理器（入站/出站处理）
├── router/                        # 路由引擎（域名/IP/端口匹配）
├── transport/                     # 传输层抽象
│   ├── dccp/                      # DCCP 协议实现（拥塞控制、握手、数据包）
│   ├── config.go                  # 传输配置
│   └── transport.go               # 传输接口定义
├── cmd/gui/services/              # GUI 后端服务层
│   ├── node.go                    # 节点管理器（CRUD / 分组 / 持久化）
│   ├── core.go                    # 核心进程管理（启动/停止/重启/日志）
│   ├── proxy.go                   # Windows 系统代理（注册表配置）
│   ├── stats.go                   # 流量统计（速度计算 / 格式化）
│   └── config.go                  # 配置生成器（多协议配置构建）
├── frontend/                      # React + TypeScript 前端
│   ├── src/
│   │   ├── components/
│   │   │   └── Sidebar.tsx        # 侧边栏导航组件
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx      # 仪表盘（流量监控 / 核心控制）
│   │   │   ├── ServerList.tsx     # 服务器节点管理（CRUD / 导入导出）
│   │   │   ├── Settings.tsx       # 系统代理设置
│   │   │   └── Logs.tsx           # 实时日志查看器
│   │   ├── stores/
│   │   │   └── appStore.ts        # Zustand 全局状态管理
│   │   ├── types/
│   │   │   └── index.ts           # TypeScript 类型定义
│   │   ├── styles/
│   │   │   └── index.css          # Tailwind CSS 样式
│   │   ├── main.tsx               # React 入口
│   │   └── App.tsx                # 根组件（路由 / 初始化）
│   ├── public/
│   │   └── vite.svg               # 应用图标
│   ├── index.html                 # HTML 模板
│   ├── package.json               # 前端依赖
│   ├── vite.config.ts             # Vite 构建配置
│   ├── tailwind.config.js         # Tailwind 主题配置
│   ├── tsconfig.json              # TypeScript 配置
│   └── postcss.config.js          # PostCSS 配置
├── app.go                         # Wails 主应用（Go ⇄ 前端桥接）
├── wails.json                     # Wails 构建配置
├── go.mod                         # Go 模块定义
├── go.sum                         # Go 依赖锁
├── Makefile                       # 构建命令
├── .gitignore
└── build/
    ├── oiwest-core.exe            # CLI 核心程序
    └── bin/
        └── oiwest-core-gui.exe    # GUI 桌面程序
```

## 配置文件

### CLI 配置

```json
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
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
    }
  ],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom"
    }
  ],
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [
      {
        "type": "field",
        "domain": ["geosite:cn"],
        "outboundTag": "direct"
      }
    ]
  }
}
```

### GUI 配置

GUI 的节点数据保存在 `~/.oiwest/servers.json`，核心配置文件保存在 `~/.oiwest/config.json`，由 GUI 程序自动管理。

## DCCP 协议说明

DCCP（Datagram Congestion Control Protocol）是 IETF RFC 4340 定义的传输层协议。它提供了：

- **不可靠数据传输** — 类似 UDP
- **拥塞控制** — 类似 TCP
- **连接建立与拆除** — 具有三次握手和四次挥手
- **多种拥塞控制算法** — CCID2（TCP-like）、CCID3（TFRC）、CCID4（TFRC-SP）

本项目将其引入代理通信领域，利用 DCCP 的特性在高延迟/高丢包网络环境下提供更好的传输性能。

## 兼容性

| 软件 | 兼容状态 |
|------|--------|
| v2ray-core | ✅ 兼容 |
| Xray-core | ✅ 兼容 |
| sing-box | ✅ 兼容 |

## 常用命令

```bash
make build          # 构建 CLI 核心
make clean          # 清理构建产物
make test           # 运行测试
make run            # 测试模式运行 CLI
make run-debug      # 调试模式运行 CLI
make install        # 安装到 $GOPATH/bin
make lint           # 代码检查
make fmt            # 代码格式化
make vet            # 代码静态分析
make mod            # 依赖整理与验证
make deps           # 下载依赖
wails build         # 构建 GUI 程序
wails dev           # 开发模式运行 GUI
```

## 技术栈

### CLI 核心
- **语言**: Go 1.22+
- **协议**: DCCP (RFC 4340), TCP, WebSocket, QUIC, gRPC
- **安全**: AES-256-GCM, ChaCha20-Poly1305, XTLS, Reality

### GUI 程序
- **框架**: Wails v2 (Go ⇄ 前端桥接)
- **前端**: React 18 + TypeScript
- **状态管理**: Zustand
- **样式**: Tailwind CSS 3 + Lucide Icons
- **构建**: Vite 5
- **打包**: 单个 Windows exe 文件

## 许可证

MIT License

---

**Warning**: 本项目仅供学习和研究目的使用，请遵守当地法律法规。
