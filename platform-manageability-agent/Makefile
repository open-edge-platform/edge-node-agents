# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= platform-manageability-agent
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
	@echo "---MAKEFILE BUILD---"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/platform-manageability-agent/platform-manageability-agent.go
	@echo "---END MAKEFILE Build---"

pmabuild_with_race:
	@echo "---MAKEFILE BUILD WITH RACE---"
	CGO_ENABLED=1 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -race -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/platform-manageability-agent/platform-manageability-agent.go
	@echo "---END MAKEFILE BUILD WITH RACE---"

build: pmabuild

test: common-unit-test

fuzztest: common-fuzztest

package:
	@echo "---MAKEFILE PACKAGE---"
	mkdir -p $(PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PACKAGE_BUILD_DIR)
	cp -r ../LICENSES $(PACKAGE_BUILD_DIR)
	cp ../common.mk $(PACKAGE_BUILD_DIR)
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_BUILD_DIR)/Makefile
	cd $(PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b
	@echo "---END MAKEFILE PACKAGE---"

tarball: common-tarball

deb-push: common-deb-push

integration_test:
	@echo "TODO: Add integration tests for platform-manageability-agent"
