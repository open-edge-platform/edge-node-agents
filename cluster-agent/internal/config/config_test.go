// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func createConfigFile(t *testing.T, version, serverURL, guid string, heartbeat time.Duration, accessTokenPath string) string {
	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)

	c := config.Config{
		Version:    version,
		ServerAddr: serverURL,
		Heartbeat:  heartbeat,
		GUID:       guid,
		JWT: config.JWT{
			AccessTokenPath: accessTokenPath,
		},
	}

	file, err := yaml.Marshal(c)
	require.Nil(t, err)

	_, err = f.Write(file)
	require.Nil(t, err)

	err = f.Close()
	require.Nil(t, err)
	return f.Name()
}

func TestValidConfig(t *testing.T) {
	version := "v0.2.0"
	serverURL := "cluster-orchestrator.intel.com:12345"
	guid := "0dbfbf99-90c5-400d-be08-67cea05cbc03"

	fileName := createConfigFile(t, version, serverURL, guid, 10*time.Second, "/etc/intel_edge_node/tokens/cluster-agent")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.Nil(t, err)
	assert.Equal(t, version, cfg.Version)
	assert.Equal(t, serverURL, cfg.ServerAddr)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, "/etc/intel_edge_node/tokens/cluster-agent", cfg.JWT.AccessTokenPath)
}

func TestInvalidConfigPath(t *testing.T) {
	cfg, err := config.New("non_existent_path")
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestInvalidConfigContent(t *testing.T) {
	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)
	defer os.Remove(f.Name()) // clean up

	_, err = f.WriteString("not a yaml")
	require.Nil(t, err)

	cfg, err := config.New(f.Name())
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestInvalidConfigNoAddr(t *testing.T) {
	fileName := createConfigFile(t, "0.2.0", "1234", "", 10*time.Second, "/accessTokenPath")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestInvalidConfigNoGuid(t *testing.T) {
	fileName := createConfigFile(t, "0.2.0", "", "abc.com:123", 10*time.Second, "/accessTokenPath")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestInvalidToken(t *testing.T) {
	fileName := createConfigFile(t, "0.2.0", "1234", "abc.com:123", 10*time.Second, "")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestInvalidConfigHeartbeat0s(t *testing.T) {
	fileName := createConfigFile(t, "0.2.0", "1234", "abc.com:123", 0*time.Second, "/accessTokenPath")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.Nil(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, 10*time.Second, cfg.Heartbeat)
}

func TestConfigNoLogLevel(t *testing.T) {
	fileName := createConfigFile(t, "0.2.0", "1234", "abc.com:123", 10*time.Second, "/accessTokenPath")
	defer os.Remove(fileName) // clean up

	cfg, err := config.New(fileName)
	require.Nil(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "info", cfg.LogLevel)
}

func TestConfigSymlinkFile(t *testing.T) {
	version := "v0.2.0"
	serverURL := "cluster-orchestrator.intel.com:12345"
	guid := "0dbfbf99-90c5-400d-be08-67cea05cbc03"

	fileName := createConfigFile(t, version, serverURL, guid, 10*time.Second, "/etc/intel_edge_node/tokens/cluster-agent")
	defer os.Remove(fileName)

	symlinkConfig := "/tmp/symlink_config.txt"
	defer os.Remove(symlinkConfig)
	err := os.Symlink(fileName, symlinkConfig)
	require.Nil(t, err)

	cfg, err := config.New(symlinkConfig)
	require.NotNil(t, err)
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
		if err == nil {
			if conf == nil {
				t.Error("No error received but configuration is nil!")
			}
			if conf != nil {
				if conf.Version == "" {
					t.Error("Version is not set in configuration")
				}
				if conf.GUID == "" {
					t.Error("GUID is not set in configuration")
				}
				if conf.GUID != "error" && conf.GUID != "warning" && conf.GUID != "info" && conf.GUID != "debug" {
					t.Error("GUID set to unsupported value")
				}
				if conf.ServerAddr == "" {
					t.Error("ServiceAddr is not set in configuration")
				}
				if conf.JWT.AccessTokenPath == "" {
					t.Errorf("JWT.accessTokenPath is required")
				}
			}
		}
	})
}

func FuzzConfigNew(f *testing.F) {
	clusteragentConfigFileContents := []byte("# SPDX-FileCopyrightText: (C) 2025 Intel Corporation\n#\n# SPDX-License-Identifier: Apache-2.0\n\n---\nversion: \"v0.2.0\"\n# Globally unique identifier read from motherboard. Might be obtained with:\n# sudo cat /sys/class/dmi/id/product_uuid\nGUID: \"00000000-0000-0000-0000-000000000000\"\n# Connection parameters\nclusterOrchestratorURL: \"localhost:443\"\n")
	f.Add(clusteragentConfigFileContents)
	f.Fuzz(func(t *testing.T, testConfigFileContents []byte) {
		testFile, err := os.CreateTemp("", "cluster-agent.yaml")
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
		if err == nil {
			if conf == nil {
				t.Error("No error returned but configuration is nil!")
			}
			if conf != nil {
				if conf.GUID == "" {
					t.Error("Guid is not set in configuration")
				}
				if conf.ServerAddr == "" {
					t.Error("ServerAddr is not set in configuration")
				}
				if conf.JWT.AccessTokenPath == "" {
					t.Errorf("JWT.accessTokenPath is required")
				}
			}
		}
	})
}
