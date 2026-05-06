// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

var clusterLog = logger.Logger

// represents detected cluster information
type ClusterInfo struct {
	Type           string    `json:"type"`           // k3s
	Status         string    `json:"status"`         // running, stopped, error
	Version        string    `json:"version"`        // cluster version
	KubeconfigPath string    `json:"kubeconfigPath"` // path to kubeconfig file
	DetectedAt     time.Time `json:"detectedAt"`     // when cluster was detected
}

// handles detection of running clusters on the node
type ClusterDetector struct {
	nodeID      string
	clusterType config.ClusterType
}

// creates a new cluster detector
func NewClusterDetector(nodeID string, clusterType config.ClusterType) *ClusterDetector {
	return &ClusterDetector{
		nodeID:      nodeID,
		clusterType: clusterType,
	}
}

// checks if there's a cluster running on the node
func (cd *ClusterDetector) DetectCluster() (*ClusterInfo, error) {
	clusterLog.Debug("Detecting clusters on the node...")

	switch cd.clusterType.Type {
	case "k3s":
		if clusterInfo, err := cd.detectK3s(cd.clusterType.BinaryPath); err == nil {
			clusterLog.Infof("Detected K3s cluster: version=%s, status=%s", clusterInfo.Version, clusterInfo.Status)
			return clusterInfo, nil
		}
	case "rke2":
		if clusterInfo, err := cd.detectRKE2(cd.clusterType.BinaryPath); err == nil {
			clusterLog.Infof("Detected RKE2 cluster: version=%s, status=%s", clusterInfo.Version, clusterInfo.Status)
			return clusterInfo, nil
		}
	default:
		clusterLog.Warnf("Unsupported cluster type: %s", cd.clusterType.Type)
	}

	return nil, fmt.Errorf("no cluster detected on node")
}

// detects K3s cluster installation
func (cd *ClusterDetector) detectK3s(k3sBinaryPath string) (*ClusterInfo, error) {
	// Check if k3s binary exists
	if _, err := os.Stat(k3sBinaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("K3s binary not found at %s", k3sBinaryPath)
	}

	// Get K3s version
	cmd := exec.Command(k3sBinaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get K3s version: %v", err)
	}

	versionParts := strings.Fields(string(output))
	version := "unknown"
	if len(versionParts) >= 3 {
		version = strings.TrimSpace(versionParts[2])
	}

	// Check if K3s service is running
	status := "stopped"
	if cd.isServiceActive("k3s") {
		status = "running"
	}

	// Look for kubeconfig
	kubeconfigPath := "/etc/rancher/k3s/k3s.yaml"
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		kubeconfigPath = ""
	}

	return &ClusterInfo{
		Type:           "k3s",
		Status:         status,
		Version:        version,
		KubeconfigPath: kubeconfigPath,
		DetectedAt:     time.Now(),
	}, nil
}

// detects RKE2 cluster installation
func (cd *ClusterDetector) detectRKE2(rke2BinaryPath string) (*ClusterInfo, error) {
	// Check if rke2 binary exists
	if _, err := os.Stat(rke2BinaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("RKE2 binary not found at %s", rke2BinaryPath)
	}

	// Get RKE2 version
	cmd := exec.Command(rke2BinaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get RKE2 version: %v", err)
	}

	versionParts := strings.Fields(string(output))
	version := "unknown"
	if len(versionParts) >= 3 {
		version = strings.TrimSpace(versionParts[2])
	}

	// Check if RKE2 server or agent service is running
	status := "stopped"
	if cd.isServiceActive("rke2-server") || cd.isServiceActive("rke2-agent") {
		status = "running"
	}

	// Look for kubeconfig (RKE2 uses different path)
	kubeconfigPath := "/etc/rancher/rke2/rke2.yaml"
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		kubeconfigPath = ""
	}

	return &ClusterInfo{
		Type:           "rke2",
		Status:         status,
		Version:        version,
		KubeconfigPath: kubeconfigPath,
		DetectedAt:     time.Now(),
	}, nil
}

// checks if a systemd service is active
func (cd *ClusterDetector) isServiceActive(serviceName string) bool {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

// retrieves the kubeconfig content from the detected cluster
func (cd *ClusterDetector) GetKubeconfig(clusterInfo *ClusterInfo) ([]byte, error) {
	if clusterInfo == nil || clusterInfo.KubeconfigPath == "" {
		return nil, fmt.Errorf("no kubeconfig available for cluster")
	}

	clusterLog.Debugf("Reading kubeconfig from: %s", clusterInfo.KubeconfigPath)

	content, err := os.ReadFile(clusterInfo.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from %s: %v", clusterInfo.KubeconfigPath, err)
	}

	clusterLog.Infof("Successfully retrieved kubeconfig: %d bytes", len(content))
	return content, nil
}

// performs basic validation of kubeconfig content
func (cd *ClusterDetector) ValidateKubeconfig(kubeconfigData []byte) error {
	if len(kubeconfigData) == 0 {
		return fmt.Errorf("kubeconfig is empty")
	}

	content := string(kubeconfigData)

	// Basic validation - check for required fields
	requiredFields := []string{"apiVersion", "kind: Config", "clusters:", "users:", "contexts:"}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			return fmt.Errorf("kubeconfig missing required field: %s", field)
		}
	}

	clusterLog.Debug("Kubeconfig validation passed")
	return nil
}
