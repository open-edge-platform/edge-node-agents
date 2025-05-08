# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= platform-telemetry-agent
GO_MOD ?= readonly
BUILD_DIR ?= build/artifacts
PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
PKG_VERSION:=$(shell if grep -q dev VERSION; then echo $$(cat VERSION)-$$(git rev-parse --short HEAD); else cat VERSION; fi)
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
ifneq (,$(findstring dev,$(VERSION)))
	PKG_VERSION = $(VERSION)-$(COMMIT)
else
	PKG_VERSION = $(VERSION)
endif
TARBALL_DIR := $(BUILD_DIR)/$(NAME)-$(PKG_VERSION)
REGISTRY := 080137407410.dkr.ecr.us-west-2.amazonaws.com

# Include shared makefile, if it exists
ifneq ("$(wildcard ../common.mk)","")
	include ../common.mk
else ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
endif

.PHONY: all build clean lint package test fuzztest tarball

all: build lint test

clean:
	rm -rf build

lint: golint

tmmbuild:
	@echo "---MAKEFILE BUILD TELEMETRY MANAGER MOCK---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -ldflags="-s -w -extldflags=-static \
	-X main.version=$(shell cat VERSION) \
	-X main.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/telemetrymgr-mock cmd/telemetrymgr-mock/telemetrymgr-mock.go
	@echo "---END MAKEFILE BUILD TELEMETRY MANAGER MOCK---"

ptabuild:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -buildmode=pie -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/telemetryagent/main.go

ptabuild_with_race: generate_config
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/telemetryagent/main.go
	@echo "---END MAKEFILE Build---"

build: ptabuild tmmbuild ssmbuild

ptabuild_with_cover: generate_config
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=0 go build -cover -coverpkg=./cmd/telemetryagent -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/telemetryagent/main.go
	@echo "---END MAKEFILE Build---"

test: common-unit-test

fuzztest: common-fuzztest

integration_test: ptabuild_with_race tmmbuild ssmbuild
	PTA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/platform-telemetry-agent_race \
	TMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/telemetrymgr-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/tests/config_example.yaml \
	PTA_VERSION="Platform Telemetry Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./cmd/telemetryagent/main_test.go

generate_config:
	mkdir -p $(BUILD_DIR)
	cp -n tests/config_example.yaml $(BUILD_DIR)/pt-agent.yaml

cover_integration: ptabuild_with_cover tmmbuild ssmbuild
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	PTA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/platform-telemetry-agent_cover \
	TMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/telemetrymgr-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/tests/config_example.yaml \
	PTA_VERSION="Platform Telemetry Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./cmd/telemetryagent/main_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover: cover_unit cover_integration
	@echo "---MAKEFILE COVER---"
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > build/artifacts/report.xml
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > build/artifacts/coverage.xml
	@echo "---END MAKEFILE COVER---"
	
install:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

package:
	mkdir -p $(PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PACKAGE_BUILD_DIR)
	cp ../.golangci.yml $(PACKAGE_BUILD_DIR)
	cp -r ../LICENSES ${PACKAGE_BUILD_DIR}
	cp ../common.mk $(PACKAGE_BUILD_DIR)
	sed -i "s#../.golangci.yml#.golangci.yml#" $(PACKAGE_BUILD_DIR)/common.mk
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_BUILD_DIR)/Makefile
	cd $(PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b

package_test:
	sudo apt-get install -y ./$(BUILD_DIR)/platform-telemetry-agent_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable platform-telemetry-agent
	sudo systemctl start platform-telemetry-agent
	systemctl is-active platform-telemetry-agent
	sudo systemctl disable platform-telemetry-agent
	sudo apt-get purge -y platform-telemetry-agent

tarball:
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ configs/ debian/copyright internal/ Makefile VERSION go.mod go.sum ../common.mk $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)

deb-push: common-deb-push