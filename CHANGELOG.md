<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Edge Node Agents Changelog

## Cluster Agent Changelog

### 1.8.1
- Update AppArmor profile to allow environment.d read/write

### 1.8.0
- Update AppArmor profile to allow netdev creation
- Update AppArmor and sudoers to allow k3s uninstallation

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

### 1.8.1
- Relax system boot detection

### 1.8.0
- Send no error during initial 5 minutes for agent status

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

### 1.10.0
- Fix syslog parsing for health check

### 1.9.0
- Add collection of cloud-init service logs

### 1.8.1
- Update log and metrics service to start after collector service

### 1.8.0
- Initial platform observability agent release

## Platform Telemetry Agent Changelog

### 1.5.1
- Revome RKE2 references from the telemetry agent config

### 1.5.0
- cleanup old inbm entries and add new inbd(in-band daemon) entry

### 1.4.0
- Add retry/backoff to status call

### 1.3.1
- Update common to 1.6.8

### 1.3.0
- Initial platform telemetry agent release
- Dependency import updates

## Platform Update Agent Changelog

### 1.6.0
- Fix kernel parameter update

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

### 0.1.7
- Fix status reporting when vPRO disabled

### 0.1.6
- Add fuzz tests

### 0.1.5
- Fix heartbeat to Node Agent

### 0.1.4
- Add AppArmor profile

### 0.1.3
- Load mei module
- Start lms service

### 0.1.2
- Enable gRPC communication with DM manager
- Implement API calls

### 0.1.1
- Add PMA to Edge Node manifest
- Update CI for PMA
- Add sudoers file
- Add agent status reporting
- Fix Debian package installation

### 0.1.0
- Initial platform manageability agent
