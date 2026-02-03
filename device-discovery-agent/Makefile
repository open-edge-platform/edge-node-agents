# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# device-discovery-agent Makefile

SHELL = bash -eu -o pipefail

NAME ?= device-discovery-agent
FULL_NAME ?= device-discovery-agent
GO_MOD ?= readonly
VENV_NAME ?= venv_ddagent
BUILD_DIR ?= build/artifacts
PREFIX ?= /usr/local
PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
ifneq (,$(findstring dev,$(VERSION)))
	PKG_VERSION = $(VERSION)-$(COMMIT)
else
	PKG_VERSION = $(VERSION)
endif
TARBALL_DIR := $(BUILD_DIR)/$(FULL_NAME)-$(PKG_VERSION)
REGISTRY := 080137407410.dkr.ecr.us-west-2.amazonaws.com

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build directory
CMD_DIR=./cmd/device-discovery

# Include shared makefile, if it exists
ifneq ("$(wildcard ../common.mk)","")
	include ../common.mk
else ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
endif

.PHONY: all build clean test fmt vet lint ddabuild package tarball install integration_test cover package_test help deb-push
.PHONY: ombuild

all: test build

lint: golint

# Build mock servers for testing
ombuild:
	@echo "Building onboarding-mock..."
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	$(GOCMD) build -ldflags="-s -w -extldflags=-static \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/onboarding-mock cmd/onboarding-mock/onboarding-mock.go


ddabuild:
	@echo "Building $(NAME)..."
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	$(GOCMD) build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) $(CMD_DIR)

ddabuild_with_race:
	@echo "Building $(NAME) with race detector..."
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race $(CMD_DIR)

ddabuild_with_cover:
	@echo "Building $(NAME) with coverage..."
	CGO_ENABLED=0 go build -cover -coverpkg=./cmd/device-discovery -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/device-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover $(CMD_DIR)

build: ddabuild ombuild

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(NAME)

test: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v ./internal/... ./cmd/...

fuzztest: common-fuzztest

cover_unit:
	@echo "Running unit tests with coverage..."
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

integration_test: ddabuild_with_race ombuild
	@echo "Running integration tests..."
	DDA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/device-discovery-agent_race \
	OM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/onboarding-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/test/config_example.yaml \
	DDA_VERSION="Device Discovery Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./test/integration_test.go

cover_integration: ddabuild_with_cover ombuild
	@echo "Running integration tests with coverage..."
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	DDA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/device-discovery-agent_cover \
	OM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/onboarding-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/test/config_example.yaml \
	DDA_VERSION="Device Discovery Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./test/integration_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

cover: cover_unit cover_integration
	@echo "Generating combined coverage report..."
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration
	@echo "Coverage report generated at $(BUILD_DIR)/coverage/coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

install:
	@echo "Installing $(NAME)..."
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin/

package:
	@echo "---MAKEFILE DEBIAN PACKAGE---"
	mkdir -p $(PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PACKAGE_BUILD_DIR)
	cp -r ../LICENSES $(PACKAGE_BUILD_DIR)
	cp ../common.mk $(PACKAGE_BUILD_DIR)
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_BUILD_DIR)/Makefile
	cd $(PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b
	@echo "---END MAKEFILE DEBIAN PACKAGE---"

package_test: package
	@echo "Testing Debian package..."
	@echo "Setting debconf values..."
	echo "device-discovery-agent device-discovery-agent/onboarding.serviceURL string localhost" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/onboarding.servicePort string 50051" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/discovery.serviceURL string localhost" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/auth.keycloakURL string keycloak.example.com" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/onboarding.caCertPath string /etc/intel_edge_node/orch-ca-cert/orch-ca.pem" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/sysinfo.autoDetect boolean true" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/debug boolean false" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/disableInteractive boolean false" | sudo debconf-set-selections
	echo "device-discovery-agent device-discovery-agent/useKernelArgs boolean false" | sudo debconf-set-selections
	@echo "Installing package..."
	sudo apt-get install -y ./$(BUILD_DIR)/$(FULL_NAME)_$(PKG_VERSION)_amd64.deb
	@echo "Package installed successfully"
	@echo "Checking package contents..."
	dpkg -l | grep device-discovery-agent
	@echo "Verifying installed files..."
	test -f /opt/edge-node/bin/device-discovery-agent
	test -f /etc/edge-node/node/confs/device-discovery-agent.env
	test -f /etc/apparmor.d/opt.edge-node.bin.device-discovery-agent
	test -f /etc/sudoers.d/device-discovery-agent
	@echo "Verifying configuration values..."
	@grep -q "^OBM_SVC=localhost" /etc/edge-node/node/confs/device-discovery-agent.env
	@grep -q "^OBM_PORT=50051" /etc/edge-node/node/confs/device-discovery-agent.env
	@grep -q "^OBS_SVC=localhost" /etc/edge-node/node/confs/device-discovery-agent.env
	@grep -q "^AUTO_DETECT=true" /etc/edge-node/node/confs/device-discovery-agent.env
	@echo "All configuration values verified successfully"
	@echo "Cleaning up..."
	sudo apt-get purge -y device-discovery-agent
	@echo "Package test completed successfully"

tarball:
	@echo "Creating source tarball..."
	mkdir -p $(TARBALL_DIR)
	cp -r cmd configs debian internal test go.mod go.sum Makefile README.md VERSION $(TARBALL_DIR)/
	tar -czf $(BUILD_DIR)/$(FULL_NAME)-$(PKG_VERSION).tar.gz -C $(BUILD_DIR) $(FULL_NAME)-$(PKG_VERSION)
	rm -rf $(TARBALL_DIR)
	@echo "Tarball created: $(BUILD_DIR)/$(FULL_NAME)-$(PKG_VERSION).tar.gz"

deb-push: common-deb-push

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
