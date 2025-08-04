<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Edge Node Agents Changelog

## Cluster Agent Changelog

### 1.7.4
- Update AppArmor profile for K3s applications

### 1.7.3
- Add retry/backoff to status call

### 1.7.2
- Update AppArmor profile to allow base64 exec

### 1.7.1
- Remove dependency on caddy service for rancher

### 1.7.0
- Fix readiness reporting to NA during cluster install

### 1.6.1
- Update common to 1.6.8

### 1.6.0
- Initial cluster agent release
- Dependency import updates

## Common Package Changelog

### 1.6.8
- Fix Status client to use option WithNoProxy

### 1.6.7
- Initial release of common agent code
- Dependency import updates

## Hardware Discovery Agent Changelog

### 1.7.2
- Update aa armor for GPU on NUCs

### 1.7.1
- Add retry/backoff to status call

### 1.6.0
- Initial hardware discovery agent release
- Dependency import updates

## Node Agent Changelog

### 1.7.2
- Drop TLS on caddy internal endpoint

### 1.7.1
- Error on RS token if not HTTP OK

### 1.7.0
- Update release service token check for anonymous token

### 1.6.2
- Update common to 1.6.8

### 1.6.1
- Update postinst to fix permission on keys

### 1.6.0
- Initial node agent release
- Dependency import updates

## Platform Observability Agent Changelog

### 1.9.0
- Add collection of cloud-init service logs

### 1.8.1
- Update log and metrics service to start after collector service

### 1.8.0
- Initial platform observability agent release

## Platform Telemetry Agent Changelog

### 1.4.0
- Add retry/backoff to status call

### 1.3.1
- Update common to 1.6.8

### 1.3.0
- Initial platform telemetry agent release
- Dependency import updates

## Platform Update Agent Changelog

### 1.5.2
- Add retry/backoff to status call

### 1.5.0
- Downgrade internal caddy endpoint to no TLS

### 1.4.2
- Update INBM to 4.2.8.6

### 1.4.1
- Update common to 1.6.8

### 1.4.0
- Initial platform update agent release
- Dependency import updates

## Platform Manageability Agent Changelog

### 0.1.4
- Fix heartbeat to Noade Agent

### 0.1.0
- Initial platform manageability agent
