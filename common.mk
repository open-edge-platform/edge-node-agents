# common.mk - common targets for Edge Node Agents repos

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Makefile Style Guide:
# - Help will be generated from @# Help: comments in the beggining of each target
# - Use smooth parens $() for variables over curly brackets ${} for consistency
# - Continuation lines (after an \ on previous line) should start with spaces
#   not tabs - this will cause editor highligting to point out editing mistakes
# - When creating targets that run a lint or similar testing tool, print the
#   tool version first so that issues with versions in CI or other remote
#   environments can be caught

#### Go Targets ####

GOCMD := go

#### Lint targets ####

golint:
	@echo "---MAKEFILE GOLINT---"
	golangci-lint run ./...  --config ../.golangci.yml --timeout 5m
	@echo "---END MAKEFILE GOLINT---"

#### Licensing targets ####

license:
	reuse --version;\
	reuse --root . lint

#### Unit test targets ####

common-unit-test:
	@echo "---MAKEFILE TEST---"
	$(GOCMD) test ./internal/...  -race
	@echo "---END MAKEFILE TEST---"

#### Fuzzing test targets ####

FUZZ_TIME ?= 60s
SCRIPTS_DIR := ../common/ci_scripts

common-fuzztest:
	bash -c '$(SCRIPTS_DIR)/fuzz_test.sh "internal" $(FUZZ_TIME); EXIT_STATUS=$$?; if [ $$EXIT_STATUS -ne 0 ]; then exit $$EXIT_STATUS; fi'

#### Tarball targets ####

common-tarball:
	@echo "---MAKEFILE TARBALL---"

	mkdir -p $(TARBALL_DIR)
	cp -r cmd/ configs/ debian/copyright info/ internal/ Makefile VERSION go.mod go.sum ../common.mk $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	cd $(TARBALL_DIR) && go mod tidy && go mod vendor
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)

	@echo "---END MAKEFILE TARBALL---"

ssmbuild:
	@echo "---MAKEFILE BUILD---"
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -ldflags="-s -w -extldflags=-static" \
	-o $(BUILD_DIR)/status-server-mock cmd/status-server-mock/status-server-mock.go
	@echo "---END MAKEFILE Build---"

#### Help Target ####

# Precede targets with a comment that starts with # Help: to provide a description
# Example :
# # Help:  Execute my target
# my-target:
help:
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
	| awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
	| sort \
	| egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
	| xargs -I _ sh -c 'printf "%-20s " _; grep -B 1 "^_" Makefile | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'

#### Build debian packages ####

common-deb-push:
	if [ -z "$$(cat VERSION | grep 'dev')" ]; then \
		set -e; \
		echo "Uploading artifacts..."; \
		cd build/artifacts; \
		for DEB_PKG in *.deb; do \
			PKG_VER=$$(dpkg-deb -f "$${DEB_PKG}" Version); \
			PKG_NAME=$$(dpkg-deb -f "$${DEB_PKG}" Package); \
			REPOSITORY=en/deb/$${PKG_NAME}; \
			URL=$(REGISTRY)/edge-orch/$${REPOSITORY}:$${PKG_VER}; \
			echo "Pushing to URL: $${URL}"; \
			oras push $${URL} \
			--artifact-type application/vnd.intel.orch.deb ./$${DEB_PKG}; \
		done; \
		cd -; \
	fi
