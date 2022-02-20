.PHONY: build clean

GO_ENV=CGO_ENABLED=1
GO_MODULE=GO111MODULE=on
GO=env $(GO_ENV) $(GO_MODULE) go

UNAME := $(shell uname)

ifeq ($(BLADE_VERSION), )
	BLADE_VERSION=1.5.0
endif

BUILD_TARGET=target
BUILD_TARGET_DIR_NAME=chaosblade-$(BLADE_VERSION)
BUILD_TARGET_PKG_DIR=$(BUILD_TARGET)/chaosblade-$(BLADE_VERSION)
BUILD_TARGET_YAML=$(BUILD_TARGET_PKG_DIR)/yaml
BUILD_IMAGE_PATH=build/image/blade

CRI_OS_YAML_FILE_NAME=chaosblade-cri-spec-$(BLADE_VERSION).yaml
CRI_OS_YAML_FILE_PATH=$(BUILD_TARGET_YAML)/$(CRI_OS_YAML_FILE_NAME)

DOCKER_OS_YAML_FILE_NAME=chaosblade-docker-spec-$(BLADE_VERSION).yaml
DOCKER_OS_YAML_FILE_PATH=$(BUILD_TARGET_YAML)/$(DOCKER_OS_YAML_FILE_NAME)

CHAOSBLADE_PATH=build/cache/chaosblade

ifeq ($(GOOS), linux)
	GO_FLAGS=-ldflags="-linkmode external -extldflags -static"
endif

build: pre_build build_yaml

build_linux: build

pre_build:
	rm -rf $(BUILD_TARGET_PKG_DIR)
	mkdir -p $(BUILD_TARGET_YAML)

build_yaml: build/spec.go
	$(GO) run $< $(CRI_OS_YAML_FILE_PATH) cri $(CHAOSBLADE_PATH)/yaml/chaosblade-jvm-spec-$(BLADE_VERSION).yaml
	$(GO) run $< $(DOCKER_OS_YAML_FILE_PATH) docker $(CHAOSBLADE_PATH)/yaml/chaosblade-jvm-spec-$(BLADE_VERSION).yaml

# test
test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...
# clean all build result
clean:
	go clean ./...
	rm -rf $(BUILD_TARGET)
	rm -rf $(BUILD_IMAGE_PATH)/$(BUILD_TARGET_DIR_NAME)
