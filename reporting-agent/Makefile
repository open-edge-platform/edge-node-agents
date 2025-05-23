# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# reporting-agent Makefile

SHELL = bash -eu -o pipefail

FULL_NAME ?= reporting-agent
VENV_NAME ?= venv_reportingagent
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

.PHONY: test build tarball
all: test build

lint: golint

cover_unit:
	@echo "Running unit tests with coverage..."
	# mkdir -p $(BUILD_DIR)/coverage/unit
	# go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	# go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	# go tool covdata func -i=$(BUILD_DIR)/coverage/unit

cover: cover_unit
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/reporting-agent#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml

rabuild:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	$(GOCMD) build -trimpath -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static" \
	-o $(BUILD_DIR)/$(NAME) cmd/reporting-agent.go

rabuild_with_race:
	CGO_ENABLED=1 go build -race \
	-o $(BUILD_DIR)/$(NAME)_race cmd/reporting-agent.go

rabuild_with_cover:
	CGO_ENABLED=0 go build -cover -coverpkg=./cmd -covermode count \
	-o $(BUILD_DIR)/$(NAME)_cover cmd/reporting-agent.go

build: rabuild

test: common-unit-test

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

tarball:
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ internal/ Makefile VERSION go.mod go.sum ../common.mk $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(FULL_NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(FULL_NAME)-$(PKG_VERSION)

fuzztest: common-fuzztest

integration_test:
	echo "Running integration tests..."

package:
	echo "Packaging..."

deb-push:
	echo "Debian package is not supported"
