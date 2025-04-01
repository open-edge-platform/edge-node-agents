<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Edge Node Agents Changelog

## Cluster Agent Changelog

### 1.5.13
- Relax status check to 2x of interval

### 1.5.10
- Fix Makefile import tarball

### 1.5.9
- Update app armor profile: allow "sed" command to the cluster-agent

### 1.5.8
- Update app armor profile: allow RW to `/etc/systemd/system/rke2-server.service.d/override.conf` & `/etc/systemd/system/rke2-agent.service.d/override.conf`

### 1.5.7
- Create configuration override for rke2 systemd services to use `/etc/environment` as `EnvironmentFile`. This is needed to propagade OS proxy configuration to rke2 cluster.

### 1.5.6
- Modules path changed to 'open-edge-platform`

### 1.5.5
- Update CO API version to 0.3.10

### 1.5.4
- Update app armor profile: allow RW to `/etc/default/rke2-agent` & `/etc/default/rke2-server`

### 1.5.3
- Update dependency versions

### 1.5.2

### 1.5.1
- Cluster Agent status client

### 1.5.0
- Initial cluster agent release
- Dependency import updates
- Adjust AppArmor profile for RKE2 script install

## Common Package Changelog

### 1.6.5
- Add proto validator

### 1.6.4
- Create status server mock

### 1.6.3

### 1.6.2
- Update datatype of wrapper for getStatusInterval

### 1.6.1
- Add status interval to defintion of SendStatus API

### 1.6.0
- Initial release of common agent code
- Bump golang.org/x/net from 0.23.0 to 0.33.0
- Add common client for Sendstatus API

## Hardware Discovery Agent Changelog

### 1.5.11
- Relax status check to 2x of interval

### 1.5.5
- Fix `make tarball` command for agent
- Fix Makefile import tarball

### 1.5.4
- Fix flickering status

### 1.5.3
- Update module paths

### 1.5.2
- Update dependency versions
- Send status to Node Agent
- Fix CPU Topology detection

### 1.5.1

### 1.5.0
- Initial hardware discovery agent release
- Dependency import updates

## Node Agent Changelog

### 1.5.12
- Update jwt-go dependency version to 4.5.2

### 1.5.11
- Fix Makefile import tarball

### 1.5.10
- Support nw endpoint status using oras client

### 1.5.9
- Fix RSType config

### 1.5.8
- Update module paths

### 1.5.7
- Add support for no-auth release service to node agent

### 1.5.6
- Fix status socket persistence across restarts
- Update app armour profile for connect agent

### 1.5.5
- Add folder for attestation manager service
- Fix agent name in default config

### 1.5.4
- Add proto validation

### 1.5.3
- Update dependency versions
- Support to monitor network endpoints for status

### 1.5.2

### 1.5.1
- Fix AppArmor profile
- Add status service termination

### 1.5.0
- Initial node agent release
- Add status service implementation
- Dependency import updates

## Platform Observability Agent Changelog

### 1.7.4
- Fix Makefile import tarball

### 1.7.3
- Update dependency versions to latest

### 1.7.2
- Fix tarball build

### 1.7.1

### 1.7.0
- Initial platform observability agent release

## Platform Telemetry Agent Changelog

### 1.2.11
- Relax status check to 2x of interval

### 1.2.5
- Fix `make tarball` command for agent
- Fix Makefile import tarball

### 1.2.3
- Send ststus to Node Agent

### 1.2.2
- Update dependency versions

### 1.2.1

### 1.2.0
- Initial platform telemetry agent release
- Dependency import updates

## Platform Update Agent Changelog

### 1.3.9
- Relax status check to 2x of interval

### 1.3.6
- Fix `make tarball` command for agent
- Fix Makefile import tarball

### 1.3.3
- Send ststus to Node Agent

### 1.3.2
- Update dependency versions

### 1.3.1
- README added for platform update agent

### 1.3.0
- Initial platform update agent release
- Dependency import updates
