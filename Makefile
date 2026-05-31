.PHONY: all build clean test run install lint fmt vet mod deps
.PHONY: build-all build-all-platforms
.PHONY: build-windows-amd64 build-windows-arm64 build-windows-386
.PHONY: build-linux-amd64 build-linux-arm64 build-linux-arm build-linux-386
.PHONY: build-linux-mips build-linux-mipsle build-linux-mips64 build-linux-mips64le
.PHONY: build-darwin-amd64 build-darwin-arm64
.PHONY: build-android-arm64 build-android-arm
.PHONY: build-openwrt-mips build-openwrt-mipsel build-openwrt-arm
.PHONY: build-openwrt-x86 build-openwrt-arm64
.PHONY: build-daemon-all
.PHONY: build-gui-windows build-gui-linux build-gui-darwin

APP_NAME := oiwest-core
DAEMON_NAME := oiwest-daemon
GUI_NAME := oiwest-core-gui
BUILD_DIR := build
GO := go
GOFLAGS := -v
LDFLAGS := -s -w
BUILDTAGS_DAEMON := -tags "!cgo"
BUILDTAGS_GUI := -tags "cgo"
CGO_ENABLED_DAEMON := 0
CGO_ENABLED_GUI := 1

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

# ============================================================
# Default build
# ============================================================
all: build

build:
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED_GUI) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./app/cmd/cli

build-daemon:
	@echo "Building $(DAEMON_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(DAEMON_NAME) ./app/cmd/daemon

# ============================================================
# Windows builds
# ============================================================
build-windows-amd64:
	@echo "=== Building for Windows AMD64 ==="
	@mkdir -p $(BUILD_DIR)/windows-amd64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-amd64/$(APP_NAME).exe ./app/cmd/cli
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-amd64/$(DAEMON_NAME).exe ./app/cmd/daemon

build-windows-arm64:
	@echo "=== Building for Windows ARM64 ==="
	@mkdir -p $(BUILD_DIR)/windows-arm64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-arm64/$(APP_NAME).exe ./app/cmd/cli
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-arm64/$(DAEMON_NAME).exe ./app/cmd/daemon

build-windows-386:
	@echo "=== Building for Windows x86 ==="
	@mkdir -p $(BUILD_DIR)/windows-386
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=386 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-386/$(APP_NAME)-386.exe ./app/cmd/cli
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=windows GOARCH=386 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-386/$(DAEMON_NAME)-386.exe ./app/cmd/daemon

build-windows-all: build-windows-amd64 build-windows-arm64 build-windows-386

# ============================================================
# Linux desktop/server builds
# ============================================================
build-linux-amd64:
	@echo "=== Building for Linux AMD64 (Ubuntu/Debian/Fedora/CentOS/Arch) ==="
	@mkdir -p $(BUILD_DIR)/linux-amd64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(DAEMON_NAME) ./app/cmd/daemon
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(APP_NAME) ./app/cmd/cli

build-linux-arm64:
	@echo "=== Building for Linux ARM64 ==="
	@mkdir -p $(BUILD_DIR)/linux-arm64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm64/$(DAEMON_NAME) ./app/cmd/daemon
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm64/$(APP_NAME) ./app/cmd/cli

build-linux-arm:
	@echo "=== Building for Linux ARMv7 ==="
	@mkdir -p $(BUILD_DIR)/linux-arm
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm/$(DAEMON_NAME) ./app/cmd/daemon
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm/$(APP_NAME) ./app/cmd/cli

build-linux-386:
	@echo "=== Building for Linux x86 ==="
	@mkdir -p $(BUILD_DIR)/linux-386
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=386 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-386/$(DAEMON_NAME) ./app/cmd/daemon
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=linux GOARCH=386 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-386/$(APP_NAME) ./app/cmd/cli

# MIPS variants (for some OpenWrt routers)
build-linux-mips:
	@echo "=== Building for Linux MIPS ==="
	@mkdir -p $(BUILD_DIR)/linux-mips
	GOOS=linux GOARCH=mips GOMIPS=softfloat CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-mips/$(DAEMON_NAME) ./app/cmd/daemon

build-linux-mipsle:
	@echo "=== Building for Linux MIPS Little-Endian ==="
	@mkdir -p $(BUILD_DIR)/linux-mipsle
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-mipsle/$(DAEMON_NAME) ./app/cmd/daemon

