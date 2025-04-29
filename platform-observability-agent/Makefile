# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL=/bin/bash -o pipefail
BUILD_DIR ?= $(shell pwd)/build/artifacts
NAME=platform-observability-agent
PACKAGE_DIR ?= $(BUILD_DIR)/package
INSTALL_DIR ?= $(BUILD_DIR)/install
PKG_VERSION := $(shell if grep -q dev VERSION; then echo $$(cat VERSION)-$$(git rev-parse --short HEAD); else cat VERSION; fi)
ASSETS_DIR ?= assets
FB_VERSION = 3.2.9
FB_DIR = fluent-bit_$(FB_VERSION)
TELEGRAF_VERSION = 1.34.0-1
TELEGRAF_DIR = telegraf_$(TELEGRAF_VERSION)
OTELCOL_VERSION = 0.122.1
OTELCOL_DIR = otelcol_$(OTELCOL_VERSION)


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

.PHONY: clean build lint test package package_test package_depends setup_fb_repo download_fb tarball

clean:
	rm -rf build

build:
	@echo "no binaries to build, skipping"

lint:
	@echo "skipping linter"

test:
	@echo "no unit tests to run, skipping"

integration_test:
	@echo "no integration tests to run, skipping"

fuzztest:
	@echo "no fuzz tests to run, skipping"

download_fb:
	mkdir -p $(ASSETS_DIR)
	cd $(ASSETS_DIR); wget https://packages.fluentbit.io/ubuntu/jammy/pool/main/f/fluent-bit/fluent-bit_$(FB_VERSION)_amd64.deb

unpack_fb:
	rm -f -r $(BUILD_DIR)/$(FB_DIR)
	mkdir -p $(BUILD_DIR)/$(FB_DIR)
	dpkg-deb -R $(ASSETS_DIR)/fluent-bit_$(FB_VERSION)_amd64.deb $(BUILD_DIR)/$(FB_DIR)

download_telegraf:
	mkdir -p $(ASSETS_DIR)
	cd $(ASSETS_DIR); wget https://dl.influxdata.com/telegraf/releases/telegraf_$(TELEGRAF_VERSION)_amd64.deb

unpack_telegraf:
	rm -f -r $(BUILD_DIR)/$(TELEGRAF_DIR)
	mkdir -p $(BUILD_DIR)/$(TELEGRAF_DIR)
	dpkg-deb -R $(ASSETS_DIR)/telegraf_$(TELEGRAF_VERSION)_amd64.deb $(BUILD_DIR)/$(TELEGRAF_DIR)

# https://opentelemetry.io/docs/collector/installation/#deb-installation
download_otelcol:
	mkdir -p $(ASSETS_DIR)
	cd $(ASSETS_DIR); wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v$(OTELCOL_VERSION)/otelcol-contrib_$(OTELCOL_VERSION)_linux_amd64.deb

unpack_otelcol:
	rm -f -r $(BUILD_DIR)/$(OTELCOL_DIR)
	mkdir -p $(BUILD_DIR)/$(OTELCOL_DIR)
	dpkg-deb -R $(ASSETS_DIR)/otelcol-contrib_$(OTELCOL_VERSION)_linux_amd64.deb $(BUILD_DIR)/$(OTELCOL_DIR)

download_deps: download_fb download_telegraf download_otelcol

unpack_deps: unpack_fb unpack_telegraf unpack_otelcol

install: unpack_fb unpack_telegraf unpack_otelcol
	install -D $(BUILD_DIR)/$(FB_DIR)/lib/fluent-bit/libfluent-bit.so $(INSTALL_DIR)/lib/libfluent-bit.so
	install -D $(BUILD_DIR)/$(FB_DIR)/opt/fluent-bit/bin/fluent-bit $(INSTALL_DIR)/opt/fluent-bit/bin/fluent-bit
	install -D $(BUILD_DIR)/$(FB_DIR)/opt/fluent-bit/bin/fluent-bit $(INSTALL_DIR)/opt/health-check/bin/fluent-bit
	install -D $(BUILD_DIR)/$(TELEGRAF_DIR)/usr/bin/telegraf $(INSTALL_DIR)/opt/telegraf/bin/telegraf
	install -D $(BUILD_DIR)/$(OTELCOL_DIR)/usr/bin/otelcol-contrib $(INSTALL_DIR)/opt/otelcol/bin/otelcol

package_depends:
	sudo apt-get update
	sudo apt-get -y install devscripts debhelper libpq5

package: package_depends download_deps
	mkdir -p $(PACKAGE_DIR)
	cp -r assets configs scripts debian Makefile VERSION $(PACKAGE_DIR)
	cp -r ../LICENSES ${PACKAGE_DIR}
	git clone https://github.com/intel/perfmon.git $(PACKAGE_DIR)/perfmon
	cp ../common.mk $(PACKAGE_DIR)
	sed -i "s#VERSION#$(PKG_VERSION)#" $(PACKAGE_DIR)/debian/changelog
	sed -i "s#../common.mk#common.mk#" $(PACKAGE_DIR)/Makefile
	cd $(PACKAGE_DIR); debuild --preserve-env --preserve-envvar PATH -us -uc -b
	rm -rf $(ASSETS_DIR)

package_test:
	echo "platform-observability-agent platform-observability-agent/host string testhost.com" | sudo debconf-set-selections
	echo "platform-observability-agent platform-observability-agent/port string 553" | sudo debconf-set-selections
	echo "platform-observability-agent platform-observability-agent/metrics-host string testhost.com" | sudo debconf-set-selections
	echo "platform-observability-agent platform-observability-agent/metrics-port string 553" | sudo debconf-set-selections
	sudo apt-get install -y $(BUILD_DIR)/platform-observability-agent_*_amd64.deb
	sudo cp test/fluent-bit-test.conf /etc/fluent-bit/fluent-bit.conf
	sudo cp test/health-check-test.conf /etc/health-check/health-check.conf
	sudo cp test/otelcol-test.yaml /etc/otelcol/otelcol.yaml
	sudo systemctl enable platform-observability-logging platform-observability-health-check platform-observability-metrics platform-observability-collector
	sudo systemctl start platform-observability-logging platform-observability-health-check platform-observability-metrics platform-observability-collector
	systemctl is-active platform-observability-logging
	systemctl is-active platform-observability-health-check
	systemctl is-active platform-observability-metrics
	systemctl is-active platform-observability-collector
	sudo systemctl disable platform-observability-logging platform-observability-health-check platform-observability-metrics platform-observability-collector
	sudo apt-get purge -y platform-observability-agent

tarball:
	mkdir -p $(TARBALL_DIR)
	cp -r configs/ debian/copyright scripts/ Makefile VERSION $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)

deb-push: common-deb-push