# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= pm-agent
GO_MOD ?= readonly
BUILD_DIR ?= build/artifacts
PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
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

SHELL := /bin/bash
.SHELLFLAGS := -e -c

ifeq ("$(wildcard ./common.mk)","")
	ifeq ("$(wildcard ../common.mk)","")
		$(error Neither ./common.mk nor ../common.mk found. Please provide a common.mk file.)
	else
		include ../common.mk
	endif
else
	include ./common.mk
endif

# Documented targets:
#   all:        Build, lint, and test the agent
#   pmabuild:   Build the agent binary
#   pmabuild_with_race: Build with Go race detector
#   pmabuild_with_cover: Build with cover flags
#   pmainstall: Install the agent binary to $(PREFIX)/bin
#   package:    Build a Debian package (requires dh_make, dpkg-buildpackage)
#   tarball:    Create a tarball using common-tarball
#   clean:      Remove build artifacts
#   test:       Run unit tests (from common.mk)
#   fuzztest:   Run fuzz tests (from common.mk)

.PHONY: all build clean help lint package test fuzztest tarball deb-push integration_test

all: build test

lint: golint

clean:
	@echo "---MAKEFILE CLEAN---"
	rm -rf build
	@echo "---END MAKEFILE CLEAN---"

pmainstall:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

pmabuild:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/platform-manageability-agent/platform-manageability-agent.go

pmabuild_with_cover:
	CGO_ENABLED=0 go build -cover -coverpkg=./cmd/platform-manageability-agent -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/platform-manageability-agent/platform-manageability-agent.go

pmabuild_with_race:
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/platform-manageability-agent/platform-manageability-agent.go

dmmbuild:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/dm-manager-mock cmd/dm-manager-mock/dm-manager-mock.go

build: pmabuild ssmbuild dmmbuild

integration_test: pmabuild_with_race ssmbuild dmmbuild
	sudo cp test/_dummy-cert.pem /usr/local/share/ca-certificates/_dummy.crt
	sudo update-ca-certificates
	PMA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/pm-agent_race \
	DMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/dm-manager-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy-cert.pem \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy-key.pem \
	PMA_VERSION="Platform Manageability Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./cmd/platform-manageability-agent/platform-manageability-agent_test.go

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover: cover_unit cover_integration
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/platform-manageability-agent#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml

cover_integration: pmabuild_with_cover ssmbuild dmmbuild
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	PMA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/pm-agent_cover \
	DMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/dm-manager-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	TEST_CONFIG_PATH=$(shell pwd)/test/test_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy-cert.pem \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy-key.pem \
	PMA_VERSION="Platform Manageability Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./cmd/platform-manageability-agent/platform-manageability-agent_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

test: common-unit-test

fuzztest: common-fuzztest

package:
	mkdir -p $(PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PACKAGE_BUILD_DIR)
	cp -r ../LICENSES $(PACKAGE_BUILD_DIR)
	cp ../common.mk $(PACKAGE_BUILD_DIR)
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_BUILD_DIR)/Makefile
	cd $(PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b

tarball: common-tarball

deb-push: common-deb-push
