# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= platform-update-agent
BUILD_DIR ?= build/artifacts
INBM_DIR := inbm
INBM_VERSION := $(shell if grep -q dev $(INBM_DIR)/VERSION; then echo $$(cat $(INBM_DIR)/VERSION)-$$(git rev-parse --short HEAD); else cat $(INBM_DIR)/VERSION; fi)
INBM_OUTPUT_DIR_NAME ?= output
PUA_PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
GO_MOD_MODE ?= readonly
FUZZTIME ?= 1s
PKG_VERSION := $(shell if grep -q dev VERSION; then echo $$(cat VERSION)-$$(git rev-parse --short HEAD); else cat VERSION; fi)
TARBALL_DIR := $(BUILD_DIR)/$(NAME)-$(PKG_VERSION)
REGISTRY := 080137407410.dkr.ecr.us-west-2.amazonaws.com

# Include shared makefile, if it exists
ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
else
	include ../common.mk
endif

.PHONY: all build clean help lint package test fuzztest

all: build lint test

clean:
	@echo "---MAKEFILE CLEAN---"
	rm -rf build
	@echo "---END MAKEFILE CLEAN---"

lint: golint inbm-buf-lint

puainstall:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

puabuild:
	@echo "---MAKEFILE PUA BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD_MODE) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.version=$(shell cat VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/$(NAME) cmd/platform-update-agent/platform_update_agent.go
	@echo "---END MAKEFILE PUA Build---"

