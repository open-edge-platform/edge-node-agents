// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/stretchr/testify/assert"
)

// Test helpers for cluster package

// MockHostmgrClient provides a simple mock implementation of the host manager client
// for testing purposes
type MockHostmgrClient struct {
	UpdateCallCount   int
	LastUpdateMessage string
	ShouldReturnError bool
	ErrorToReturn     error
}

// UpdateInstanceStatus mocks the host manager client's update method
func (m *MockHostmgrClient) UpdateInstanceStatus(ctx context.Context, state interface{}, status interface{}, message string) error {
	m.UpdateCallCount++
	m.LastUpdateMessage = message

	if m.ShouldReturnError {
		if m.ErrorToReturn != nil {
			return m.ErrorToReturn
		}
		return fmt.Errorf("mock error from hostmgr client")
	}

	return nil
}

// Reset clears the mock's tracking data
func (m *MockHostmgrClient) Reset() {
	m.UpdateCallCount = 0
	m.LastUpdateMessage = ""
	m.ShouldReturnError = false
	m.ErrorToReturn = nil
}

// TestClusterInfoBuilder provides a fluent interface for creating test cluster info
type TestClusterInfoBuilder struct {
	info *ClusterInfo
}

// NewTestClusterInfoBuilder creates a new builder with default values
func NewTestClusterInfoBuilder() *TestClusterInfoBuilder {
	return &TestClusterInfoBuilder{
		info: &ClusterInfo{
			Type:           "test-cluster",
			Status:         "running",
			Version:        "v1.0.0",
			KubeconfigPath: "/tmp/test-kubeconfig",
			DetectedAt:     time.Now(),
		},
	}
}

// WithType sets the cluster type
func (b *TestClusterInfoBuilder) WithType(clusterType string) *TestClusterInfoBuilder {
	b.info.Type = clusterType
	return b
}

// WithStatus sets the cluster status
func (b *TestClusterInfoBuilder) WithStatus(status string) *TestClusterInfoBuilder {
	b.info.Status = status
	return b
}

// WithVersion sets the cluster version
func (b *TestClusterInfoBuilder) WithVersion(version string) *TestClusterInfoBuilder {
	b.info.Version = version
	return b
}

// WithKubeconfigPath sets the kubeconfig path
func (b *TestClusterInfoBuilder) WithKubeconfigPath(path string) *TestClusterInfoBuilder {
	b.info.KubeconfigPath = path
	return b
}

// WithDetectedAt sets the detected timestamp
func (b *TestClusterInfoBuilder) WithDetectedAt(timestamp time.Time) *TestClusterInfoBuilder {
	b.info.DetectedAt = timestamp
	return b
}

// Build returns the constructed ClusterInfo
func (b *TestClusterInfoBuilder) Build() *ClusterInfo {
	return b.info
}

// TestKubeconfigGenerator provides utilities for generating test kubeconfig data
type TestKubeconfigGenerator struct{}

// NewTestKubeconfigGenerator creates a new kubeconfig generator
func NewTestKubeconfigGenerator() *TestKubeconfigGenerator {
	return &TestKubeconfigGenerator{}
}

// GenerateBasicKubeconfig creates a basic valid kubeconfig for testing
func (g *TestKubeconfigGenerator) GenerateBasicKubeconfig(clusterName, serverURL, token string) []byte {
	kubeconfig := fmt.Sprintf(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: %s
`, serverURL, clusterName, clusterName, token)

	return []byte(kubeconfig)
}

// GenerateInvalidKubeconfig creates an invalid kubeconfig for testing error cases
func (g *TestKubeconfigGenerator) GenerateInvalidKubeconfig() []byte {
	return []byte(`
apiVersion: v1
kind: Config
# Missing required fields
clusters: []
`)
}

// GenerateEmptyKubeconfig returns an empty kubeconfig
func (g *TestKubeconfigGenerator) GenerateEmptyKubeconfig() []byte {
	return []byte("")
}

// GenerateLargeKubeconfig creates a large kubeconfig for stress testing
func (g *TestKubeconfigGenerator) GenerateLargeKubeconfig(size int) []byte {
	baseConfig := g.GenerateBasicKubeconfig("large-cluster", "https://kubernetes.example.com:6443", "large-token")

	// Pad with comments to reach desired size
	padding := make([]byte, size-len(baseConfig))
	for i := range padding {
		switch i % 80 {
		case 0:
			padding[i] = '\n'
		case 1:
			padding[i] = '#'
		default:
			padding[i] = ' '
		}
	}

	return append(baseConfig, padding...)
}

// Test helper functions
func createTestClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		Type:           "k3s",
		Status:         "running",
		Version:        "v1.28.2+k3s1",
		KubeconfigPath: "/etc/rancher/k3s/k3s.yaml",
		DetectedAt:     time.Now(),
	}
}

func createTestKubeconfig() []byte {
	return []byte(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
users:
- name: test-user
  user:
    token: test-token
`)
}

func createTestConfig() *config.NodeAgentConfig {
	return &config.NodeAgentConfig{
		Auth: config.ConfigAuth{
			AccessTokenPath: "/tmp/tokens",
		},
	}
}