build-linux-mips64:
	@echo "=== Building for Linux MIPS64 ==="
	@mkdir -p $(BUILD_DIR)/linux-mips64
	GOOS=linux GOARCH=mips64 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-mips64/$(DAEMON_NAME) ./app/cmd/daemon

build-linux-mips64le:
	@echo "=== Building for Linux MIPS64 LE ==="
	@mkdir -p $(BUILD_DIR)/linux-mips64le
	GOOS=linux GOARCH=mips64le CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-mips64le/$(DAEMON_NAME) ./app/cmd/daemon

build-linux-all: build-linux-amd64 build-linux-arm64 build-linux-arm build-linux-386 \
               build-linux-mips build-linux-mipsle build-linux-mips64 build-linux-mips64le

# ============================================================
# OpenWrt specific builds (popular router architectures)
# ============================================================
build-openwrt-mips:
	@echo "=== Building for OpenWrt MIPS (ar71xx/ath79) ==="
	@mkdir -p $(BUILD_DIR)/openwrt-mips
	GOOS=linux GOARCH=mips GOMIPS=softfloat CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -X main.targetDistro=openwrt" -o $(BUILD_DIR)/openwrt-mips/$(DAEMON_NAME) ./app/cmd/daemon

build-openwrt-mipsel:
	@echo "=== Building for OpenWrt MIPSEL (ramips/mt7621/mediatek) ==="
	@mkdir -p $(BUILD_DIR)/openwrt-mipsel
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -X main.targetDistro=openwrt" -o $(BUILD_DIR)/openwrt-mipsel/$(DAEMON_NAME) ./app/cmd/daemon

build-openwrt-arm:
	@echo "=== Building for OpenWrt ARM (bcm27xx/bcm53xx/ipq40xx) ==="
	@mkdir -p $(BUILD_DIR)/openwrt-arm
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -X main.targetDistro=openwrt" -o $(BUILD_DIR)/openwrt-arm/$(DAEMON_NAME) ./app/cmd/daemon

build-openwrt-arm64:
	@echo "=== Building for OpenWrt ARM64 (rockchip/ipq807x) ==="
	@mkdir -p $(BUILD_DIR)/openwrt-arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -X main.targetDistro=openwrt" -o $(BUILD_DIR)/openwrt-arm64/$(DAEMON_NAME) ./app/cmd/daemon

build-openwrt-x86:
	@echo "=== Building for OpenWrt x86 (x86_64) ==="
	@mkdir -p $(BUILD_DIR)/openwrt-x86
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -X main.targetDistro=openwrt" -o $(BUILD_DIR)/openwrt-x86/$(DAEMON_NAME) ./app/cmd/daemon

build-openwrt-all: build-openwrt-mips build-openwrt-mipsel build-openwrt-arm build-openwrt-arm64 build-openwrt-x86

# ============================================================
# macOS builds
# ============================================================
build-darwin-amd64:
	@echo "=== Building for macOS AMD64 ==="
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-amd64/$(DAEMON_NAME) ./app/cmd/daemon
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-amd64/$(APP_NAME) ./app/cmd/cli

build-darwin-arm64:
	@echo "=== Building for macOS ARM64 (Apple Silicon) ==="
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-arm64/$(DAEMON_NAME) ./app/cmd/daemon
	CGO_ENABLED=$(CGO_ENABLED_DAEMON) GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-arm64/$(APP_NAME) ./app/cmd/cli

build-darwin-all: build-darwin-amd64 build-darwin-arm64

# ============================================================
# Android builds
# ============================================================
build-android-arm64:
	@echo "=== Building for Android ARM64 ==="
	@mkdir -p $(BUILD_DIR)/android-arm64
	GOOS=android GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/android-arm64/$(DAEMON_NAME) ./app/cmd/daemon

build-android-arm:
	@echo "=== Building for Android ARMv7 ==="
	@mkdir -p $(BUILD_DIR)/android-arm
	GOOS=android GOARCH=arm GOARM=7 CGO_ENABLED=$(CGO_ENABLED_DAEMON) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/android-arm/$(DAEMON_NAME) ./app/cmd/daemon

build-android-all: build-android-arm64 build-android-arm

# ============================================================
# Daemon (headless) builds - all platforms
# ============================================================
build-daemon-all: \
	build-windows-amd64 build-windows-arm64 \
	build-linux-amd64 build-linux-arm64 build-linux-arm build-linux-386 \
	build-darwin-amd64 build-darwin-arm64

