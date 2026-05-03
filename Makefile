.PHONY: all build clean test run install lint

APP_NAME := oiwest-core
BUILD_DIR := build
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

all: build

build:
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) build

build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(MAKE) build

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) build

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) build

build-all: build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 build-darwin-arm64

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned build directory"

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/$(APP_NAME) -test

run-debug:
	$(GO) run ./cmd/$(APP_NAME) -debug -test

install:
	$(GO) install ./cmd/$(APP_NAME)

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
