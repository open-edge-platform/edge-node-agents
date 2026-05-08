<!---
  SPDX-FileCopyrightText: (C) 2026 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Node Agent Cluster Detection and Kubeconfig Management

This document describes the cluster detection and kubeconfig management functionality added to the node-agent.

## Overview

The node-agent now has the ability to:
1. **Detect running clusters** on the node (only k3s and RKE2 supported at the moment).
2. **Retrieve kubeconfig** from detected clusters.
3. **Send dedicated cluster status updates** to the host manager when kubeconfig changes.
4. **Manage kubeconfig lifecycle** independently of node status reporting.

## Architecture

### Components Added

1. **`cluster.ClusterDetector`** - Detects running cluster installations
2. **`cluster.KubeconfigManager`** - Manages kubeconfig lifecycle and dedicated cluster status communication

### Detection Logic

The cluster detector supports multiple cluster types and checks for:
- **K3s clusters**: Configurable binary path (defaults to `/var/lib/rancher/k3s/bin/k3s`), checks systemd service status
- **RKE2 clusters**: Configurable binary path (defaults to `/usr/local/bin/rke2`), checks systemd service status

When a cluster is detected, it:
- Checks if the cluster service is running
- Locates the kubeconfig file in standard locations
- Validates the kubeconfig content
- **Sends dedicated cluster status updates** to the host manager via `UpdateClusterStatus` API

The system looks for kubeconfig files in these locations:
- **K3s**: `/etc/rancher/k3s/k3s.yaml`
- **RKE2**: `/etc/rancher/rke2/rke2.yaml`

Add the following section to your `node-agent.yaml`:

```yaml
cluster:
  # Default configuration (both K3s and RKE2)
  detectionEnabled: true      # Enable/disable cluster detection
  detectionInterval: 120s     # How often to check for clusters (default: 2 minutes)
  
  # Generalized cluster configuration (recommended)
  # Will automatically default to K3s
  clusterTypes:
    type: k3s
    binaryPath: "/usr/local/bin/k3s"
```

### Kubeconfig Management

The kubeconfig manager:
- Tracks kubeconfig changes using SHA256 hashing
- Only notifies the host manager when kubeconfig content changes
- Uses dedicated `UpdateClusterStatus` API call
- Provides methods to clear kubeconfig when clusters are removed
