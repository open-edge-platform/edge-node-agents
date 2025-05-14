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

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/config"
)

func createConfigFile(t *testing.T, version string, logLevel string, url string, interval time.Duration, accessTokenPath string) string {
	f, err := os.CreateTemp(t.TempDir(), "test_config")
	require.NoError(t, err)

	c := config.Config{
		Version:  version,
		LogLevel: logLevel,
		Onboarding: config.Onboarding{
			ServiceURL: url,
		},
		JWT: config.JWT{
			AccessTokenPath: accessTokenPath,
		},
		UpdateInterval: interval,
	}

	file, err := yaml.Marshal(c)
	require.NoError(t, err)

	_, err = f.Write(file)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
	return f.Name()
}

func TestNewValidConfigNoArgs(t *testing.T) {
	logLevel := "info"
	version := "v0.2.0"
	url := "localhost"
	interval := 15 * time.Second

	fileName := createConfigFile(t, version, logLevel, url, interval, "/etc/intel_edge_node/tokens/hd-agent/access_token")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.NoError(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, url, cfg.Onboarding.ServiceURL)
	assert.Equal(t, interval, cfg.UpdateInterval)
	assert.Equal(t, "/etc/intel_edge_node/tokens/hd-agent/access_token", cfg.JWT.AccessTokenPath)
}

func TestNewInvalidConfigPath(t *testing.T) {
	cfg, err := config.New("non_existent_path")
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNewInvalidConfigContent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test_config")
	require.NoError(t, err)
	defer os.Remove(f.Name()) // clean up

	_, err = f.WriteString("not a yaml for onboarding or tls")
	require.NoError(t, err)

	cfg, err := config.New(f.Name())
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestSymlinkConfigPath(t *testing.T) {
	logLevel := "info"
	version := "v0.2.0"
	url := "localhost"
	interval := 15 * time.Second

	fileName := createConfigFile(t, version, logLevel, url, interval, "/etc/edge/node/tokens/hd-agent/access_token")
	defer os.Remove(fileName)

	symlinkConfig := "/tmp/symlink_config.txt"
	defer os.Remove(symlinkConfig)
	err := os.Symlink(fileName, symlinkConfig)
	require.NoError(t, err)

	cfg, err := config.New(symlinkConfig)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestPartialConfigFile(t *testing.T) {
	url := "localhost"

	fileName := createConfigFile(t, "", "", url, 0*time.Second, "/etc/intel_edge_node/tokens/hd-agent/access_token")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", cfg.Version)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, url, cfg.Onboarding.ServiceURL)
	assert.Equal(t, "/etc/intel_edge_node/tokens/hd-agent/access_token", cfg.JWT.AccessTokenPath)
	assert.Equal(t, 30*time.Second, cfg.UpdateInterval)
}

func TestInvalidLogLevel(t *testing.T) {
	fileName := createConfigFile(t, "v0.2.0", "invalid", "localhost", 15*time.Second, "/etc/intel_edge_node/tokens/hd-agent/access_token")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNoServiceURL(t *testing.T) {
	fileName := createConfigFile(t, "v0.2.0", "info", "", 15*time.Second, "/etc/intel_edge_node/tokens/hd-agent/access_token")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNoTokenPath(t *testing.T) {
	fileName := createConfigFile(t, "v0.2.0", "info", "localhost", 15*time.Second, "")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func FuzzNew(f *testing.F) {
	f.Add("/tmp/test_file.yaml")
	f.Fuzz(func(t *testing.T, testConfigPath string) {
		conf, err := config.New(testConfigPath)

		// Test if error received, if yes, confirm that no config has been provided
		if err != nil {
			if conf != nil {
				t.Errorf("Error %v returned but configuration is not nil!", err)
			}
		}

		// Test if no error received, if yes, confirm that configuration has been set
		if err == nil { //nolint
			if conf == nil {
				t.Error("No error received but configuration is nil!")
			}
			if conf != nil {
				if conf.Version == "" {
					t.Error("Version is not set in configuration")
				}
				if conf.LogLevel == "" {
					t.Error("LogLevel is not set in configuration")
				}
				if conf.LogLevel != "error" && conf.LogLevel != "warning" && conf.LogLevel != "info" && conf.LogLevel != "debug" {
					t.Error("LogLevel set to unsupported value")
				}
				if conf.Onboarding.ServiceURL == "" {
					t.Error("ServiceURL is not set in configuration")
				}
				if conf.UpdateInterval <= 0 {
					t.Error("UpdateInterval is not set in configuration")
				}
				if conf.JWT.AccessTokenPath == "" {
					t.Errorf("JWT.accessTokenPath is required")
				}
			}
		}
	})
}

func FuzzConfigNew(f *testing.F) {
	exampleConfigFileContents := []byte("# SPDX-FileCopyrightText: (C) 2025 Intel Corporation\n#\n# SPDX-License-Identifier: Apache-2.0\n\n---\nversion: v0.2.0\nlogLevel: info\nonboarding:\n  serviceURL: \"localhost\"\ninterval: \"30s\"\njwt:\n accessTokenPath: /etc/intel_edge_node/tokens/hd-agent/access_token\n :")
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

		conf, err := config.New(testFile.Name())
		if err != nil {
			if conf != nil {
				t.Errorf("Error %v returned but configuration is not nil!", err)
			}
		}
		if err == nil { //nolint
			if conf == nil {
				t.Error("No error returned but configuration is nil!")
			}
			if conf != nil {
				if conf.Version == "" {
					t.Error("Version is not set in configuration")
				}
				if conf.LogLevel == "" {
					t.Error("LogLevel is not set in configuration")
				}
				if conf.LogLevel != "error" && conf.LogLevel != "warning" && conf.LogLevel != "info" && conf.LogLevel != "debug" {
					t.Error("LogLevel set to unsupported value")
				}
				if conf.Onboarding.ServiceURL == "" {
					t.Error("ServiceURL is not set in configuration")
				}
				if conf.UpdateInterval <= 0 {
					t.Error("UpdateInterval is not set in configuration")
				}
				if conf.JWT.AccessTokenPath == "" {
					t.Errorf("JWT.accessTokenPath is required")
				}
			}
		}
	})
}
