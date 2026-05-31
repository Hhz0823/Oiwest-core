# Oiwest Core v2.0.1

**发布日期：** 2025-05-06

> **多平台通用代理协议内核** — 兼容 v2ray-core、Xray-core、sing-box 生态。  
> 支持 11 种传输协议、16 种代理协议、18 种平台架构。

---

## ✨ 更新内容

### 🆕 新增功能

- **全平台协议注册框架** — 新增 `proxy/registry.go`，支持插件式协议注册，便于扩展自定义入站/出站协议
- **11 种传输协议** — 新增 mKCP、XHTTP、WebSocket v2、完整 QUIC/gRPC 实现
- **BBR 拥塞控制算法族** — 集成 7 种 BBR 变体：BBR、BBRv2、BBRv3、BBRPlus、BBR-ECN、BBR-Adaptive、BBR-ProbeRTT
- **WorkerPool 多线程引擎** — 可配置工作线程池、任务队列、重试机制、动态扩缩容
- **IPv4/IPv6 双栈支持** — 多线路连接管理、故障切换、4 种负载均衡策略（latency/random/roundrobin/multiline）
- **TLS 证书自动生成** — 支持 RSA/ECDSA/Ed25519 密钥生成、CA 签发、到期自动续期
- **无头守护进程模式** — 新增 `oiwest-daemon`，支持 Linux 服务器/OpenWrt/Android 无头运行
- **平台自动检测** — 自动识别操作系统/架构/发行版，自适应配置目录

### 🔧 改进

- 版本升级至 `v2.0.1`，API 版本升级至 `v2`
- 全面重构代理管理器，支持动态协议注册
- 路由引擎增强，支持更多匹配条件
- Shadowsocks 完整 AEAD 加密实现
- VMess/VLESS/Trojan 协议实现优化

### 🐛 修复

- 修复 `syscall.SIGTERM` 在 Windows 上不兼容的问题，改用平台抽象信号处理
- 修复 GUI 程序中硬编码 `.exe` 后缀导致的跨平台兼容性问题
- 修复 `sysinfo.go` 中使用 Windows 特定 API 导致 macOS/Linux 交叉编译失败的问题
- 修复网络套接字选项中 `syscall.Handle` 在 Linux/Unix 上的类型不匹配

### ⚠️ 重大变更

- 配置文件结构：新增 `workerPool`、`bbr`、`dualStack`、`certificate` 配置节
- 传输层常量：新增 `TransportKCP`、`TransportXHTTP`、`TransportWebSocketv2`
- 代理协议枚举：统一使用 `proxy.ProtocolType` 字符串常量

---

## 📥 下载

每个 ZIP 包含 `oiwest-core`（CLI 工具）和 `oiwest-daemon`（无头守护进程）。

### CLI + Daemon（14 个平台）

