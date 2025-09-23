# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= in-band-manageability
GO_MOD ?= readonly
BUILD_DIR ?= build/artifacts
INBM_PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
PREFIX ?= /usr/local
VERSION := $(shell if [ -f VERSION ]; then cat VERSION; else echo "5.0.0"; fi)
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

.PHONY: all build clean help lint package test fuzztest tarball generate-proto

all: build lint test

lint: golint

clean:
	@echo "---MAKEFILE CLEAN---"
	rm -rf build
	@echo "---END MAKEFILE CLEAN---"

install:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/inbc $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/inbd $(DESTDIR)$(PREFIX)/bin

generate-proto:
	@echo "---MAKEFILE GENERATE PROTO---"
	@# Help: Generate protobuf files
	buf generate
	@echo "---END MAKEFILE GENERATE PROTO---"

inbcbuild:
	@echo "---MAKEFILE BUILD INBC---"
	@# Help: Build the inbc binary
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X main.Version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.Version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.GitCommit=$(COMMIT) \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")" \
	-o $(BUILD_DIR)/inbc cmd/inbc/main.go
	@echo "---END MAKEFILE BUILD INBC---"

inbdbuild:
	@echo "---MAKEFILE BUILD INBD---"
	@# Help: Build the inbd binary
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.Version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.GitCommit=$(COMMIT) \
	-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")" \
	-o $(BUILD_DIR)/inbd cmd/inbd/main.go
	@echo "---END MAKEFILE BUILD INBD---"

copy-licenses:
	@echo "---MAKEFILE COPY LICENSES---"
	@# Help: Copy license files to build directory
	mkdir -p $(BUILD_DIR)
	cp -r retain-3rd-party-notices $(BUILD_DIR)/
	cp third-party-programs.txt $(BUILD_DIR)/
	cp LICENSE $(BUILD_DIR)/
	@echo "---END MAKEFILE COPY LICENSES---"

build: generate-proto inbcbuild inbdbuild copy-licenses

test: common-unit-test

integration_test:
	@echo "no integration tests to run, skipping"

fuzztest: common-fuzztest

cover:
	@echo "---MAKEFILE COVERAGE---"
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/in-band-manageability#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml
	@echo "---END MAKEFILE COVERAGE---"

package: build
	@echo "---MAKEFILE DEBIAN PACKAGE---"
	@# Help: Build Debian package using debian packaging system
	mkdir -p $(INBM_PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(INBM_PACKAGE_BUILD_DIR)
	cp ../.golangci.yml $(INBM_PACKAGE_BUILD_DIR)
	cp -r ../LICENSES $(INBM_PACKAGE_BUILD_DIR)
	cp ../common.mk $(INBM_PACKAGE_BUILD_DIR)
	sed -i "s#../.golangci.yml#.golangci.yml#" $(INBM_PACKAGE_BUILD_DIR)/common.mk
	sed -i "s#VERSION#$(PKG_VERSION)#" $(INBM_PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(INBM_PACKAGE_BUILD_DIR)/Makefile
	cd $(INBM_PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b
	@echo "---END MAKEFILE DEBIAN PACKAGE---"

package_test: package
	@echo "---MAKEFILE PACKAGE TEST---"
	sudo apt-get install -y ./$(BUILD_DIR)/in-band-manageability_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable inbd
	@echo "Package installed and service enabled successfully"
	sudo apt-get remove -y in-band-manageability
	@echo "---END MAKEFILE PACKAGE TEST---"
	sudo systemctl disable inbd
	sudo apt-get purge -y in-band-manageability
	@echo "---END MAKEFILE PACKAGE TEST---"

tarball:
	@echo "---MAKEFILE TARBALL---"
	@# Help: Create source tarball
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ pkg/ proto/ internal/ debian/ configs/ scripts/ Makefile go.mod go.sum ../common.mk $(TARBALL_DIR)
	if [ -f VERSION ]; then cp VERSION $(TARBALL_DIR); fi
	cp *.yaml *.md *.txt LICENSE $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)
	@echo "---END MAKEFILE TARBALL---"

deb-push: common-deb-push
