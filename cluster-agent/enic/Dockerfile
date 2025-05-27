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

# Install k3s package to enable air gap installs
RUN mkdir -p /var/lib/rancher/k3s/agent/images/
RUN curl -L -o /var/lib/rancher/k3s/agent/images/k3s-airgap-images-amd64.tar.zst "https://github.com/k3s-io/k3s/releases/download/v1.33.0%2Bk3s1/k3s-airgap-images-amd64.tar.zst"
RUN curl -L -o /usr/local/bin/k3s https://github.com/k3s-io/k3s/releases/download/v1.33.0%2Bk3s1/k3s && chmod +x /usr/local/bin/k3s
RUN curl -L -o /opt/install.sh https://get.k3s.io && chmod +x /opt/install.sh

RUN systemctl enable cluster-agent
