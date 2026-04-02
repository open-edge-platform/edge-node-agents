// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

var log = logger.Logger

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
	nodeID string
}

// creates a new cluster detector
func NewClusterDetector(nodeID string) *ClusterDetector {
	return &ClusterDetector{
		nodeID: nodeID,
	}
}

// checks if there's a cluster running on the node
func (cd *ClusterDetector) DetectCluster() (*ClusterInfo, error) {
	log.Debug("Detecting clusters on the node...")

	if clusterInfo, err := cd.detectK3s(); err == nil {
		log.Infof("Detected K3s cluster: version=%s, status=%s", clusterInfo.Version, clusterInfo.Status)
		return clusterInfo, nil
	}

	return nil, fmt.Errorf("no cluster detected on node")
}

// detects K3s cluster installation
func (cd *ClusterDetector) detectK3s() (*ClusterInfo, error) {
	// Check if k3s binary exists
	k3sBinary := "/usr/local/bin/k3s"
	if _, err := os.Stat(k3sBinary); os.IsNotExist(err) {
		return nil, fmt.Errorf("K3s binary not found")
	}

	// Get K3s version
	cmd := exec.Command(k3sBinary, "--version")
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

	log.Debugf("Reading kubeconfig from: %s", clusterInfo.KubeconfigPath)

	content, err := os.ReadFile(clusterInfo.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from %s: %v", clusterInfo.KubeconfigPath, err)
	}

	log.Infof("Successfully retrieved kubeconfig: %d bytes", len(content))
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

	log.Debug("Kubeconfig validation passed")
	return nil
}
