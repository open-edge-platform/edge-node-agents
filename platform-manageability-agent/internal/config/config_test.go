// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package config_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

var log = logger.Logger

func createConfigFile(t *testing.T, version string, logLevel string, guid string, url string, enabled bool,
	statusInterval time.Duration, metricsInterval time.Duration, statusEndpoint string, metricsEndpoint string,
	rpsAddress string, accessTokenPath string) string {

	f, err := os.CreateTemp(t.TempDir(), "test_config")
	require.NoError(t, err)

	c := config.Config{
		Version:  version,
		LogLevel: logLevel,
		GUID:     guid,
		Manageability: config.ConfigManageability{
			Enabled:           true,
			ServiceURL:        url,
			HeartbeatInterval: statusInterval,
		},
		StatusEndpoint: statusEndpoint,
		Metrics: config.ConfigMetrics{
			Enabled:  enabled,
			Endpoint: metricsEndpoint,
			Interval: metricsInterval,
		},
		RPSAddress:      rpsAddress,
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
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, statusInterval, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, true, cfg.Metrics.Enabled)
	assert.Equal(t, metricsEndpoint, cfg.Metrics.Endpoint)
	assert.Equal(t, metricsInterval, cfg.Metrics.Interval)
	assert.Equal(t, rpsAddress, cfg.RPSAddress)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
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
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

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
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, "", "", guid, url, true, 0*time.Second, 0*time.Second, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Version)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, 10*time.Second, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, true, cfg.Metrics.Enabled)
	assert.Equal(t, metricsEndpoint, cfg.Metrics.Endpoint)
	assert.Equal(t, 10*time.Second, cfg.Metrics.Interval)
	assert.Equal(t, rpsAddress, cfg.RPSAddress)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
}

func TestDisabledMetricsWithIntervalEndpointSet(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, false, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, statusInterval, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, false, cfg.Metrics.Enabled)
	assert.Equal(t, metricsEndpoint, cfg.Metrics.Endpoint)
	assert.Equal(t, metricsInterval, cfg.Metrics.Interval)
	assert.Equal(t, rpsAddress, cfg.RPSAddress)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
}

func TestDisabledMetricsWithIntervalEndpointNotSet(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusInterval := 30 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, false, statusInterval, 0*time.Second, statusEndpoint, "", rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, statusInterval, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, false, cfg.Metrics.Enabled)
	assert.Equal(t, "", cfg.Metrics.Endpoint)
	assert.Equal(t, 0*time.Second, cfg.Metrics.Interval)
	assert.Equal(t, rpsAddress, cfg.RPSAddress)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
}

func TestMissingHeartbeatIntervals(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, 0*time.Second, 0*time.Second, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, url, cfg.Manageability.ServiceURL)
	assert.Equal(t, 10*time.Second, cfg.Manageability.HeartbeatInterval)
	assert.Equal(t, statusEndpoint, cfg.StatusEndpoint)
	assert.Equal(t, true, cfg.Metrics.Enabled)
	assert.Equal(t, metricsEndpoint, cfg.Metrics.Endpoint)
	assert.Equal(t, 10*time.Second, cfg.Metrics.Interval)
	assert.Equal(t, rpsAddress, cfg.RPSAddress)
	assert.Equal(t, accessTokenPath, cfg.AccessTokenPath)
}

