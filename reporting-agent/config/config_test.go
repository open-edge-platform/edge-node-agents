// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_Default_NoConfigFile checks that LoadConfig returns defaults when no config file is provided.
func TestLoadConfig_Default_NoConfigFile(t *testing.T) {
	cmd := newFakeCobraCmd(t, "")
	cfg := LoadConfig(cmd)
	require.Equal(t, setDefaults(), cfg, "LoadConfig should return default config when no config file is provided")
}

// TestLoadConfig_FilePresent checks that LoadConfig loads config from a valid file.
func TestLoadConfig_FilePresent(t *testing.T) {
	tmpFile := createTempConfigFile(t, `
k8s:
  k3sKubectlPath: "/custom/k3s"
  k3sKubeConfigPath: "/custom/k3s.yaml"
  rke2KubectlPath: "/custom/rke2"
  rke2KubeConfigPath: "/custom/rke2.yaml"
`)
	defer os.Remove(tmpFile)

	cmd := newFakeCobraCmd(t, tmpFile)
	cfg := LoadConfig(cmd)
	require.Equal(t, "/custom/k3s", cfg.K8s.K3sKubectlPath, "LoadConfig should load k3sKubectlPath from file")
	require.Equal(t, "/custom/k3s.yaml", cfg.K8s.K3sKubeConfigPath, "LoadConfig should load k3sKubeConfigPath from file")
	require.Equal(t, "/custom/rke2", cfg.K8s.Rke2KubectlPath, "LoadConfig should load rke2KubectlPath from file")
	require.Equal(t, "/custom/rke2.yaml", cfg.K8s.Rke2KubeConfigPath, "LoadConfig should load rke2KubeConfigPath from file")
}

// TestLoadConfig_FileUnreadable checks that LoadConfig returns defaults if config file is unreadable.
func TestLoadConfig_FileUnreadable(t *testing.T) {
	cmd := newFakeCobraCmd(t, "/nonexistent/path/to/config.yaml")
	cfg := LoadConfig(cmd)
	require.Equal(t, setDefaults(), cfg, "LoadConfig should return default config if config file is unreadable")
}

// TestLoadConfig_UnmarshalError checks that LoadConfig returns defaults if unmarshal fails.
func TestLoadConfig_UnmarshalError(t *testing.T) {
	// This YAML is valid, but the structure is not compatible with Config struct, so unmarshal will fail.
	tmpFile := createTempConfigFile(t, `
k8s: "this-should-be-a-map-not-a-string"
`)
	defer os.Remove(tmpFile)

	cmd := newFakeCobraCmd(t, tmpFile)
	cfg := LoadConfig(cmd)
	require.Equal(t, setDefaults(), cfg, "LoadConfig should return default config if unmarshal fails")
}

// TestSetDefaults returns the expected default config.
func TestSetDefaults(t *testing.T) {
	def := setDefaults()
	require.Equal(t, "/var/lib/rancher/k3s/bin/k3s kubectl", def.K8s.K3sKubectlPath, "SetDefaults should set correct K3sKubectlPath")
	require.Equal(t, "/etc/rancher/k3s/k3s.yaml", def.K8s.K3sKubeConfigPath, "SetDefaults should set correct K3sKubeConfigPath")
	require.Equal(t, "/var/lib/rancher/rke2/bin/kubectl", def.K8s.Rke2KubectlPath, "SetDefaults should set correct rke2KubectlPath")
	require.Equal(t, "/etc/rancher/rke2/rke2.yaml", def.K8s.Rke2KubeConfigPath, "SetDefaults should set correct rke2KubeConfigPath")
}

// newFakeCobraCmd creates a cobra.Command with a --config flag set to the given value.
func newFakeCobraCmd(t *testing.T, configPath string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	err := cmd.Flags().Set("config", configPath)
	require.NoError(t, err, "Should set config flag without error")
	return cmd
}

// createTempConfigFile creates a temporary YAML config file with the given content.
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0640), "Should write temp config file") // Gosec: use 0640
	return tmpFile
}