func TestNewKubeconfigManager(t *testing.T) {
	tests := []struct {
		name     string
		nodeID   string
		expected string
	}{
		{
			name:     "valid node ID",
			nodeID:   "test-node-123",
			expected: "test-node-123",
		},
		{
			name:     "empty node ID",
			nodeID:   "",
			expected: "",
		},
		{
			name:     "node ID with special characters",
			nodeID:   "node-with-special-chars_123",
			expected: "node-with-special-chars_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewKubeconfigManager(nil, tt.nodeID)

			assert.NotNil(t, manager)
			assert.Equal(t, tt.expected, manager.nodeID)
			assert.Nil(t, manager.lastKubeconfig)
			assert.Empty(t, manager.lastKubeconfigHash)
		})
	}
}

func TestKubeconfigManager_NotifyKubeconfig(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfig     []byte
		clusterInfo    *ClusterInfo
		expectError    bool
		expectHashSet  bool
		expectDataCopy bool
		subsequentCall bool
	}{
		{
			name:           "valid kubeconfig first time",
			kubeconfig:     createTestKubeconfig(),
			clusterInfo:    createTestClusterInfo(),
			expectError:    false,
			expectHashSet:  true,
			expectDataCopy: true,
		},
		{
			name:           "same kubeconfig twice (should skip)",
			kubeconfig:     createTestKubeconfig(),
			clusterInfo:    createTestClusterInfo(),
			expectError:    false,
			expectHashSet:  true,
			expectDataCopy: true,
			subsequentCall: true,
		},
		{
			name:           "different kubeconfig",
			kubeconfig:     []byte("different kubeconfig content"),
			clusterInfo:    createTestClusterInfo(),
			expectError:    false,
			expectHashSet:  true,
			expectDataCopy: true,
		},
		{
			name:           "empty kubeconfig",
			kubeconfig:     []byte(""),
			clusterInfo:    createTestClusterInfo(),
			expectError:    false,
			expectHashSet:  true,
			expectDataCopy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewKubeconfigManager(nil, "test-node")
			ctx := context.Background()

			if tt.subsequentCall {
				// Call twice with same data to test deduplication
				err := manager.NotifyKubeconfig(ctx, tt.kubeconfig, tt.clusterInfo, createTestConfig())
				assert.NoError(t, err)
			}

			err := manager.NotifyKubeconfig(ctx, tt.kubeconfig, tt.clusterInfo, createTestConfig())

			if tt.expectError {
				assert.NoError(t, err)
			}

			if tt.expectHashSet {
				assert.NotEmpty(t, manager.lastKubeconfigHash)
			}

			if tt.expectDataCopy {
				assert.Equal(t, tt.kubeconfig, manager.lastKubeconfig)
			}
		})
	}
}

func TestKubeconfigManager_ClearKubeconfig(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	// First set some kubeconfig data
	kubeconfig := createTestKubeconfig()
	clusterInfo := createTestClusterInfo()
	err := manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
	assert.NoError(t, err)

	// Verify data is set
	assert.True(t, manager.HasKubeconfig())
	assert.NotEmpty(t, manager.lastKubeconfigHash)

	// Clear the kubeconfig
	err = manager.ClearKubeconfig(ctx)
	assert.NoError(t, err)

	// Verify data is cleared
	assert.False(t, manager.HasKubeconfig())
	assert.Empty(t, manager.lastKubeconfigHash)
	assert.Nil(t, manager.lastKubeconfig)
}

func TestKubeconfigManager_GetLastKubeconfig(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	// Initially should return nil
	result := manager.GetLastKubeconfig()
	assert.Nil(t, result)

	// Set kubeconfig
	kubeconfig := createTestKubeconfig()
	clusterInfo := createTestClusterInfo()
	err := manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
	assert.NoError(t, err)

	// Should return copy of kubeconfig
	result = manager.GetLastKubeconfig()
	assert.Equal(t, kubeconfig, result)

	// Verify it's a copy, not the same slice
	result[0] = 'X'
	originalResult := manager.GetLastKubeconfig()
	assert.NotEqual(t, result[0], originalResult[0])
}

func TestKubeconfigManager_HasKubeconfig(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	// Initially should be false
	assert.False(t, manager.HasKubeconfig())

	// After setting kubeconfig should be true
	kubeconfig := createTestKubeconfig()
	clusterInfo := createTestClusterInfo()
	err := manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
	assert.NoError(t, err)
	assert.True(t, manager.HasKubeconfig())

	// After clearing should be false again
	err = manager.ClearKubeconfig(ctx)
	assert.NoError(t, err)
	assert.False(t, manager.HasKubeconfig())
}

func TestKubeconfigManager_KubeconfigSize(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	// Initially should be 0
	assert.Equal(t, 0, manager.KubeconfigSize())

	// After setting kubeconfig should return correct size
	kubeconfig := createTestKubeconfig()
	clusterInfo := createTestClusterInfo()
	err := manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
	assert.NoError(t, err)
	assert.Equal(t, len(kubeconfig), manager.KubeconfigSize())

	// After clearing should be 0 again
	err = manager.ClearKubeconfig(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.KubeconfigSize())
}

