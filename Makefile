.PHONY: build clean linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 build_all help

GO_ENV=CGO_ENABLED=0
GO_MODULE=GO111MODULE=on
GO=env $(GO_ENV) $(GO_MODULE) go

UNAME := $(shell uname)

# Get version from Git tag, fallback to default if no tag
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
ifeq ($(GIT_TAG), )
	GIT_TAG=1.7.4
endif

ifeq ($(BLADE_VERSION), )
	BLADE_VERSION=$(GIT_TAG)
endif

CHAOSBLADE_PATH=build/cache/chaosblade

# JVM spec file path configuration
ifeq ($(JVM_SPEC_PATH), )
	JVM_SPEC_PATH=$(CHAOSBLADE_PATH)/yaml
endif

BUILD_TARGET=target

# Architecture-specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
PLATFORM=$(GOOS)_$(GOARCH)
BUILD_TARGET_PLATFORM_DIR=$(BUILD_TARGET)/chaosblade-$(BLADE_VERSION)-$(PLATFORM)
BUILD_TARGET_PLATFORM_YAML=$(BUILD_TARGET_PLATFORM_DIR)/yaml
BINARY_NAME=chaosblade-exec-cri

CRI_OS_YAML_FILE_NAME=chaosblade-cri-spec-$(BLADE_VERSION).yaml
CRI_OS_YAML_FILE_PATH_PLATFORM=$(BUILD_TARGET_PLATFORM_YAML)/$(CRI_OS_YAML_FILE_NAME)

ifeq ($(GOOS), linux)
	GO_FLAGS=-ldflags="-s -w"
endif


# Default target
.DEFAULT_GOAL := help

# Build all platforms
build_all: clean
	@echo "Building for all supported platforms..."
	@echo "======================================"
	@echo ""
	@echo "Building Linux AMD64..."
	$(MAKE) linux_amd64
	@echo ""
	@echo "Building Linux ARM64..."
	$(MAKE) linux_arm64
	@echo ""
	@echo "Building macOS AMD64..."
	$(MAKE) darwin_amd64
	@echo ""
	@echo "Building macOS ARM64..."
	$(MAKE) darwin_arm64
	@echo ""
	@echo ""
	@echo "======================================"
	@echo "All platforms built successfully!"
	@echo "Output directories:"
	@ls -1 target/ | grep chaosblade- | sed 's/^/  - /'

# Help command
help:
	@echo "ChaosBlade CRI Executor - Build System"
	@echo "======================================"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@echo "  Platform-specific builds:"
	@echo "    build     	   - Build for current platform"
	@echo "    linux_amd64     - Build for Linux AMD64 architecture"
	@echo "    linux_arm64     - Build for Linux ARM64 architecture"
	@echo "    darwin_amd64    - Build for macOS AMD64 architecture"
	@echo "    darwin_arm64    - Build for macOS ARM64 architecture"
	@echo ""
	@echo "  Build targets:"
	@echo "    build_all       - Build for all supported platforms"
	@echo ""
	@echo "  Utility targets:"
	@echo "    clean           - Clean all build artifacts"
	@echo "    test            - Run tests with race detection"
	@echo "    help            - Show this help message"
	@echo ""
	@echo "  Examples:"
	@echo "    make build_all      # Build for all platforms"
	@echo "    make linux_amd64    # Build for Linux AMD64"
	@echo "    make darwin_amd64   # Build for macOS AMD64"
	@echo "    make clean          # Clean build artifacts"
	@echo "    make build JVM_SPEC_PATH=/path/to/jvm/specs  # Build with custom JVM spec path"
	@echo ""
	@echo "  Version management:"
	@echo "    Version is automatically detected from Git tags (e.g., v1.7.4)"
	@echo "    If no Git tag is found, default version 1.7.4 is used"
	@echo "    Build output: target/chaosblade-{version}-{platform}/"
	@echo ""
	@echo "  Environment variables:"
	@echo "    BLADE_VERSION   - Override version (default: from Git tag)"
	@echo "    JVM_SPEC_PATH   - JVM spec file directory (default: build/cache/chaosblade/yaml)"
	@echo "    GOOS           - Target OS (linux, darwin)"
	@echo "    GOARCH         - Target architecture (amd64, arm64)"
	@echo ""


build:
	$(MAKE) build_platform GOOS=$(GOOS) GOARCH=$(GOARCH)
# Platform-specific builds
linux_amd64:
	$(MAKE) build_platform GOOS=linux GOARCH=amd64

linux_arm64:
	$(MAKE) build_platform GOOS=linux GOARCH=arm64

darwin_amd64:
	$(MAKE) build_platform GOOS=darwin GOARCH=amd64

darwin_arm64:
	$(MAKE) build_platform GOOS=darwin GOARCH=arm64


# Build for specific platform
build_platform: pre_build_platform build_yaml_platform

pre_build_platform:
	mkdir -p $(BUILD_TARGET_PLATFORM_YAML)


# Check if JVM spec file exists, create empty one if not
ensure_jvm_spec:
	@if [ ! -f $(JVM_SPEC_PATH)/chaosblade-jvm-spec-$(BLADE_VERSION).yaml ]; then \
		echo "JVM spec file not found at $(JVM_SPEC_PATH)/chaosblade-jvm-spec-$(BLADE_VERSION).yaml, creating empty file..."; \
		mkdir -p $(JVM_SPEC_PATH); \
		echo "# Empty JVM spec file for CRI build" > $(JVM_SPEC_PATH)/chaosblade-jvm-spec-$(BLADE_VERSION).yaml; \
	fi

# Build CRI-only specification (without JVM dependency)
build_cri_spec: pre_build_platform
	@echo "Building CRI-only specification..."
	$(GO) run build/spec.go $(CRI_OS_YAML_FILE_PATH_PLATFORM) cri

# Legacy build with JVM dependency (for backward compatibility)
build_yaml: ensure_jvm_spec
	$(GO) run build/spec.go $(CRI_OS_YAML_FILE_PATH_PLATFORM) cri $(JVM_SPEC_PATH)/chaosblade-jvm-spec-$(BLADE_VERSION).yaml

build_yaml_platform: ensure_jvm_spec
	env CGO_ENABLED=0 GO111MODULE=on GOOS= GOARCH= go run build/spec.go $(CRI_OS_YAML_FILE_PATH_PLATFORM) cri $(JVM_SPEC_PATH)/chaosblade-jvm-spec-$(BLADE_VERSION).yaml

# test
test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

# clean all build result
clean:
	go clean ./...
	rm -rf $(BUILD_TARGET)
