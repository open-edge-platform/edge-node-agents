# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
# test
.DEFAULT_GOAL := help

SCRIPTS_DIR := ./ci_scripts
BUILD_DIR ?= build/artifacts
APIPKG_DIR := pkg/api

# Include shared makefile
include ../common.mk

all:
	lint test

lint: golint buf-lint

clean:
	rm -rf build

build:
	@echo "no binaries to build, skipping"

integration_test:
	@echo "no integration tests to run, skipping"

fuzztest:
	@echo "no fuzz tests to run, skipping"

package:
	@echo "no package to build, skipping"

test:
	@echo "---MAKEFILE TEST---"
	go test ./...  -race
	@echo "---END MAKEFILE TEST---"

cover:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./pkg/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/common#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml

#### Protobuf code generation tooling ###

# Help: Lint and format protobuf files
buf-lint:
	buf --version
	buf format -d --exit-code
	buf lint

# Help: Compile protoc files into code
buf-gen:
	buf --version ;\
	buf generate
