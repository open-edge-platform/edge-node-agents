FROM golang:1.24 AS builder
ARG TARGETARCH

WORKDIR /workspace
COPY ../go.mod go.mod
COPY ../go.sum go.sum
COPY ../cmd/ cmd
COPY ../configs/ configs
COPY ../info/ info
COPY ../internal/ internal
COPY ../debian/ debian
COPY ../vendor vendor

RUN CGO_ENABLED=0 GOARCH=$TARGETARCH GOOS=linux GO_VENDOR=true go build -o cluster-agent cmd/cluster-agent/cluster-agent.go

FROM ubuntu:22.04

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY

ENV http_proxy=${HTTP_PROXY}
ENV https_proxy=${HTTPS_PROXY}
ENV no_proxy=${NO_PROXY}


RUN apt-get update && \
    apt-get install -y systemd jq iptables apparmor-utils curl sudo wget lsb-release ufw apt-utils lxc make git openssl gettext-base

WORKDIR /etc/enic

COPY enic/environment /etc/environment.template
COPY enic/cluster-agent.yaml /etc/enic/
COPY enic/sample-cloud-init.sh /etc/enic/
COPY --from=builder /workspace/cluster-agent /etc/enic/
COPY enic/cluster-agent.service /etc/systemd/system

RUN envsubst < /etc/environment.template > /etc/environment

COPY enic/proxy.conf /etc/systemd/system/containerd.service.d/proxy.conf.template
COPY enic/proxy.conf /etc/systemd/system/docker.service.d/proxy.conf.template
COPY enic/rke2-server /etc/default/rke2-server.template

RUN envsubst < /etc/systemd/system/containerd.service.d/proxy.conf.template > /etc/systemd/system/containerd.service.d/proxy.conf
RUN envsubst < /etc/systemd/system/docker.service.d/proxy.conf.template > /etc/systemd/system/docker.service.d/proxy.conf
RUN envsubst < /etc/default/rke2-server.template > /etc/default/rke2-server

RUN systemctl enable cluster-agent
