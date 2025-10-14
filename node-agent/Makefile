# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= node-agent
GO_MOD ?= readonly
BUILD_DIR ?= build/artifacts
NA_PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
PREFIX ?= /usr/local
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
ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
else
	include ../common.mk
endif

.PHONY: all build clean help lint package test fuzztest tarball

all: build test

lint: golint

clean:
	@echo "---MAKEFILE CLEAN---"
	rm -rf build
	@echo "---END MAKEFILE CLEAN---"

nainstall:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

nabuild:
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/node-agent/node-agent.go
	@echo "---END MAKEFILE Build---"

nabuild_with_race:
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/node-agent/node-agent.go
	@echo "---END MAKEFILE Build---"

nabuild_with_cover:
	CGO_ENABLED=0 go build -cover -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/node-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/node-agent/node-agent.go

hmmbuild:
	@echo "---MAKEFILE BUILD HOST MANAGER MOCK---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/hostmgr-mock cmd/hostmgr-mock/hostmgr-mock.go
	@echo "---END MAKEFILE BUILD HOST MANAGER MOCK---"

build: nabuild hmmbuild

test: common-unit-test

integration_test: nabuild_with_race hmmbuild
	sudo cp test/_dummy.crt /usr/local/share/ca-certificates/_dummy.crt
	sudo update-ca-certificates
	NA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/node-agent_race \
	HMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hostmgr-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy.crt \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy.key \
	NA_VERSION="Node Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./cmd/node-agent/node-agent_test.go

fuzztest: common-fuzztest

cover:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/node-agent#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml

cover_integration: nabuild_with_cover hmmbuild
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	NA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/node-agent_cover \
	HMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hostmgr-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	NA_VERSION="Node Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./cmd/node-agent/node-agent_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

package:
	@echo "---MAKEFILE DEBIAN PACKAGE---"
	mkdir -p $(NA_PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(NA_PACKAGE_BUILD_DIR)
	cp ../.golangci.yml $(NA_PACKAGE_BUILD_DIR)
	cp -r ../LICENSES ${NA_PACKAGE_BUILD_DIR}
	cp ../common.mk $(NA_PACKAGE_BUILD_DIR)
	sed -i "s#../.golangci.yml#.golangci.yml#" $(NA_PACKAGE_BUILD_DIR)/common.mk
	sed -i "s#VERSION#$(PKG_VERSION)#" $(NA_PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(NA_PACKAGE_BUILD_DIR)/Makefile
	cd $(NA_PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b
	@echo "---END MAKEFILE DEBIAN PACKAGE---"

# TODO: Disabled for now due to issue with caddy
package_test:
	sudo apt-get update
	sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
	echo "node-agent node-agent/onboarding.serviceURL string localhost:11223" | sudo debconf-set-selections
	echo "node-agent node-agent/onboarding.enabled boolean true" | sudo debconf-set-selections
	echo "node-agent node-agent/onboarding.heartbeatInterval string 10s" | sudo debconf-set-selections
	sudo groupadd -f bm-agents --system
	sudo useradd node-agent --system -g bm-agents
	sudo useradd platform-update-agent --system -g bm-agents
	sudo apt-get install -y ./$(BUILD_DIR)/node-agent_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable node-agent
	sudo systemctl start node-agent
	sudo journalctl -xeu caddy.service
	systemctl is-active node-agent
	sudo systemctl disable node-agent caddy
	sudo apt-get purge -y node-agent caddy

tarball: common-tarball

deb-push: common-deb-push
