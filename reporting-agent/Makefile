# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# reporting-agent Makefile

# Shell config variable
SHELL := bash -eu -o pipefail

PROJECT_NAME        := reporting-agent
BUILD_DIR           ?= ./build
PACKAGE_BUILD_DIR   ?= $(BUILD_DIR)/package

VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
ifneq (,$(findstring dev,$(VERSION)))
	PKG_VERSION = $(VERSION)-$(COMMIT)
else
	PKG_VERSION = $(VERSION)
endif

TARBALL_DIR     := $(BUILD_DIR)/$(PROJECT_NAME)-$(PKG_VERSION)
REGISTRY        := 080137407410.dkr.ecr.us-west-2.amazonaws.com

GOCMD           := CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go
GOCMD_TEST      := CGO_ENABLED=1 GOARCH=amd64 GOOS=linux go
GOEXTRAFLAGS    :=-trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X main.version=$(shell cat ./VERSION)"

# Include shared makefile, if it exists
ifneq ("$(wildcard ../common.mk)","")
	include ../common.mk
else ifneq ("$(wildcard ./common.mk)","")
	include ./common.mk
endif

.DEFAULT_GOAL   := help
.PHONY: build

YAMLLINT_CONFIG_DATA  ?= {extends: default, rules: {empty-lines: {max-end: 1}, line-length: {max: 260}, braces: {min-spaces-inside: 0, max-spaces-inside: 1}, brackets: {min-spaces-inside: 0, max-spaces-inside: 1}, document-start: disable}, ignore: [$(YAMLLINT_IGNORE)]}
YAMLLINT_IGNORE       ?= .golangci.yml
SH_FILES_TO_LINT      := $(shell find . -type f -name '*.sh' -print )

## Mandatory Targets start
dependency-check-ci: ## Unsupported target
	@echo '"make $@" is unsupported'

build: ## Builds agent
	@echo "---MAKEFILE BUILD---"
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME) ./cmd/$(PROJECT_NAME).go
	@echo "---END MAKEFILE BUILD---"

lint: lint-go lint-markdown lint-yaml lint-json lint-shell lint-license ## Runs all linters

test: ## Runs tests
	@echo "---MAKEFILE TEST---"
	mkdir -p $(BUILD_DIR)/coverage/unit
	$(GOCMD_TEST) test -v ./... --race -cover -covermode atomic -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit
	go tool covdata textfmt -i=$(BUILD_DIR)/coverage/unit -o $(BUILD_DIR)/coverage/profile
	go tool cover -html $(BUILD_DIR)/coverage/profile -o $(BUILD_DIR)/coverage/coverage.html
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	cat $(BUILD_DIR)/coverage/unit/unit.out | go-junit-report -set-exit-code > $(BUILD_DIR)/report.xml
	sed -i "s#$(shell pwd)#github.com/open-edge-platform/edge-node-agents/reporting-agent#g" $(BUILD_DIR)/coverage/profile
	gocover-cobertura < $(BUILD_DIR)/coverage/profile > $(BUILD_DIR)/coverage.xml
	@echo "---END MAKEFILE TEST---"

docker-build: ## Unsupported target
	@echo '"make $@" is unsupported'

helm-build: ## Unsupported target
	@echo '"make $@" is unsupported'

docker-push: ## Unsupported target
	@echo '"make $@" is unsupported'

helm-push: ## Unsupported target
	@echo '"make $@" is unsupported'

tarball:
	@echo "---MAKEFILE TARBALL---"
	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ config/ internal/ config.yaml go.mod go.sum Makefile VERSION ../common.mk $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(PROJECT_NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(PROJECT_NAME)-$(PKG_VERSION)
	@echo "---END MAKEFILE TARBALL---"

fuzztest: # common-fuzztest #TODO: enable later

integration_test:
	echo "Running integration tests..."

package:
	echo "Packaging..."

deb-push:
	echo "Debian package is not supported"
## Mandatory Targets end

## Helper Targets start
all: clean build lint test ## Runs clean, build, lint, test targets

clean: ## Deletes build directory
	@echo "---MAKEFILE CLEAN---"
	rm -rf $(BUILD_DIR)
	@echo "---END MAKEFILE CLEAN---"

lint-go: ## Runs linters for golang source code files
	@echo "---MAKEFILE LINT-GO---"
	golangci-lint -v run
	@echo "---END MAKEFILE LINT-GO---"

install-tools: ## Installs tools required for the project, requires installed: asdf
	@echo "---MAKEFILE INSTALL-TOOLS---"
	./scripts/installTools.sh ./.tool-versions
	@echo "---END MAKEFILE INSTALL-TOOLS---"

lint-markdown: ## Runs linter for markdown files
	@echo "---MAKEFILE LINT-MARKDOWN---"
	markdownlint-cli2 --config ./.markdownlint.yml '**/*.md' "!.github" "!vendor"
	@echo "---END MAKEFILE LINT-MARKDOWN---"

lint-yaml: ## Runs linter for for yaml files
	@echo "---MAKEFILE LINT-YAML---"
	yamllint -v
	yamllint -f parsable -d '$(YAMLLINT_CONFIG_DATA)' .
	@echo "---END MAKEFILE LINT-YAML---"

lint-json: ## Runs linter for json files
	@echo "---MAKEFILE LINT-JSON---"
	./scripts/lintJsons.sh
	@echo "---END MAKEFILE LINT-JSON---"

lint-shell: ## Runs linter for shell scripts
	@echo "---MAKEFILE LINT-SHELL---"
	@if [ -n "$(SH_FILES_TO_LINT)" ]; then \
		shellcheck --version; \
		shellcheck $(SH_FILES_TO_LINT); \
	else \
		echo "No shell files to lint."; \
	fi
	@echo "---END MAKEFILE LINT-SHELL---"

lint-license: ## Runs license check
	@echo "---MAKEFILE LINT-LICENSE---"
	reuse --version
	reuse --root . lint
	@echo "---END MAKEFILE LINT-LICENSE---"
## Helper Targets end