| 文件 | 大小 | SHA256 |
|------|------|--------|
| `oiwest-core-v2.0.1-windows-amd64.zip` | 4.46 MB | `30A980C453ABFF049EF3AB137B18509B1F1A1A4BE233AEDCD504C2551F4CCCA7` |
| `oiwest-core-v2.0.1-windows-arm64.zip` | 4.00 MB | `321B22C69DEB14508BE1EFF918E6437CB158B17B09B050B44BA492317F010FFE` |
| `oiwest-core-v2.0.1-windows-386.zip` | 4.51 MB | `ECE8FD8E6B3FBC33E8A78236FE7BA9DB1DBB7B7AD579C9C97DD467D5CC884A7C` |
| `oiwest-core-v2.0.1-linux-amd64.zip` | 4.34 MB | `AF8C8A524BB0649BED240041C7EE148C662BDA3F8F49CA4945BAE056256E061A` |
| `oiwest-core-v2.0.1-linux-arm64.zip` | 3.93 MB | `C3E7E0B89ED77D6C792C502AE30701DD8EBC832455928878F5AB881D478EC928` |
| `oiwest-core-v2.0.1-linux-armv7.zip` | 4.23 MB | `942CFEAA50998B3925FC42B38FE939584E8C64614FEC972811DD79BEFEB2F4B6` |
| `oiwest-core-v2.0.1-linux-386.zip` | 4.32 MB | `CD2AFAB19FD1212797B6E7F0B1CDCE6F6C355E79D477E1D1D417058778353D18` |
| `oiwest-core-v2.0.1-linux-mips.zip` | 4.18 MB | `00F70E0A5B12EB27A7A614E68FD3E8E6237D22F097CA8AA5A78E446143B81FEA` |
| `oiwest-core-v2.0.1-linux-mipsle.zip` | 4.11 MB | `4034D1281120E96FC029F1BB9245E21FEC9FAED615B8046EE3A5A429A050FF76` |
| `oiwest-core-v2.0.1-linux-mips64.zip` | 3.94 MB | `AA18338636BF1B7F84228EA914F2BD1839E0E3816777C47760ED80D195FCBEB5` |
| `oiwest-core-v2.0.1-linux-mips64le.zip` | 3.87 MB | `1A57AA7B939AAEBDA0597FEAAB22C3E69559946FF318F18E06D1AC2DEFC9B1CD` |
| `oiwest-core-v2.0.1-darwin-amd64.zip` | 4.41 MB | `7C264C09B93550AC3D1A05E054A53879217C6E742CE2C864FB707FBC5B306F48` |
| `oiwest-core-v2.0.1-darwin-arm64.zip` | 4.08 MB | `7F7CF0368F79B426FCBB61A4D6260A007E3CCC89E177022FADBCC552542A509C` |
| `oiwest-core-v2.0.1-android-arm64.zip` | 4.12 MB | `93B7204A0853FF220976F6E30AEEEE5285C790B1E0C3AF41C25E6035C4942C94` |

### GUI 桌面程序（4 个包）

| 文件 | 大小 | SHA256 |
|------|------|--------|
| `oiwest-core-v2.0.1-gui-windows-amd64.zip` | 1.92 MB | `6C5039DD19B25A4587EFA88EE8835B37479E5B13D868D2239F56A82D361BD1DF` |
| `oiwest-core-v2.0.1-gui-windows-arm64.zip` | 1.74 MB | `FEE3BB7FFDE163C970FA4A70EAF6E395C6815B60693AF62CBCDB2A9F2CD5CB54` |
| `oiwest-core-v2.0.1-gui-darwin-amd64.zip` | 1.82 MB | `58AC9060DDC738B8A188F8EDD395C7F2CE3ACFC35C240AF9C6F5776A0EE951B9` |
| `oiwest-core-v2.0.1-gui-darwin-arm64.zip` | 1.70 MB | `35092D5D0AE00B12006349ABA98E94CC600EE0F370E4F2CE80F329F41E82884F` |

---

## 🖥️ 平台支持矩阵

| 系统 | 架构 | CLI | Daemon | GUI |
|------|------|:---:|:------:|:---:|
| Windows 10/11 | amd64 | ✅ | ✅ | ✅ |
| Windows on ARM | arm64 | ✅ | ✅ | ✅ |
| Windows 32-bit | 386 | ✅ | ✅ | — |
| Ubuntu/Debian/Fedora/CentOS | amd64 | ✅ | ✅ | ⚠️ |
| Ubuntu/Debian/Fedora | arm64 | ✅ | ✅ | — |
| Ubuntu/Debian | armv7 | ✅ | ✅ | — |
| Linux x86 | 386 | ✅ | ✅ | — |
| OpenWrt ar71xx/ath79 | mips | ✅ | ✅ | — |
| OpenWrt ramips/mt7621 | mipsle | ✅ | ✅ | — |
| OpenWrt 高端路由 | mips64 | ✅ | ✅ | — |
| OpenWrt rockchip/ipq807x | arm64 | ✅ | ✅ | — |
| macOS Intel | amd64 | ✅ | ✅ | ✅ |
| macOS Apple Silicon M1-M4 | arm64 | ✅ | ✅ | ✅ |
| Android (Termux) | arm64 | ✅ | ✅ | — |

> ⚠️ **Linux GUI**：仅可在安装了 `libgtk-3-dev` 和 `libwebkit2gtk-4.0-dev` 的 Linux 实体机上编译运行，`make build-gui-linux`。

---

## 🚀 快速开始

### 桌面 GUI（Windows / macOS）

下载对应平台的 GUI ZIP → 解压 → 双击运行 `oiwest-core-gui` 即可。

