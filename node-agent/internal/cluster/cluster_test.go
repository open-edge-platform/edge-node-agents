// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewClusterDetector(t *testing.T) {
	nodeID := "test-node-123"
	detector := NewClusterDetector(nodeID)

	if detector == nil {
		t.Fatal("NewClusterDetector should not return nil")
	}

	if detector.nodeID != nodeID {
		t.Errorf("Expected nodeID %s, got %s", nodeID, detector.nodeID)
	}
}

func TestValidateKubeconfig(t *testing.T) {
	detector := NewClusterDetector("test-node")

	tests := []struct {
		name        string
		kubeconfig  string
		shouldError bool
	}{
		{
			name:        "empty kubeconfig",
			kubeconfig:  "",
			shouldError: true,
		},
		{
			name: "valid kubeconfig",
			kubeconfig: `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
  name: example-cluster
contexts:
- context:
    cluster: example-cluster
    user: example-user
  name: example-context
users:
- name: example-user
  user:
    token: example-token
`,
			shouldError: false,
		},
		{
			name: "missing required field",
			kubeconfig: `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
  name: example-cluster
`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.ValidateKubeconfig([]byte(tt.kubeconfig))
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestKubeconfigManager(t *testing.T) {
	// Create a kubeconfig manager with nil client for testing
	mgr := NewKubeconfigManager(nil, "test-node")

	if mgr == nil {
		t.Fatal("NewKubeconfigManager should not return nil")
	}

	if mgr.HasKubeconfig() {
		t.Error("New manager should not have kubeconfig initially")
	}

	if mgr.KubeconfigSize() != 0 {
		t.Error("New manager should have kubeconfig size 0")
	}

	// Test cluster status formatting
	clusterInfo := &ClusterInfo{
		Type:           "k3s",
		Status:         "running",
		Version:        "v1.28.2+k3s1",
		KubeconfigPath: "/etc/rancher/k3s/k3s.yaml",
		DetectedAt:     time.Now(),
	}

	status := mgr.GetClusterStatus(clusterInfo)
	expectedSubstrings := []string{"k3s", "v1.28.2+k3s1", "running"}

	for _, substring := range expectedSubstrings {
		if !strings.Contains(status, substring) {
			t.Errorf("Status should contain '%s', got: %s", substring, status)
		}
	}
}

func TestClusterInfo(t *testing.T) {
	now := time.Now()
	clusterInfo := &ClusterInfo{
		Type:           "k3s",
		Status:         "running",
		Version:        "v1.28.2+k3s1",
		KubeconfigPath: "/etc/rancher/k3s/k3s.yaml",
		DetectedAt:     now,
	}

	if clusterInfo.Type != "k3s" {
		t.Errorf("Expected type 'k3s', got %s", clusterInfo.Type)
	}

	if clusterInfo.Status != "running" {
		t.Errorf("Expected status 'running', got %s", clusterInfo.Status)
	}

	if clusterInfo.DetectedAt != now {
		t.Error("DetectedAt timestamp should match")
	}
}

// This test will only run if K3s is actually installed on the system
func TestDetectK3sIntegration(t *testing.T) {
	// Skip this test if we're not in an environment with K3s
	if _, err := os.Stat("/usr/local/bin/k3s"); os.IsNotExist(err) {
		t.Skip("Skipping K3s detection test - K3s not installed")
	}

	detector := NewClusterDetector("test-node")
	clusterInfo, err := detector.detectK3s()

	if err != nil {
		t.Logf("K3s detection failed (expected if K3s not running): %v", err)
		return
	}

	if clusterInfo.Type != "k3s" {
		t.Errorf("Expected type 'k3s', got %s", clusterInfo.Type)
	}

	if clusterInfo.Version == "" || clusterInfo.Version == "unknown" {
		t.Error("Should have detected K3s version")
	}

	t.Logf("Detected K3s: version=%s, status=%s, kubeconfig=%s",
		clusterInfo.Version, clusterInfo.Status, clusterInfo.KubeconfigPath)
}

func TestDetectCluster(t *testing.T) {
	detector := NewClusterDetector("test-node")

	// This test may fail if no cluster is installed, which is expected
	clusterInfo, err := detector.DetectCluster()

	if err != nil {
		t.Logf("No cluster detected (expected): %v", err)
		return
	}

	// If a cluster is detected, validate the structure
	if clusterInfo.Type == "" {
		t.Error("Detected cluster should have a type")
	}

	if clusterInfo.Status == "" {
		t.Error("Detected cluster should have a status")
	}

	t.Logf("Detected cluster: type=%s, status=%s, version=%s",
		clusterInfo.Type, clusterInfo.Status, clusterInfo.Version)
}
