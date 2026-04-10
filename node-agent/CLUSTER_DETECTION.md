<!---
  SPDX-FileCopyrightText: (C) 2026 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Node Agent Cluster Detection and Kubeconfig Management

This document describes the cluster detection and kubeconfig management functionality added to the node-agent.

## Overview

The node-agent now has the ability to:
1. **Detect running clusters** on the node (K3s, RKE2, and other supported cluster types)
2. **Retrieve kubeconfig** from detected clusters
3. **Send dedicated cluster status updates** to the host manager when kubeconfig changes
4. **Manage kubeconfig lifecycle** independently of node status reporting

## Architecture

### Components Added

1. **`cluster.ClusterDetector`** - Detects running cluster installations
2. **`cluster.KubeconfigManager`** - Manages kubeconfig lifecycle and dedicated cluster status communication
3. **Dedicated UpdateClusterStatus API** - Separate gRPC call for cluster status updates
4. **Configuration Options** - Control cluster detection behavior and cluster type priorities

### Detection Logic

The cluster detector supports multiple cluster types and checks for:
- **K3s clusters**: Configurable binary path (defaults to `/usr/local/bin/k3s`), checks systemd service status
- **RKE2 clusters**: Configurable binary path (defaults to `/usr/local/bin/rke2`), checks systemd service status
- **Priority-based detection**: First configured cluster type found takes priority

When a cluster is detected, it:
- Retrieves the cluster version
- Checks if the cluster service is running
- Locates the kubeconfig file in standard locations
- Validates the kubeconfig content
- **Sends dedicated cluster status updates** to the host manager via `UpdateClusterStatus` API

The system looks for kubeconfig files in these locations:
- **K3s**: `/etc/rancher/k3s/k3s.yaml`
- **RKE2**: `/etc/rancher/rke2/rke2.yaml`

## Configuration

### Node Agent Configuration

Add the following section to your `node-agent.yaml`:

```yaml
cluster:
  detectionEnabled: true      # Enable/disable cluster detection
  detectionInterval: 120s     # How often to check for clusters (default: 2 minutes)
  
  # Generalized cluster configuration (recommended)
  clusterTypes:
    type: k3s
    binaryPath: "/usr/local/bin/k3s"
```

### Default Behavior

- **Detection Enabled**: By default, cluster detection is enabled when onboarding is enabled
- **Detection Interval**: By default, 120 seconds (2 minutes)
- **Cluster Types**: If no cluster types are configured, defaults to both K3s and RKE2 with standard paths
- **Dedicated Communication**: Cluster status updates use a separate `UpdateClusterStatus` gRPC call, independent of node status
## Implementation Details

### Cluster Detection Process

1. **Periodic Scanning**: Runs every 2 minutes (configurable)
2. **Multi-Type Detection**: Checks configured cluster types in priority order
3. **Service Detection**: Uses `systemctl` to check if cluster services are active
4. **Version Detection**: Runs cluster binaries to get version information
5. **Kubeconfig Validation**: Validates retrieved kubeconfig for required fields

### Dedicated Cluster Status Updates

Cluster status updates are handled via a dedicated `UpdateClusterStatus` gRPC call:

- **Separate from Node Status**: Cluster updates don't interfere with regular node status reporting
- **Change-based Notifications**: Only sends updates when kubeconfig content changes (SHA256 hash comparison)
- **Kubeconfig Hash**: Sends SHA256 hash of kubeconfig content to avoid transmission issues
- **Error Handling**: Failures in cluster status updates don't affect node status reporting

### Kubeconfig Management

The kubeconfig manager:
- Tracks kubeconfig changes using SHA256 hashing
- Only notifies the host manager when kubeconfig content changes
- Uses dedicated `UpdateClusterStatus` API call
- Provides methods to clear kubeconfig when clusters are removed
- Thread-safe operations with read/write mutex protection


### Key Functions

- `DetectCluster()` - Main cluster detection entry point (supports multiple cluster types)
- `GetKubeconfig()` - Retrieve kubeconfig content from detected cluster
- `NotifyKubeconfig()` - Send kubeconfig to host manager via dedicated UpdateClusterStatus API
- `ValidateKubeconfig()` - Basic kubeconfig validation
- `UpdateClusterStatus()` - Dedicated gRPC call for cluster status updates

## Usage Example

The functionality is automatic once configured. The node-agent will:

1. Start the cluster detection goroutine
2. Scan for configured cluster types every 2 minutes
3. Detect cluster installation and retrieve kubeconfig when cluster is running
4. Send dedicated cluster status updates to host manager when kubeconfig changes
5. Clear kubeconfig state when clusters are no longer detected

### Configuration Examples

**Default configuration (both K3s and RKE2):**
```yaml
cluster:
  detectionEnabled: true
  detectionInterval: 120s
  # Will automatically default to both K3s and RKE2
```

**K3s only configuration:**
```yaml
cluster:
  detectionEnabled: true
  detectionInterval: 60s
  clusterTypes:
    type: k3s
    binaryPath: "/usr/local/bin/k3s"
```
