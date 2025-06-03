# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel
VERSION 0.8

LOCALLY
ARG http_proxy=$(echo $http_proxy)
ARG https_proxy=$(echo $https_proxy)
ARG no_proxy=$(echo $no_proxy)
ARG HTTP_PROXY=$(echo $HTTP_PROXY)
ARG HTTPS_PROXY=$(echo $HTTPS_PROXY)
ARG NO_PROXY=$(echo $NO_PROXY)
ARG REGISTRY

FROM ${REGISTRY}golang:1.24.1-alpine3.21
ENV http_proxy=$http_proxy
ENV https_proxy=$https_proxy
ENV no_proxy=$no_proxy
ENV HTTP_PROXY=$HTTP_PROXY
ENV HTTPS_PROXY=$HTTPS_PROXY
ENV NO_PROXY=$NO_PROXY

all:
    BUILD +build
    BUILD +test
    BUILD +lint


fetch-golang:
    RUN apk add curl && curl -fsSLO https://go.dev/dl/go1.24.1.linux-amd64.tar.gz
    SAVE ARTIFACT go1.24.1.linux-amd64.tar.gz

build:
    BUILD +generate-proto
    BUILD +build-inbc
    BUILD +build-inbd

golang-base:
    RUN apk add --no-cache protoc protobuf-dev libprotobuf curl gcc musl-dev && \
        go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 && \
        go install github.com/bufbuild/buf/cmd/buf@v1.50.1 && \
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.7
    WORKDIR /work
    COPY go.mod .
    COPY go.sum .
    RUN go mod download # for caching
    COPY cmd/ ./cmd
    COPY pkg/ ./pkg
    COPY proto/ ./proto
    COPY internal/ ./internal

lint:
    FROM +golang-base
    WORKDIR /work
    RUN --mount=type=cache,target=/root/.cache \
        golangci-lint run ./...
    
test:
    BUILD +run-golang-unit-tests
    BUILD +lint

run-golang-unit-tests:
    FROM +golang-base
    
    # Run tests for all packages, generating coverage for internal/ only
    RUN --mount=type=cache,target=/root/.cache/go-build \
        CGO_ENABLED=1 go test -race -shuffle on -short ./... \
        -coverpkg=./internal/... -coverprofile=cover.out
   
    # Enforce minimum coverage threshold for internal/ directory
    RUN COVERAGE=$(go tool cover -func=cover.out | awk '/total:/ {print $3}' | tr -d '%') && MIN_COVERAGE=63.6 && echo "Total Coverage for internal/: $COVERAGE%" && echo "Minimum Required Coverage: $MIN_COVERAGE%" && awk -v coverage="$COVERAGE" -v min="$MIN_COVERAGE" 'BEGIN {if (coverage < min) {print "Coverage " coverage "% is below " min "%"; exit 1} else {print "Coverage " coverage "% meets the requirement."; exit 0}}'
    
    SAVE ARTIFACT cover.out AS LOCAL build/cover.out
    
generate-proto:
    FROM +golang-base
    COPY ./buf.gen.yaml .
    COPY ./buf.yaml .

    RUN buf generate
    SAVE ARTIFACT proto AS LOCAL ./proto
    SAVE ARTIFACT pkg/api/inbd AS LOCAL ./pkg/api/inbd

build-inbc:
    FROM +golang-base
    ARG version='0.0.0-unknown'
    RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
        go build -trimpath -o build/inbc \
            -ldflags "-s -w -extldflags '-static' -X main.Version=$version" \
            ./cmd/inbc
    SAVE ARTIFACT build/inbc AS LOCAL ./build/inbc

build-inbd:
    FROM +golang-base
    ARG version='0.0.0-unknown'
    RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
        go build -trimpath -o build/inbd \
            -ldflags "-s -w -extldflags '-static' -X main.Version=$version" \
            ./cmd/inbd
    SAVE ARTIFACT build/inbd AS LOCAL ./build/inbd

build-deb:
    BUILD +build
    FROM debian:bullseye
    WORKDIR /package
    RUN mkdir -p DEBIAN usr/bin etc etc/apparmor.d usr/lib/systemd/system usr/share

    # Copy the binaries to the package directory
    COPY build/inbc usr/bin/inbc
    COPY build/inbd usr/bin/inbd

    # Copy the configuration file to the package directory
    COPY fpm-templates/etc/intel_manageability.conf etc/intel_manageability.conf

    # Create the DEBIAN/conffiles file
    RUN echo "/etc/intel_manageability.conf" >> DEBIAN/conffiles
        
    # Set ownership and permissions for the configuration file
    RUN chown root:root etc/intel_manageability.conf
    RUN chmod 640 etc/intel_manageability.conf

    # Copy the schema file to the package directory
    COPY fpm-templates/usr/share/inbd_schema.json usr/share/inbd_schema.json
    
    # Set ownership and permissions for the schema file
    RUN chown root:root usr/share/inbd_schema.json
    RUN chmod 640 usr/share/inbd_schema.json

    # Copy the postinst script to the DEBIAN directory
    COPY fpm-templates/DEBIAN/postinst DEBIAN/postinst
    RUN chmod 755 DEBIAN/postinst
    
    # Copy other files    
    COPY fpm-templates/etc/apparmor.d/usr.bin.inbd etc/apparmor.d/usr.bin.inbd
    COPY fpm-templates/usr/bin/provision-tc usr/bin/provision-tc
    RUN chown root:root usr/bin/provision-tc
    RUN chmod 700 usr/bin/provision-tc
    COPY fpm-templates/usr/lib/systemd/system/inbd.service usr/lib/systemd/system/inbd.service
    
    # Create the control file
    RUN echo "Package: intel-inbm\nVersion: 0.0.0-unknown\nArchitecture: amd64\nMaintainer: Your Name <your-email@example.com>\nDescription: Intel In-Band Manageability Tools\n This package contains the inbc CLI and inbd daemon for Intel In-Band Manageability." > DEBIAN/control
    
    # Build the Debian package
    RUN dpkg-deb --build . /package/intel-inbm.deb
    SAVE ARTIFACT /package/intel-inbm.deb AS LOCAL ./build/intel-inbm.deb

package:
    RUN mkdir -p dist/inbm
    COPY LICENSE dist/inbm/LICENSE
    COPY installer/install-tc.sh dist/inbm/install-tc.sh
    COPY installer/uninstall-tc.sh dist/inbm/uninstall-tc.sh
    COPY build/intel-inbm.deb dist/inbm/intel-inbm.deb

    SAVE ARTIFACT dist/inbm AS LOCAL ./dist/inbm