func TestMissingServiceURL(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, "", true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
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
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, "")
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
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, "", metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingMetricsEndpoint(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, "", rpsAddress, accessTokenPath)
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
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "invalid-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestInvalidMetricsEndpoint(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "invalid-metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingGUID(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	url := "localhost"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	rpsAddress := "test-address.test.com"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, "", url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, rpsAddress, accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestMissingRPSAddress(t *testing.T) {
	version := "v0.1.0"
	logLevel := "info"
	guid := "aaaaaaaa-0000-1111-2222-bbbbbbbbcccc"
	url := "localhost"
	statusInterval := 30 * time.Second
	metricsInterval := 15 * time.Second
	statusEndpoint := "unix://test-socket.sock"
	metricsEndpoint := "unix://metrics-test-socket.sock"
	accessTokenPath := "/etc/intel_edge_node/tokens/platform-manageability-agent/access_token"

	fileName := createConfigFile(t, version, logLevel, guid, url, true, statusInterval, metricsInterval, statusEndpoint, metricsEndpoint, "", accessTokenPath)
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName, log)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func FuzzNew(f *testing.F) {
	f.Add("/tmp/test_file.yaml")
	f.Fuzz(func(t *testing.T, testConfigPath string) {
		conf, err := config.New(testConfigPath, log)

		// Test if error received, if yes, confirm that no config has been provided
		if err != nil {
			if conf != nil {
				t.Errorf("Error %v returned but configuration is no nil!", err)
			}
		}

		// Test if no error received, if yes, confirm that required configurations have been set
		if err == nil {
			if conf == nil {
				t.Error("No error received but configuration is nil!")
			}
			if conf != nil {
				if conf.GUID == "" {
					t.Error("GUID is not set in configuration")
				}
				if conf.Manageability.ServiceURL == "" {
					t.Error("ServiceURL is not set in configuration")
				}
				if conf.Manageability.HeartbeatInterval <= 0 {
					t.Error("HeartbeatInterval is set to an invalid value in configuration")
				}
				if conf.RPSAddress == "" {
					t.Error("RPSAddress is not set in configuration")
				}
				if conf.StatusEndpoint == "" {
					t.Error("StatusEndpoint is not set in configuration")
				}
				if !strings.HasPrefix(conf.StatusEndpoint, "unix://") {
					t.Error("StatusEndpoint is not a Unix socket address")
				}
				if conf.Metrics.Endpoint == "" {
					t.Error("MetricsEndpoint is not set in configuration")
				}
				if !strings.HasPrefix(conf.Metrics.Endpoint, "unix://") {
					t.Error("MetricsEndpoint is not a Unix socket address")
				}
				if conf.Metrics.Interval <= 0 {
					t.Error("MetricsInterval is set to an invalid value in configuration")
				}
				if conf.AccessTokenPath == "" {
					t.Error("AccessTokenPath is not set in configuration")
				}
			}
		}
	})
}

func FuzzConfigNew(f *testing.F) {
	exampleConfigFileContents := []byte("# SPDX-FileCopyrightText: (C) 2025 Intel Corporation\n# SPDX-License-Identifier: Apache-2.0\n\n---\nversion: v0.1.0\nlogLevel: info\nGUID: 'aaaaaaaa-0000-1111-2222-bbbbbbbbcccc'\nmanageability:\n  enabled: true\n  serviceURL: 'infra.test.edgeorch.intel.com:443'\n  heartbeatInterval: 10s\nrpsAddress: 'rps.test.edgeorch.intel.com'\nstatusEndpoint: 'unix:///run/node-agent/node-agent.sock'\nmetrics:\nenabled: true\nendpoint: 'unix:///run/platform-observability-agent/platform-observability-agent.sock'\ninterval: 10s\naccessTokenPath: /etc/intel_edge_node/tokens/platform-manageability-agent/access_token")
	f.Add(exampleConfigFileContents)
	f.Fuzz(func(t *testing.T, testConfigFileContents []byte) {
		testFile, err := os.CreateTemp(t.TempDir(), "example_config.yaml")
		if err != nil {
			t.Error("Error creating test config file")
		}
		defer os.Remove(testFile.Name())

		_, err = testFile.Write(testConfigFileContents)
		if err != nil {
			t.Error("Error writing information to config file")
		}

		err = testFile.Close()
		if err != nil {
			t.Error("Error closing file")
		}

		conf, err := config.New(testFile.Name(), log)
		if err != nil {
			if conf != nil {
				t.Errorf("Error %v returned but configuration is not nil!", err)
			}
		}
		if err == nil {
			if conf == nil {
				t.Error("No error returned but configuration is nil!")
			}
			if conf != nil {
				if conf.GUID == "" {
					t.Error("GUID is not set in configuration")
				}
				if conf.Manageability.ServiceURL == "" {
					t.Error("ServiceURL is not set in configuration")
				}
				if conf.Manageability.HeartbeatInterval <= 0 {
					t.Error("HeartbeatInterval is set to an invalid value in configuration")
				}
				if conf.RPSAddress == "" {
					t.Error("RPSAddress is not set in configuration")
				}
				if conf.StatusEndpoint == "" {
					t.Error("StatusEndpoint is not set in configuration")
				}
				if !strings.HasPrefix(conf.StatusEndpoint, "unix://") {
					t.Error("StatusEndpoint is not a Unix socket address")
				}
				if conf.Metrics.Endpoint == "" && conf.Metrics.Enabled {
					t.Error("MetricsEndpoint is not set in configuration")
				}
				if !strings.HasPrefix(conf.Metrics.Endpoint, "unix://") && conf.Metrics.Enabled {
					t.Error("MetricsEndpoint is not a Unix socket address")
				}
				if conf.Metrics.Interval <= 0 && conf.Metrics.Enabled {
					t.Error("MetricsInterval is set to an invalid value in configuration")
				}
				if conf.AccessTokenPath == "" {
					t.Error("AccessTokenPath is not set in configuration")
				}
			}
		}
	})
}
