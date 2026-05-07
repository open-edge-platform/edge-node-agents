// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/hostmgr_client"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

var managerLog = logger.Logger

// manages kubeconfig lifecycle and communication with host manager
// tracks the last known kubeconfig and only notifies host manager
// on changes to avoid unnecessary updates
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
func (km *KubeconfigManager) NotifyKubeconfig(ctx context.Context, kubeconfigData []byte, clusterInfo *ClusterInfo, confs *config.NodeAgentConfig) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if clusterInfo == nil {
		return fmt.Errorf("clusterInfo cannot be nil")
	}

	// Calculate hash to detect changes
	hashBytes := sha256.Sum256(kubeconfigData)
	currentHash := fmt.Sprintf("%x", hashBytes)

	//Avoid unnecessary updates to host manager and DB when kubeconfig content is the same
	if km.lastKubeconfigHash == currentHash {
		managerLog.Debug("Kubeconfig unchanged, skipping notification")
		return nil
	}

	managerLog.Infof("Notifying host manager about kubeconfig update (cluster: %s, version: %s)",
		clusterInfo.Type, clusterInfo.Version)

	// Encode kubeconfig in base64 before storing to avoid any formatting issues while transmitting over gRPC and storing in DB
	kubeconfigBlob := base64.StdEncoding.EncodeToString(kubeconfigData)

	// Notify host manager about kubeconfig update
	tokenFile := filepath.Join(confs.Auth.AccessTokenPath, "node-agent", config.AccessToken)

	// Only update cluster status if hostmgr client is available (skip for tests with nil client)
	if km.hostmgrClient != nil {
		err := km.hostmgrClient.UpdateClusterStatus(utils.GetAuthContext(ctx, tokenFile), kubeconfigBlob)
		if err != nil {
			managerLog.Errorf("not able to update node status to running : %v", err)
			return fmt.Errorf("failed to update cluster status: %v", err)
		}
	}

	// Update tracking information
	km.lastKubeconfig = make([]byte, len(kubeconfigData))
	copy(km.lastKubeconfig, kubeconfigData)
	km.lastKubeconfigHash = currentHash

	managerLog.Infof("Successfully notified host manager about kubeconfig (%d bytes)", len(kubeconfigData))
	return nil
}

// set the kubeconfig to nil and notify host manager to clear it
// avoid having stale kubeconfig data in host manager when cluster is deleted or becomes unreachable
func (km *KubeconfigManager) ClearKubeconfig(ctx context.Context, confs *config.NodeAgentConfig) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	managerLog.Info("Clearing kubeconfig from host manager")

	if km.hostmgrClient != nil {
		tokenFile := filepath.Join(confs.Auth.AccessTokenPath, "node-agent", config.AccessToken)

		// Empty string means "clear kubeconfig" on Host Manager side.
		if err := km.hostmgrClient.UpdateClusterStatus(utils.GetAuthContext(ctx, tokenFile), ""); err != nil {
			managerLog.Errorf("failed to clear kubeconfig in host manager: %v", err)
			return fmt.Errorf("failed to clear kubeconfig in host manager: %w", err)
		}
	}

	km.lastKubeconfig = nil
	km.lastKubeconfigHash = ""

	managerLog.Info("Kubeconfig cleared successfully")
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
		if len(km.lastKubeconfig) > 0 {
			return "cluster: unknown (kubeconfig cached)"
		}
		return "cluster: none detected"
	}

	status := fmt.Sprintf("cluster: %s %s (%s)", clusterInfo.Type, clusterInfo.Version, clusterInfo.Status)
	if len(km.lastKubeconfig) > 0 {
		status += fmt.Sprintf(", kubeconfig: %d bytes", len(km.lastKubeconfig))
	}
	return status
}
