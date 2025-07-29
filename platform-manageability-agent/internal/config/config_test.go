// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

var log = logger.Logger

func createConfigFile(t *testing.T, version string, logLevel string, guid string, url string, interval time.Duration,
	statusEndpoint string, accessTokenPath string, address string) string {
	f, err := os.CreateTemp(t.TempDir(), "test_config")
	require.NoError(t, err)

	c := config.Config{
		Version:  version,
		LogLevel: logLevel,
		GUID:     guid,
		Manageability: config.ConfigManageability{
			Enabled:           true,
			ServiceURL:        url,
			HeartbeatInterval: interval,
		},
		StatusEndpoint:  statusEndpoint,
		MetricsEndpoint: statusEndpoint,
		MetricsInterval: interval,
		RPSAddress:      address,
		AccessTokenPath: accessTokenPath,
	}

	file, err := yaml.Marshal(c)
	require.NoError(t, err)

	_, err = f.Write(file)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
	return f.Name()
}

func TestValidConfig(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	interval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, interval, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, interval, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, statusEndpoint, cfg.MetricsEndpoint)
	assert.Equal(t, interval, cfg.MetricsInterval)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
	assert.Equal(t, address, cfg.RPSAddress)
}

func TestInvalidConfigPath(t *testing.T) {
	cfg, err := config.New("non_existent_path", log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestInvalidConfigContent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test_config")
	require.NoError(t, err)
	defer os.Remove(f.Name()) // clean up

	_, err = f.WriteString("not a yaml for onboarding or tls")
	require.NoError(t, err)

	cfg, err := config.New(f.Name(), log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestSymlinkConfigPath(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	interval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, interval, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName)

	symlinkConfig := "/tmp/sysmlink_config.txt"
	defer os.Remove(symlinkConfig)
	err := os.Symlink(fileName, symlinkConfig)
	require.NoError(t, err)

	cfg, err := config.New(symlinkConfig, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestPartialConfigFile(t *testing.T) {
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, "", "", guid, url, 0*time.Second, statusEndpoint, accessTokenPath, "")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Version)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, 10*time.Second, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, statusEndpoint, cfg.MetricsEndpoint)
	assert.Equal(t, 10*time.Second, cfg.MetricsInterval)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
	assert.Equal(t, "", cfg.RPSAddress)
}

func TestMissingHeartbeatIntervals(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, 0*time.Second, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, 10*time.Second, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, statusEndpoint, cfg.MetricsEndpoint)
	assert.Equal(t, 10*time.Second, cfg.MetricsInterval)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
	assert.Equal(t, address, cfg.RPSAddress)
}

func TestMissingServiceURL(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	interval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, "", interval, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingTokenPath(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	interval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, interval, statusEndpoint, "", address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingStatusEndpoint(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	interval := 30 * time.Second
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, interval, "", accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestInvalidStatusEndpoint(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	interval := 30 * time.Second
	statusEndpoint := "invalid-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, guid, url, interval, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingGUID(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	url := "localhost"
	interval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"
	address := "infra.test.edgeorch.intel.com:443"

	fileName := createConfigFile(t, version, logLevel, "", url, interval, statusEndpoint, accessTokenPath, address)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}
