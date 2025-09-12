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
	@# Help: Build Debian package using fpm-templates
	mkdir -p $(INBM_PACKAGE_BUILD_DIR)/DEBIAN $(INBM_PACKAGE_BUILD_DIR)/usr/bin $(INBM_PACKAGE_BUILD_DIR)/etc $(INBM_PACKAGE_BUILD_DIR)/etc/apparmor.d $(INBM_PACKAGE_BUILD_DIR)/etc/intel_manageability/public $(INBM_PACKAGE_BUILD_DIR)/usr/lib/systemd/system $(INBM_PACKAGE_BUILD_DIR)/usr/share
	
	# Copy the binaries
	cp $(BUILD_DIR)/inbc $(INBM_PACKAGE_BUILD_DIR)/usr/bin/inbc
	cp $(BUILD_DIR)/inbd $(INBM_PACKAGE_BUILD_DIR)/usr/bin/inbd
	
	# Create JWT token directory structure and empty access_token file
	mkdir -p $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node/tokens/release-service
	touch $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node/tokens/release-service/access_token
	
	# Copy configuration files
	if [ -f fpm-templates/etc/intel_manageability.conf ]; then \
		cp fpm-templates/etc/intel_manageability.conf $(INBM_PACKAGE_BUILD_DIR)/etc/intel_manageability.conf; \
	fi
	if [ -f fpm-templates/etc/firmware_tool_info.conf ]; then \
		cp fpm-templates/etc/firmware_tool_info.conf $(INBM_PACKAGE_BUILD_DIR)/etc/firmware_tool_info.conf; \
	fi
	
	# Copy schema files
	if [ -f fpm-templates/usr/share/inbd_schema.json ]; then \
		cp fpm-templates/usr/share/inbd_schema.json $(INBM_PACKAGE_BUILD_DIR)/usr/share/inbd_schema.json; \
	fi
	if [ -f fpm-templates/usr/share/firmware_tool_config_schema.json ]; then \
		cp fpm-templates/usr/share/firmware_tool_config_schema.json $(INBM_PACKAGE_BUILD_DIR)/usr/share/firmware_tool_config_schema.json; \
	fi
	
	# Copy other files
	if [ -f fpm-templates/etc/apparmor.d/usr.bin.inbd ]; then \
		cp fpm-templates/etc/apparmor.d/usr.bin.inbd $(INBM_PACKAGE_BUILD_DIR)/etc/apparmor.d/usr.bin.inbd; \
	fi
	if [ -f fpm-templates/usr/bin/UpdateFirmwareBlobFwupdtool.sh ]; then \
		cp fpm-templates/usr/bin/UpdateFirmwareBlobFwupdtool.sh $(INBM_PACKAGE_BUILD_DIR)/usr/bin/UpdateFirmwareBlobFwupdtool.sh; \
		chmod 755 $(INBM_PACKAGE_BUILD_DIR)/usr/bin/UpdateFirmwareBlobFwupdtool.sh; \
	fi
	if [ -f fpm-templates/usr/bin/provision-tc ]; then \
		cp fpm-templates/usr/bin/provision-tc $(INBM_PACKAGE_BUILD_DIR)/usr/bin/provision-tc; \
		chmod 700 $(INBM_PACKAGE_BUILD_DIR)/usr/bin/provision-tc; \
	fi
	if [ -f fpm-templates/usr/lib/systemd/system/inbd.service ]; then \
		cp fpm-templates/usr/lib/systemd/system/inbd.service $(INBM_PACKAGE_BUILD_DIR)/usr/lib/systemd/system/inbd.service; \
	fi
	
	# Copy postinst script
	if [ -f fpm-templates/DEBIAN/postinst ]; then \
		cp fpm-templates/DEBIAN/postinst $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/postinst; \
		chmod 755 $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/postinst; \
	fi
	
	# Create conffiles
	echo "/etc/intel_manageability.conf" > $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/conffiles
	echo "/etc/firmware_tool_info.conf" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/conffiles
	echo "/etc/intel_edge_node/tokens/release-service/access_token" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/conffiles
	
	# Set proper permissions
	chmod 640 $(INBM_PACKAGE_BUILD_DIR)/etc/intel_manageability.conf 2>/dev/null || true
	chmod 644 $(INBM_PACKAGE_BUILD_DIR)/etc/firmware_tool_info.conf 2>/dev/null || true
	chmod 750 $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node 2>/dev/null || true
	chmod 750 $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node/tokens 2>/dev/null || true
	chmod 750 $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node/tokens/release-service 2>/dev/null || true
	chmod 640 $(INBM_PACKAGE_BUILD_DIR)/etc/intel_edge_node/tokens/release-service/access_token 2>/dev/null || true
	chmod 644 $(INBM_PACKAGE_BUILD_DIR)/usr/share/inbd_schema.json 2>/dev/null || true
	chmod 644 $(INBM_PACKAGE_BUILD_DIR)/usr/share/firmware_tool_config_schema.json 2>/dev/null || true
	
	# Create control file
	echo "Package: intel-inbm" > $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	echo "Version: $(PKG_VERSION)" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	echo "Architecture: amd64" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	echo "Maintainer: Intel In-Band Manageability Team <team@intel.com>" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	echo "Description: Intel In-Band Manageability Tools" >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	echo " This package contains the inbc CLI and inbd daemon for Intel In-Band Manageability." >> $(INBM_PACKAGE_BUILD_DIR)/DEBIAN/control
	
	# Build the package
	dpkg-deb --build $(INBM_PACKAGE_BUILD_DIR) $(BUILD_DIR)/intel-inbm.deb
	@echo "---END MAKEFILE DEBIAN PACKAGE---"

tarball:
	@echo "---MAKEFILE TARBALL---"
	@# Help: Create source tarball
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ pkg/ proto/ internal/ fpm-templates/ Makefile go.mod go.sum ../common.mk $(TARBALL_DIR)
	if [ -f VERSION ]; then cp VERSION $(TARBALL_DIR); fi
	cp *.yaml *.md *.txt LICENSE $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)
	@echo "---END MAKEFILE TARBALL---"

deb-push: common-deb-push
