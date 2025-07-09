# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= platform-manageability-agent
GO_MOD ?= readonly
BUILD_DIR ?= build/artifacts
PMA_PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
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

all: build lint test

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
	dh_make -n --single --packagename $(NAME) || true
	dpkg-buildpackage -us -uc -b
	@echo "---END MAKEFILE PACKAGE---"

tarball: common-tarball
