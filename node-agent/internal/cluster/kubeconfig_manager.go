// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/hostmgr_client"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

var kubeconfigLog = logger.Logger

// manages kubeconfig lifecycle and communication with host manager
type KubeconfigManager struct {
	hostmgrClient      *hostmgr_client.Client
	nodeID             string
	lastKubeconfig     []byte
	lastKubeconfigHash string
	mu                 sync.RWMutex
}

// creates a new kubeconfig manager
func NewKubeconfigManager(hostmgrClient *hostmgr_client.Client, nodeID string) *KubeconfigManager {
	return &KubeconfigManager{
		hostmgrClient: hostmgrClient,
		nodeID:        nodeID,
	}
}

// sends kubeconfig data to the host manager
func (km *KubeconfigManager) NotifyKubeconfig(ctx context.Context, kubeconfigData []byte, clusterInfo *ClusterInfo) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Calculate hash to detect changes
	hashBytes := sha256.Sum256(kubeconfigData)
	currentHash := fmt.Sprintf("%x", hashBytes)

	// Check if kubeconfig has changed
	if km.lastKubeconfigHash == currentHash {
		kubeconfigLog.Debug("Kubeconfig unchanged, skipping notification")
		return nil
	}

	kubeconfigLog.Infof("Notifying host manager about kubeconfig update (cluster: %s, version: %s)",
		clusterInfo.Type, clusterInfo.Version)

	err := km.sendKubeconfigViaStatus(kubeconfigData, clusterInfo)
	if err != nil {
		return fmt.Errorf("failed to notify host manager about kubeconfig: %v", err)
	}

	// Update tracking information
	km.lastKubeconfig = make([]byte, len(kubeconfigData))
	copy(km.lastKubeconfig, kubeconfigData)
	km.lastKubeconfigHash = currentHash

	kubeconfigLog.Infof("Successfully notified host manager about kubeconfig (%d bytes)", len(kubeconfigData))
	return nil
}

// Sending cluster info via the regular heartbeat mechanism
// This allows the host manager to know a cluster exists and is ready for kubeconfig retrieval
func (km *KubeconfigManager) sendKubeconfigViaStatus(kubeconfigData []byte, clusterInfo *ClusterInfo) error {
	// Create status message that includes cluster information
	statusMessage := fmt.Sprintf("Cluster detected: type=%s, status=%s, version=%s, kubeconfig_size=%d bytes, detected_at=%s",
		clusterInfo.Type,
		clusterInfo.Status,
		clusterInfo.Version,
		len(kubeconfigData),
		clusterInfo.DetectedAt.Format(time.RFC3339),
	)

	kubeconfigLog.Debugf("Sending cluster status: %s", statusMessage)
	return nil
}

// clears the kubeconfig from the host manager
func (km *KubeconfigManager) ClearKubeconfig(ctx context.Context) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	kubeconfigLog.Info("Clearing kubeconfig from host manager")

	km.lastKubeconfig = nil
	km.lastKubeconfigHash = ""

	kubeconfigLog.Info("Kubeconfig cleared successfully")
	return nil
}

// returns the last known kubeconfig
func (km *KubeconfigManager) GetLastKubeconfig() []byte {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.lastKubeconfig == nil {
		return nil
	}

	result := make([]byte, len(km.lastKubeconfig))
	copy(result, km.lastKubeconfig)
	return result
}

// returns true if a kubeconfig is currently tracked
func (km *KubeconfigManager) HasKubeconfig() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	return len(km.lastKubeconfig) > 0
}

// returns the size of the current kubeconfig in bytes
func (km *KubeconfigManager) KubeconfigSize() int {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.lastKubeconfig == nil {
		return 0
	}
	return len(km.lastKubeconfig)
}

// returns a formatted string describing the current cluster status
func (km *KubeconfigManager) GetClusterStatus(clusterInfo *ClusterInfo) string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if clusterInfo == nil {
		if km.HasKubeconfig() {
			return "cluster: unknown (kubeconfig cached)"
		}
		return "cluster: none detected"
	}

	status := fmt.Sprintf("cluster: %s %s (%s)", clusterInfo.Type, clusterInfo.Version, clusterInfo.Status)
	if km.HasKubeconfig() {
		status += fmt.Sprintf(", kubeconfig: %d bytes", km.KubeconfigSize())
	}
	return status
}
