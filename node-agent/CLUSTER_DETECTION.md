<!---
  SPDX-FileCopyrightText: (C) 2026 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Node Agent Cluster Detection and Kubeconfig Management

This document describes the cluster detection and kubeconfig management functionality added to the node-agent.

## Overview

The node-agent now has the ability to:
1. **Detect running clusters** on the node (K3s and standard Kubernetes)
2. **Retrieve kubeconfig** from detected clusters
3. **Notify the host manager** about cluster status and kubeconfig availability
4. **Include cluster information** in status reports

## Architecture

### Components Added

1. **`cluster.ClusterDetector`** - Detects running cluster installations
2. **`cluster.KubeconfigManager`** - Manages kubeconfig lifecycle and communication
3. **Enhanced Status Service** - Includes cluster information in status reports
4. **Configuration Options** - Control cluster detection behavior

### Detection Logic

The cluster detector checks for:
- **K3s clusters**: Looks for `/usr/local/bin/k3s` binary and checks systemd service status

When a cluster is detected, it:
- Retrieves the cluster version
- Checks if the cluster service is running
- Locates the kubeconfig file in standard locations
- Validates the kubeconfig content

### Kubeconfig Locations

The system looks for kubeconfig files in these locations:
- **K3s**: `/etc/rancher/k3s/k3s.yaml`

## Configuration

### Node Agent Configuration

Add the following section to your `node-agent.yaml`:

```yaml
cluster:
  detectionEnabled: true      # Enable/disable cluster detection
  detectionInterval: 120s     # How often to check for clusters (default: 2 minutes)
```

### Default Behavior

- **Detection Enabled**: By default, cluster detection is enabled when onboarding is enabled
- **Detection Interval**: 120 seconds (2 minutes)
- **Status Integration**: Cluster information is automatically included in status reports

## Implementation Details

### Cluster Detection Process

1. **Periodic Scanning**: Runs every 2 minutes (configurable)
2. **Service Detection**: Uses `systemctl` to check if cluster services are active
3. **Version Detection**: Runs cluster binaries to get version information
4. **Kubeconfig Validation**: Validates retrieved kubeconfig for required fields

### Status Reporting

Cluster information is included in the node status sent to the host manager:

```
"5 of 6 components running; cluster: k3s v1.28.2+k3s1 (running), kubeconfig: 2048 bytes"
```

If no cluster is detected:
```
"5 of 6 components running; cluster: none detected"
```

### Kubeconfig Management

The kubeconfig manager:
- Tracks kubeconfig changes using SHA256 hashing
- Only notifies the host manager when kubeconfig changes
- Provides methods to clear kubeconfig when clusters are removed


### Key Functions

- `DetectCluster()` - Main cluster detection entry point
- `GetKubeconfig()` - Retrieve kubeconfig content
- `NotifyKubeconfig()` - Send kubeconfig to host manager
- `ValidateKubeconfig()` - Basic kubeconfig validation

## Usage Example

The functionality is automatic once configured. The node-agent will:

1. Start the cluster detection goroutine
2. Scan for clusters every 2 minutes
3. Report cluster status in heartbeats
4. Notify about kubeconfig changes
