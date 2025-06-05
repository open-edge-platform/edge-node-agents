# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

NAME ?= platform-update-agent
BUILD_DIR ?= build/artifacts
INBM_VERSION ?= 4.2.8.6
INBM_DIR_NAME ?= inbm
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

lint: golint

puainstall:
	install -d $(DESTDIR)$(PREFIX)/bin
	install $(BUILD_DIR)/$(NAME) $(DESTDIR)$(PREFIX)/bin

inbmdownload:
	@echo "---MAKEFILE INBM DOWNLOAD---"
	mkdir -p ${BUILD_DIR}
	rm -rf intel-inb-manageability
	git clone https://github.com/intel/intel-inb-manageability.git
	(cd intel-inb-manageability && git checkout tags/v$(INBM_VERSION))
	intel-inb-manageability/build.sh --build-windows=false --build-check=false
	cp intel-inb-manageability/dist/inbm/* ./$(BUILD_DIR)
	tar zxvf ./$(BUILD_DIR)/Intel-Manageability.preview.tar.gz --directory ./$(BUILD_DIR)
	@echo "---END MAKEFILE INBM DOWNLOAD---"

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

build: puabuild mmbuild

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

package: inbmdownload
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