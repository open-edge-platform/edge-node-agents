# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# hardware-discovery-agent Makefile

SHELL = bash -eu -o pipefail

NAME ?= hd-agent
FULL_NAME ?= hardware-discovery-agent
GO_MOD ?= readonly
VENV_NAME ?= venv_hdagent
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

# Include shared makefile, if it exists
ifneq ("$(wildcard ../common.mk)","")
	include ../common.mk
else ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
endif

.PHONY: test fuzztest build install tarball $(BUILD_DIR)/hdagent-linux-amd64 $(BUILD_DIR)/hdagent-linux-arm64
all: test build $(BUILD_DIR)/hdagent-linux-amd64 $(BUILD_DIR)/hdagent-linux-arm64

lint: golint

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover_integration: hdabuild_with_cover hmmbuild ssmbuild
	mkdir -p $(BUILD_DIR)/coverage/integration
	GOCOVERDIR=$(shell pwd)/$(BUILD_DIR)/coverage/integration \
	HDA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hd-agent_cover \
	HMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hostmgr-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/test/logLevel_info_good_mock_address_config.yaml \
	ERROR_WRONG_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_error_wrong_mock_address_config.yaml \
	DEBUG_GOOD_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_debug_good_mock_address_config.yaml \
	INFO_WRONG_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_info_wrong_mock_address_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy.crt \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy.key \
	HDA_VERSION="Hardware Discovery Agent v$(VERSION)-$(COMMIT)" \
	go test -v ./cmd/hdagent/hdagent_test.go | tee $(BUILD_DIR)/coverage/integration/integration.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/integration
	go tool covdata func -i=$(BUILD_DIR)/coverage/integration

cover: cover_unit cover_integration
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit,$(BUILD_DIR)/coverage/integration
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml

hdabuild:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	$(GOCMD) build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME) cmd/hdagent/hdagent.go

hdabuild_with_race:
	CGO_ENABLED=1 go build -race -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_race cmd/hdagent/hdagent.go

hdabuild_with_cover:
	CGO_ENABLED=0 go build -cover -coverpkg=./cmd/hdagent -covermode count -ldflags=" \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.version=$(VERSION) \
	-X github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/hdagent/hdagent.go

hmmbuild:
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT)" \
	-o $(BUILD_DIR)/hostmgr-mock cmd/hostmgr-mock/hostmgr-mock.go
	@echo "---END MAKEFILE Build---"


build: hdabuild hmmbuild ssmbuild

build-linux: $(BUILD_DIR)/hdagent-linux-amd64 $(BUILD_DIR)/hdagent-linux-arm64

test: common-unit-test

integration_test: hdabuild_with_race hmmbuild ssmbuild
	sudo cp test/_dummy.crt /usr/local/share/ca-certificates/_dummy.crt
	sudo update-ca-certificates
	HDA_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hd-agent_race \
	HMM_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/hostmgr-mock \
	SS_BINARY_PATH=$(shell pwd)/$(BUILD_DIR)/status-server-mock \
	WORKING_CONFIG_PATH=$(shell pwd)/test/logLevel_info_good_mock_address_config.yaml \
	ERROR_WRONG_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_error_wrong_mock_address_config.yaml \
	DEBUG_GOOD_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_debug_good_mock_address_config.yaml \
	INFO_WRONG_ADDRESS_CONFIG_PATH=$(shell pwd)/test/logLevel_info_wrong_mock_address_config.yaml \
	TEST_CERT_PATH=$(shell pwd)/test/_dummy.crt \
	TEST_KEY_PATH=$(shell pwd)/test/_dummy.key \
	HDA_VERSION="Hardware Discovery Agent v$(VERSION)-$(COMMIT)" \
	GORACE="log_path=stdout" \
	go test -v ./cmd/hdagent/hdagent_test.go

fuzztest: common-fuzztest

install:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

$(BUILD_DIR)/hdagent-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOCMD) build -o $@ cmd/hdagent/hdagent.go

$(BUILD_DIR)/hdagent-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOCMD) build -o $@ cmd/hdagent/hdagent.go

$(VENV_NAME): requirements.txt
	python3 -m venv $@;\
	. ./$@/bin/activate ; set -u ;\
	python -m pip install --upgrade pip;\
	python -m pip install -r requirements.txt
	echo "To enter virtualenv, run 'source $@/bin/activate'"

yamllint: $(VENV_NAME)
	. ./$</bin/activate ; set -u ;\
  	yamllint .

clean:
	rm -rf build

package:
	mkdir -p $(PACKAGE_BUILD_DIR)
	cp -r $(shell ls . | grep -v build*) $(PACKAGE_BUILD_DIR)
	cp -r ../LICENSES ${PACKAGE_BUILD_DIR}
	cp ../common.mk $(PACKAGE_BUILD_DIR)
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_BUILD_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_BUILD_DIR)/Makefile
	cd $(PACKAGE_BUILD_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b

package_test:
	echo "hardware-discovery-agent hardware-discovery-agent/onboarding.serviceURL string localhost:11223" | sudo debconf-set-selections
	sudo apt-get install -y ./$(BUILD_DIR)/hardware-discovery-agent_$(PKG_VERSION)_amd64.deb
	sudo systemctl enable hardware-discovery-agent
	sudo systemctl start hardware-discovery-agent
	systemctl is-active hardware-discovery-agent
	sudo systemctl disable hardware-discovery-agent
	sudo apt-get purge -y hardware-discovery-agent

tarball:
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ config/ debian/copyright internal/ Makefile VERSION go.mod go.sum ../common.mk $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(FULL_NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(FULL_NAME)-$(PKG_VERSION)

deb-push:
	if [ -z "$$(cat VERSION | grep 'dev')" ]; then \
		echo "Uploading artifacts..."; \
		cd build/artifacts; \
		for DEB_PKG in *.deb; do \
			PKG_VER=$$(dpkg-deb -f "$${DEB_PKG}" Version); \
			PKG_NAME=$$(dpkg-deb -f "$${DEB_PKG}" Package); \
			REPOSITORY=en/deb/$${PKG_NAME}; \
			oras push $${REGISTRY}/edge-orch/$${REPOSITORY}:$${PKG_VER} \
			--artifact-type application/vnd.intel.orch.deb ./$${DEB_PKG}; \
		done; \
		cd -; \
	fi
