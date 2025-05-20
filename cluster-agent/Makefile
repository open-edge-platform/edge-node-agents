# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL=/bin/bash -o pipefail
NAME ?= cluster-agent
BUILD_DIR ?= build/artifacts
GO_MOD ?= readonly
CA_PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
ifneq (,$(findstring dev,$(VERSION)))
	PKG_VERSION = $(VERSION)-$(COMMIT)
else
	PKG_VERSION = $(VERSION)
endif
TARBALL_DIR := $(BUILD_DIR)/$(NAME)-$(PKG_VERSION)
CA_VERSION="Cluster Agent v$(VERSION)-$(COMMIT)" \
REGISTRY := 080137407410.dkr.ecr.us-west-2.amazonaws.com

# Include shared makefile, if it exists
ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
else
	include ../common.mk
endif

.PHONY: all build clean help lint list package test fuzztest tarball

all: build lint test

clean:
	@echo "---MAKEFILE CLEAN---"
	rm -rf build
	@echo "---END MAKEFILE CLEAN---"

lint: golint

cainstall:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

cabuild:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/cluster-agent/cluster-agent.go

cabuild_with_race:
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/cluster-agent/cluster-agent.go

cabuild_with_cover:
	CGO_ENABLED=0 go build -cover -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/cluster-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/cluster-agent/cluster-agent.go

combuild:
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static \
	-X main.version=$(shell cat VERSION) \
	-X main.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/cluster-orch-mock cmd/cluster-orch-mock/cluster-orch-mock.go
	@echo "---END MAKEFILE Build---"

build: cabuild combuild ssmbuild

test: common-unit-test

integration_test: cabuild_with_race combuild ssmbuild
	sudo cp test/_dummy.crt /usr/local/share/ca-certificates/_dummy.crt
	sudo update-ca-certificates
	CA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/cluster-agent_race \
	COM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/cluster-orch-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy.crt \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy.key \
	CA_VERSION="Cluster Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./cmd/cluster-agent/cluster-agent_test.go

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover_integration: cabuild_with_cover combuild ssmbuild
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	CA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/cluster-agent_cover \
	COM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/cluster-orch-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy.crt \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy.key \
	CA_VERSION="Cluster Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./cmd/cluster-agent/cluster-agent_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

fuzztest: common-fuzztest

cover: cover_unit cover_integration
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > build/artifacts/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/cluster-agent#g" build/artifacts/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > build/artifacts/coverage.xml

cover_old:
	mkdir -p build/artifacts
	go test ./internal/... -coverprofile=build/artifacts/coverage.out -v -covermode count | go-junit-report -set-exit-code > build/artifacts/report.xml
	go tool cover -html=build/artifacts/coverage.out -o build/artifacts/cover.html
	gocover-cobertura < build/artifacts/coverage.out > build/artifacts/coverage.xml

package:
	mkdir -p $(CA_PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(CA_PACKAGE_BUILD_DIR)
	cp ../.golangci.yml $(CA_PACKAGE_BUILD_DIR)
	cp -r ../LICENSES ${CA_PACKAGE_BUILD_DIR}
	cp ../common.mk $(CA_PACKAGE_BUILD_DIR)
	sed -i "s#../.golangci.yml#.golangci.yml#" $(CA_PACKAGE_BUILD_DIR)/common.mk
	sed -i "s#VERSION#$(PKG_VERSION)#" $(CA_PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(CA_PACKAGE_BUILD_DIR)/Makefile
	cd $(CA_PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b

package_test:
	echo "cluster-agent cluster-agent/cluster-orchestrator-url string localhost:12345" | sudo debconf-set-selections
	sudo apt-get install -y ./$(BUILD_DIR)/cluster-agent_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable cluster-agent
	sudo systemctl start cluster-agent
	systemctl is-active cluster-agent
	sudo systemctl disable cluster-agent
	sudo apt-get purge -y cluster-agent

tarball: common-tarball
#common deb package push
deb-push: common-deb-push