### 守护进程模式（Linux 服务器 / OpenWrt / Android）

```bash
# 解压
unzip oiwest-core-v2.0.1-linux-amd64.zip

# 启动守护进程（自动检测平台，自动创建配置）
./oiwest-daemon
```

### CLI 模式

```bash
# 指定配置文件
./oiwest-core -config config.json

# 测试模式（使用内置默认配置）
./oiwest-core -test

# 调试模式
./oiwest-core -debug
```

---

## 🔧 从源码构建

```bash
# 前置依赖：Go 1.22.0+
git clone https://github.com/Hhz0823/oiwest-core.git
cd oiwest-core
go mod download

# 编译当前平台
make build

# 编译全部 14 个平台
make build-all

# 编译守护进程
make build-daemon
```

---

## 📦 功能特性总览

### 代理协议

| 入站 | 出站 |
|------|------|
| VMess · VLESS · Trojan · Shadowsocks | Freedom · Blackhole · Direct · DNS |
| SOCKS5 · HTTP · Dokodemo-door | VMess · VLESS · Trojan · Shadowsocks |
| Loopback · DCCP | SOCKS · DCCP |

### 传输协议

TCP · mKCP · WebSocket · WebSocket v2 · HTTP/2 · QUIC · gRPC · XHTTP · DCCP (RFC 4340)

### BBR 拥塞控制

BBR · BBRv2 · BBRv3 · BBRPlus · BBR-ECN · BBR-Adaptive · BBR-ProbeRTT

### 网络特性

- IPv4/IPv6 双栈：latency / random / roundrobin / multiline 策略
- WorkerPool：多线程任务池，默认 2×CPU 核心，支持动态扩缩容
- TLS 证书：RSA/ECDSA/Ed25519 自动生成，CA 签发，到期续期
- 路由引擎：域名 / IP(CIDR) / 端口 / 协议 / 入站标签 多条件匹配
- 多路复用：帧级 MuxSession，128+ 并发流，KeepAlive 保活

### 混淆与隐匿

XTLS · Vision · Reality · 随机填充 · XOR · uTLS 指纹仿真（Chrome/Firefox/Safari/iOS/Android/Edge/360/QQ/Random）

### 加密

AES-128-GCM · AES-256-GCM · ChaCha20-Poly1305 · XChaCha20-Poly1305

---

## 📁 项目结构

```
oiwest-core/
├── cmd/oiwest-core/          # CLI 入口
├── cmd/oiwest-daemon/        # 无头守护进程入口
├── cmd/gui/services/         # GUI 后端服务层
├── core/                     # 核心引擎 + WorkerPool
├── proxy/                    # 代理协议实现 (registry + vmess/vless/trojan/ss)
├── transport/                # 传输协议 (TCP/mKCP/WS/QUIC/gRPC/XHTTP/DCCP/BBR)
├── common/                   # 公共组件 (buf/crypto/net/tls/platform)
├── features/                 # 高级特性 (multiplex + stealth)
├── config/                   # 配置解析
├── router/                   # 路由引擎
├── frontend/                 # Wails GUI 前端 (React + Vite)
├── app.go                    # Wails 主应用
├── Makefile                  # 交叉编译脚本
└── build/zip/                # ZIP 分发包
```

---

## 🐳 Docker / OpenWrt / Android

### Android (Termux)

```bash
pkg install golang
git clone https://github.com/Hhz0823/oiwest-core.git
cd oiwest-core
make build-android-arm64
./build/android-arm64/oiwest-daemon
```

或直接下载 `oiwest-core-v2.0.1-android-arm64.zip`。

### OpenWrt

选择对应架构的 ZIP，解压到路由器：

```bash
# 例：MT7621 路由器
wget https://github.com/Hhz0823/oiwest-core/releases/download/v2.0.1/oiwest-core-v2.0.1-linux-mipsle.zip
unzip oiwest-core-v2.0.1-linux-mipsle.zip -d /etc/oiwest-core
chmod +x /etc/oiwest-core/oiwest-daemon
/etc/oiwest-core/oiwest-daemon
```

---

## 📜 许可

MIT License

---

<p align="center">
  <sub>Built with ❤️ by Oiwest</sub>
</p>