# ============================================================
# Build all platforms (daemon + CLI, no GUI)
# ============================================================
build-all: \
	build-windows-all \
	build-linux-all \
	build-darwin-all \
	build-android-all \
	build-openwrt-all

# ============================================================
# GUI builds (requires CGO and Wails frontend)
# Prepare GUI frontend embed directory
prepare-gui-frontend:
	@echo "Preparing GUI frontend assets..."
	@rm -rf gui/frontend_dist
	cp -r gui/frontend/dist gui/frontend_dist

# ============================================================
build-gui-windows:prepare-gui-frontend
	@echo "=== Building GUI for Windows AMD64 ==="
	@mkdir -p $(BUILD_DIR)/gui-windows-amd64
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) -H windowsgui" -o $(BUILD_DIR)/gui-windows-amd64/$(GUI_NAME).exe ./gui

build-gui-linux:prepare-gui-frontend
	@echo "=== Building GUI for Linux AMD64 ==="
	@mkdir -p $(BUILD_DIR)/gui-linux-amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/gui-linux-amd64/$(GUI_NAME) ./gui

build-gui-darwin:prepare-gui-frontend
	@echo "=== Building GUI for macOS Universal ==="
	@mkdir -p $(BUILD_DIR)/gui-darwin-amd64
	@mkdir -p $(BUILD_DIR)/gui-darwin-arm64
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/gui-darwin-amd64/$(GUI_NAME) ./gui
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/gui-darwin-arm64/$(GUI_NAME) ./gui

build-gui-all: build-gui-windows build-gui-linux build-gui-darwin

# ============================================================
clean-gui-frontend:
	@rm -rf gui/frontend_dist
	@echo "Cleaned GUI frontend embed directory"

# Cleanup
# ============================================================
clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned build directory"

# ============================================================
# Testing & Quality
# ============================================================
test:
	$(GO) test ./...

test-vet:
	$(GO) vet ./...

run:
	$(GO) run ./app/cmd/cli -test

run-debug:
	$(GO) run ./app/cmd/cli -debug -test

run-daemon:
	$(GO) run ./app/cmd/daemon

install:
	$(GO) install ./app/cmd/cli

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

mod:
	$(GO) mod tidy
	$(GO) mod verify

deps:
	$(GO) mod download

# ============================================================
# Info
# ============================================================
info:
	@echo "=== Oiwest Core Build System ==="
	@echo "Go version : $$($(GO) version)"
	@echo "GOOS       : $(GOOS)"
	@echo "GOARCH     : $(GOARCH)"
	@echo ""
	@echo "Available targets:"
	@echo "  build-windows-amd64   - Windows 10+ AMD64"
	@echo "  build-windows-arm64   - Windows 10+ ARM64"
	@echo "  build-windows-386     - Windows x86"
	@echo "  build-linux-amd64     - Linux AMD64 (Ubuntu/Debian/Fedora/CentOS)"
	@echo "  build-linux-arm64     - Linux ARM64"
	@echo "  build-linux-arm       - Linux ARMv7"
	@echo "  build-linux-386       - Linux x86"
	@echo "  build-linux-mips*     - Linux MIPS/MIPSLE/MIPS64"
	@echo "  build-openwrt-mips    - OpenWrt MIPS (ar71xx/ath79)"
	@echo "  build-openwrt-mipsel  - OpenWrt MIPSEL (ramips/mt7621)"
	@echo "  build-openwrt-arm     - OpenWrt ARMv7 (ipq40xx/bcm27xx)"
	@echo "  build-openwrt-arm64   - OpenWrt ARM64 (rockchip/ipq807x)"
	@echo "  build-openwrt-x86     - OpenWrt x86_64"
	@echo "  build-android-arm64   - Android ARM64"
	@echo "  build-android-arm     - Android ARMv7"
	@echo "  build-darwin-amd64    - macOS Intel"
	@echo "  build-darwin-arm64    - macOS Apple Silicon"
	@echo "  build-gui-windows     - GUI Windows AMD64"
	@echo "  build-gui-linux       - GUI Linux AMD64"
	@echo "  build-gui-darwin     - GUI macOS Universal"
	@echo "  build-all             - All daemon platforms"
	@echo "  build-gui-all         - All GUI platforms"