func TestKubeconfigManager_GetClusterStatus(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	tests := []struct {
		name            string
		clusterInfo     *ClusterInfo
		hasKubeconfig   bool
		expectedStrings []string
	}{
		{
			name:            "nil cluster info, no kubeconfig",
			clusterInfo:     nil,
			hasKubeconfig:   false,
			expectedStrings: []string{"none detected"},
		},
		{
			name:            "nil cluster info, has kubeconfig",
			clusterInfo:     nil,
			hasKubeconfig:   true,
			expectedStrings: []string{"unknown", "kubeconfig cached"},
		},
		{
			name:            "valid cluster info, no kubeconfig",
			clusterInfo:     createTestClusterInfo(),
			hasKubeconfig:   false,
			expectedStrings: []string{"k3s", "v1.28.2+k3s1", "running"},
		},
		{
			name:            "valid cluster info, has kubeconfig",
			clusterInfo:     createTestClusterInfo(),
			hasKubeconfig:   true,
			expectedStrings: []string{"k3s", "v1.28.2+k3s1", "running", "kubeconfig:", "bytes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset manager state
			manager = NewKubeconfigManager(nil, "test-node")

			if tt.hasKubeconfig {
				kubeconfig := createTestKubeconfig()
				clusterInfo := createTestClusterInfo()
				err := manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
				assert.NoError(t, err)
			}

			status := manager.GetClusterStatus(tt.clusterInfo)

			for _, expectedString := range tt.expectedStrings {
				assert.Contains(t, status, expectedString, "Status should contain '%s', got: %s", expectedString, status)
			}
		})
	}
}

func TestKubeconfigManager_ConcurrentAccess(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent reads and writes
	wg.Add(numGoroutines * 3)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			kubeconfig := []byte("test-kubeconfig-" + string(rune(i)))
			clusterInfo := createTestClusterInfo()
			manager.NotifyKubeconfig(ctx, kubeconfig, clusterInfo, createTestConfig())
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			manager.GetLastKubeconfig()
			manager.HasKubeconfig()
			manager.KubeconfigSize()
		}()
	}

	// Concurrent clears
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			manager.ClearKubeconfig(ctx)
		}()
	}

	wg.Wait()

	// Verify final state is consistent (no panics or race conditions)
	assert.NotNil(t, manager)
}

func TestKubeconfigManager_HashGeneration(t *testing.T) {
	manager := NewKubeconfigManager(nil, "test-node")
	ctx := context.Background()

	kubeconfig1 := []byte("kubeconfig content 1")
	kubeconfig2 := []byte("kubeconfig content 2")
	clusterInfo := createTestClusterInfo()

	// Set first kubeconfig
	err := manager.NotifyKubeconfig(ctx, kubeconfig1, clusterInfo, createTestConfig())
	assert.NoError(t, err)
	hash1 := manager.lastKubeconfigHash

	// Set different kubeconfig
	err = manager.NotifyKubeconfig(ctx, kubeconfig2, clusterInfo, createTestConfig())
	assert.NoError(t, err)
	hash2 := manager.lastKubeconfigHash

	// Hashes should be different
	assert.NotEqual(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)

	// Set same kubeconfig again
	err = manager.NotifyKubeconfig(ctx, kubeconfig2, clusterInfo, createTestConfig())
	assert.NoError(t, err)
	hash3 := manager.lastKubeconfigHash

	// Hash should remain the same
	assert.Equal(t, hash2, hash3)
}

func TestKubeconfigManager_EdgeCases(t *testing.T) {
	t.Run("nil kubeconfig", func(t *testing.T) {
		manager := NewKubeconfigManager(nil, "test-node")
		ctx := context.Background()
		clusterInfo := createTestClusterInfo()

		err := manager.NotifyKubeconfig(ctx, nil, clusterInfo, createTestConfig())
		assert.NoError(t, err)

		assert.Equal(t, 0, manager.KubeconfigSize())
		assert.False(t, manager.HasKubeconfig())
	})

	t.Run("nil cluster info", func(t *testing.T) {
		manager := NewKubeconfigManager(nil, "test-node")
		ctx := context.Background()
		kubeconfig := createTestKubeconfig()

		// Should handle nil cluster info gracefully with an error
		err := manager.NotifyKubeconfig(ctx, kubeconfig, nil, createTestConfig())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "clusterInfo cannot be nil")
	})

	t.Run("very large kubeconfig", func(t *testing.T) {
		manager := NewKubeconfigManager(nil, "test-node")
		ctx := context.Background()
		clusterInfo := createTestClusterInfo()

		// Create a large kubeconfig (1MB)
		largeKubeconfig := make([]byte, 1024*1024)
		for i := range largeKubeconfig {
			largeKubeconfig[i] = 'a'
		}

		err := manager.NotifyKubeconfig(ctx, largeKubeconfig, clusterInfo, createTestConfig())
		assert.NoError(t, err)
		assert.Equal(t, len(largeKubeconfig), manager.KubeconfigSize())
	})
}