puabuild_with_race:
	CGO_ENABLED=1 go build -race -trimpath -mod=$(GO_MOD_MODE) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w  \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.version=$(shell cat VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/platform-update-agent/platform_update_agent.go

puabuild_with_cover:
	CGO_ENABLED=0 go build -cover -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.version=$(shell cat VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/platform-update-agent/info.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/platform-update-agent/platform_update_agent.go

mmbuild:
	@echo "---MAKEFILE MM MOCK BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static \
	-X main.version=$(shell cat VERSION) \
	-X main.commit=$(shell git rev-parse --short HEAD)" \
	-o $(BUILD_DIR)/maintenance-mngr-mock cmd/mock-server/mock-server.go
	@echo "---END MAKEFILE Build---"

build: puabuild mmbuild inbm-build

# In fuzzing test, we have observed the OOM issues that appear to be caused by the fuzzer.
# A WA is to add "-parallel=1" to the fuzzing test. Please note that this WA may slow down the test.
# For further details, you can refer to https://jira.devtools.intel.com/browse/NEXMANAGE-790
fuzztest:
	sudo cp mocks/server-cert.pem /usr/local/share/ca-certificates/server-cert.crt
	sudo update-ca-certificates
	go test ./cmd/platform-update-agent/ -fuzz=FuzzGrpcCommunication -parallel=1 -fuzztime=${FUZZTIME}
	go test ./cmd/platform-update-agent/ -fuzz=FuzzLogic -parallel=1 -fuzztime=${FUZZTIME}

test: common-unit-test

integration_test:
	@echo "no integration tests to run, skipping"

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./cmd/platform-update-agent/...
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover_integration:

cover: cover_unit cover_integration
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > build/artifacts/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/platform-update-agent#g" build/artifacts/coverage/profile 
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > build/artifacts/coverage.xml

package: inbm-package-dist
	mkdir -p $(PUA_PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PUA_PACKAGE_BUILD_DIR)
	cp ../.golangci.yml $(PUA_PACKAGE_BUILD_DIR)
	cp -r ../LICENSES ${PUA_PACKAGE_BUILD_DIR}
	cp ../common.mk $(PUA_PACKAGE_BUILD_DIR)
	sed -i "s#../.golangci.yml#.golangci.yml#" $(PUA_PACKAGE_BUILD_DIR)/common.mk
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PUA_PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PUA_PACKAGE_BUILD_DIR)/Makefile
	cd $(PUA_PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b

package_test:
	echo "platform-update-agent platform-update-agent/updateServiceURL string localhost:12345" | sudo debconf-set-selections
	echo "platform-update-agent platform-update-agent/enaDebURL string files.internal.example.intel.com" | sudo debconf-set-selections
	sudo apt-get install -y ./$(BUILD_DIR)/platform-update-agent_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable platform-update-agent
	sudo systemctl start platform-update-agent
	systemctl is-active platform-update-agent
	sudo systemctl stop platform-update-agent
	sudo systemctl disable platform-update-agent
	sudo apt-get purge -y platform-update-agent

tarball: common-tarball

deb-push: common-deb-push

# INBM Framework Integration
INBM_PACKAGE_DIR := $(BUILD_DIR)/inbm-package

.PHONY: inbm-build inbm-test inbm-clean inbm-build-deb inbm-package-dist inbm-buf-gen inbm-buf-lint

# INBM protobuf generation using buf (consistent with common module)
inbm-buf-lint:
	@echo "---MAKEFILE INBM BUF LINT---"
	cd $(INBM_DIR) && buf --version
	cd $(INBM_DIR) && buf format -w
	cd $(INBM_DIR) && buf lint
	@echo "---END MAKEFILE INBM BUF LINT---"

inbm-buf-gen:
	@echo "---MAKEFILE INBM BUF GENERATE---"
	cd $(INBM_DIR) && buf --version
	cd $(INBM_DIR) && buf generate --path proto/inbd/v1/inbd.proto
	@echo "---END MAKEFILE INBM BUF GENERATE---"

inbm-build: inbm-buf-gen
	@echo "---MAKEFILE INBM BUILD---"
	cd $(INBM_DIR) && CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -buildmode=pie -trimpath -mod=$(GO_MOD_MODE) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w" -o ../$(BUILD_DIR)/inbc cmd/inbc/main.go
	cd $(INBM_DIR) && CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -buildmode=pie -trimpath -mod=$(GO_MOD_MODE) -gcflags="all=-spectre=all" -asmflags="all=-spectre=all" -ldflags="all=-s -w" -o ../$(BUILD_DIR)/inbd cmd/inbd/main.go
	@echo "---END MAKEFILE INBM BUILD---"

inbm-test:
	@echo "---MAKEFILE INBM TEST---"
	cd $(INBM_DIR) && go test ./... -v
	@echo "---END MAKEFILE INBM TEST---"

inbm-clean:
	cd $(INBM_DIR) && go clean -cache -modcache

inbm-build-deb: inbm-build
	@echo "---MAKEFILE INBM BUILD DEB---"
	rm -rf $(INBM_PACKAGE_DIR)
	mkdir -p $(INBM_PACKAGE_DIR)/DEBIAN
	mkdir -p $(INBM_PACKAGE_DIR)/usr/bin
	mkdir -p $(INBM_PACKAGE_DIR)/etc/intel_manageability/public
	mkdir -p $(INBM_PACKAGE_DIR)/etc/apparmor.d
	mkdir -p $(INBM_PACKAGE_DIR)/etc/intel_edge_node/tokens/release-service
	mkdir -p $(INBM_PACKAGE_DIR)/usr/lib/systemd/system
	mkdir -p $(INBM_PACKAGE_DIR)/usr/share
	# Copy binaries
	cp $(BUILD_DIR)/inbc $(INBM_PACKAGE_DIR)/usr/bin/inbc
	cp $(BUILD_DIR)/inbd $(INBM_PACKAGE_DIR)/usr/bin/inbd
	# Create JWT token directory and empty token file
	touch $(INBM_PACKAGE_DIR)/etc/intel_edge_node/tokens/release-service/access_token
	# Copy configuration files from INBM fpm-templates
	cp $(INBM_DIR)/fpm-templates/etc/intel_manageability.conf $(INBM_PACKAGE_DIR)/etc/intel_manageability.conf || true
	cp $(INBM_DIR)/fpm-templates/etc/firmware_tool_info.conf $(INBM_PACKAGE_DIR)/etc/firmware_tool_info.conf || true
	# Copy other template files
	cp $(INBM_DIR)/fpm-templates/etc/apparmor.d/usr.bin.inbd $(INBM_PACKAGE_DIR)/etc/apparmor.d/usr.bin.inbd || true
	cp $(INBM_DIR)/fpm-templates/usr/bin/UpdateFirmwareBlobFwupdtool.sh $(INBM_PACKAGE_DIR)/usr/bin/UpdateFirmwareBlobFwupdtool.sh || true
	cp $(INBM_DIR)/fpm-templates/usr/bin/provision-tc $(INBM_PACKAGE_DIR)/usr/bin/provision-tc || true
	cp $(INBM_DIR)/fpm-templates/usr/lib/systemd/system/inbd.service $(INBM_PACKAGE_DIR)/usr/lib/systemd/system/inbd.service || true
	# Copy schema files
	cp $(INBM_DIR)/fpm-templates/usr/share/inbd_schema.json $(INBM_PACKAGE_DIR)/usr/share/inbd_schema.json || true
	cp $(INBM_DIR)/fpm-templates/usr/share/firmware_tool_config_schema.json $(INBM_PACKAGE_DIR)/usr/share/firmware_tool_config_schema.json || true
	# Copy postinst script
	cp $(INBM_DIR)/fpm-templates/DEBIAN/postinst $(INBM_PACKAGE_DIR)/DEBIAN/postinst || true
	# Create conffiles
	echo "/etc/intel_manageability.conf" > $(INBM_PACKAGE_DIR)/DEBIAN/conffiles
	echo "/etc/firmware_tool_info.conf" >> $(INBM_PACKAGE_DIR)/DEBIAN/conffiles
	echo "/etc/intel_edge_node/tokens/release-service/access_token" >> $(INBM_PACKAGE_DIR)/DEBIAN/conffiles
	# Set permissions
	chmod 755 $(INBM_PACKAGE_DIR)/usr/bin/inbc
	chmod 755 $(INBM_PACKAGE_DIR)/usr/bin/inbd
	chmod 640 $(INBM_PACKAGE_DIR)/etc/intel_manageability.conf || true
	chmod 644 $(INBM_PACKAGE_DIR)/etc/firmware_tool_info.conf || true
	chmod 750 $(INBM_PACKAGE_DIR)/etc/intel_edge_node || true
	chmod 750 $(INBM_PACKAGE_DIR)/etc/intel_edge_node/tokens || true
	chmod 750 $(INBM_PACKAGE_DIR)/etc/intel_edge_node/tokens/release-service || true
	chmod 640 $(INBM_PACKAGE_DIR)/etc/intel_edge_node/tokens/release-service/access_token || true
	chmod 644 $(INBM_PACKAGE_DIR)/usr/share/inbd_schema.json || true
	chmod 644 $(INBM_PACKAGE_DIR)/usr/share/firmware_tool_config_schema.json || true
	chmod 755 $(INBM_PACKAGE_DIR)/usr/bin/UpdateFirmwareBlobFwupdtool.sh || true
	chmod 700 $(INBM_PACKAGE_DIR)/usr/bin/provision-tc || true
	chmod 755 $(INBM_PACKAGE_DIR)/DEBIAN/postinst || true
	# Create control file
	printf "Package: intel-inbm\nVersion: $(INBM_VERSION)\nArchitecture: amd64\nMaintainer: Intel Corporation\nDescription: Intel In-Band Manageability Tools\n This package contains the inbc CLI and inbd daemon for Intel In-Band Manageability.\n" > $(INBM_PACKAGE_DIR)/DEBIAN/control
	# Build the Debian package
	dpkg-deb --build $(INBM_PACKAGE_DIR) $(BUILD_DIR)/intel-inbm.deb
	@echo "---END MAKEFILE INBM BUILD DEB---"

inbm-package-dist: inbm-build-deb
	@echo "---MAKEFILE INBM PACKAGE DIST---"
	# Copy license and legal files from INBM directory
	cp $(INBM_DIR)/LICENSE $(BUILD_DIR)/LICENSE || true
	cp -r $(INBM_DIR)/retain-3rd-party-notices $(BUILD_DIR)/retain-3rd-party-notices || true
	cp $(INBM_DIR)/third-party-programs.txt $(BUILD_DIR)/third-party-programs.txt || true
	# Copy installer scripts from INBM directory
	cp $(INBM_DIR)/installer/install-tc.sh $(BUILD_DIR)/install-tc.sh || true
	cp $(INBM_DIR)/installer/uninstall-tc.sh $(BUILD_DIR)/uninstall-tc.sh || true
	cp $(INBM_DIR)/installer/uninstall-python-tc.sh $(BUILD_DIR)/uninstall-python-tc.sh || true
	@echo "---END MAKEFILE INBM PACKAGE DIST---"