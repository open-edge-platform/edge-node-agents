// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

const testOnboardingServiceURL = "http://unit.testing.com"
const testOnboardingEnabled = true
const testOnboardingHeartbeatInterval = 10 * time.Second
const testLogLevel = "info"
const testGUID = "TEST-GUID-TEST-GUID"
const testAuthAccessTokenURL = "keycloak.test"
const testAuthRsTokenURL = "token-provider.test"
const testStatusEndpoint = "/run/agent/test.sock"

// Disabling lint locally - G101: Potential hardcoded credentials (gosec)
const testAuthAccessTokenPath = "/etc/intel_edge_node/tokens/node-agent"  // #nosec G101
const testAuthClientCredsPath = "/etc/intel_edge_node/client-credentials" // #nosec G101

var testAuthTokenClients = []string{"one", "two", "three"}

var testStatusClients = []string{"one", "two", "three"}

var testStatusOutboundClients = []string{"one", "two", "three"}

var testNetworkStatusInterval = 6 * testOnboardingHeartbeatInterval

var testStatusNetworkEndpoints = []config.NetworkEndpoint{{Name: "one", URL: "http://one.com"}}

// Function to create a test configuration file with injected configurations
func createConfigFile(t *testing.T, testGUID string, onboardingServiceURL string, logLevel string) string {

	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)

	c := config.NodeAgentConfig{
		Version:  "v0.1.0",
		LogLevel: logLevel,
		GUID:     testGUID,
		Onboarding: config.ConfigOnboarding{
			Enabled:    testOnboardingEnabled,
			ServiceURL: onboardingServiceURL,
		},
		Status: config.ConfigStatus{
			Endpoint:         testStatusEndpoint,
			ServiceClients:   testStatusClients,
			OutboundClients:  testStatusOutboundClients,
			NetworkEndpoints: testStatusNetworkEndpoints,
		},
		Auth: config.ConfigAuth{
			AccessTokenURL:  testAuthAccessTokenURL,
			RsTokenURL:      testAuthRsTokenURL,
			AccessTokenPath: testAuthAccessTokenPath,
			ClientCredsPath: testAuthClientCredsPath,
			TokenClients:    testAuthTokenClients,
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

// Expect error when configuration file does not exist
func TestNoConfigFileExists(t *testing.T) {
	cfg, err := config.New("random_config_file")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// Expect error when not config path configured
func TestNoConfigFilePath(t *testing.T) {
	cfg, err := config.New("")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

var expectedCfg = config.NodeAgentConfig{
	Version:  "v0.1.0",
	LogLevel: testLogLevel,
	GUID:     testGUID,
	Onboarding: config.ConfigOnboarding{
		Enabled:           testOnboardingEnabled,
		ServiceURL:        testOnboardingServiceURL,
		HeartbeatInterval: testOnboardingHeartbeatInterval,
	},
	Status: config.ConfigStatus{
		Endpoint:              testStatusEndpoint,
		ServiceClients:        testStatusClients,
		OutboundClients:       testStatusOutboundClients,
		NetworkStatusInterval: testNetworkStatusInterval,
		NetworkEndpoints:      testStatusNetworkEndpoints,
	},
	Auth: config.ConfigAuth{
		AccessTokenURL:  testAuthAccessTokenURL,
		AccessTokenPath: testAuthAccessTokenPath,
		RsTokenURL:      testAuthRsTokenURL,
		ClientCredsPath: testAuthClientCredsPath,
		TokenClients:    testAuthTokenClients,
	},
}

// Verify an existing configuration file read
func TestConfigFileExists(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, expectedCfg, *cfg)
}

// Expect error if critical configuration not present
func TestConfigFileNoOnboardingServiceURL(t *testing.T) {
	fileName := createConfigFile(t, testGUID, "", testLogLevel)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TesConfigFiletNoGuid(t *testing.T) {
	fileName := createConfigFile(t, "", testOnboardingServiceURL, testLogLevel)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// No error expected
func TesConfigFiletNoLogLevel(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, "")
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestConfigFileSymlinkFile(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel)
	defer os.Remove(fileName)

	symlinkConfig := "/tmp/symlink_config.txt"
	defer os.Remove(symlinkConfig)
	err := os.Symlink(fileName, symlinkConfig)
	require.Nil(t, err)

	cfg, err := config.New(symlinkConfig)
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

// Fuzz test
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
				if conf.Onboarding.Enabled != true && conf.Onboarding.Enabled != false {
					t.Error("OnboardingEnabled set to unsupported value")
				}
				if conf.Onboarding.HeartbeatInterval == 0 {
					t.Error("OnboardingHeartbeatInterval set to unsupported value")
				}
			}
		}
	})
}

func FuzzConfigNew(f *testing.F) {
	exampleConfigFileContents := []byte("# SPDX-FileCopyrightText: (C) 2025 Intel Corporation\n\n# SPDX-License-Identifier: Apache-2.0\n\n---\nversion: v0.4.0\nlogLevel: info\nGUID: 'aaaaaaaa-0000-1111-2222-bbbbbbbbcccc'\nonboarding:\nenabled: true\nserviceURL: example.test.orch.intel.com:443\nheartbeatInterval: 10s\ntls:\n certPath: /etc/edge-node/node/certs\n keyPath: /etc/edge-node/node/.keys\n mTLSEnabled: false\n orgName: \"Intel Corporation\"\n caCert: /usr/local/share/ca-certificates/orch-ca.crt\nvault:\n serviceURL: vault.test.orch.intel.com\n role: \"kind-dot-internal\"\n path: \"pki_int_edge_node\"\nprovisioning:\nenabled: true\n serviceURL: provision.local.com")
	f.Add(exampleConfigFileContents)
	f.Fuzz(func(t *testing.T, testConfigFileContents []byte) {
		testFile, err := os.CreateTemp("", "node-agent.yaml")
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
				if conf.LogLevel == "" {
					t.Error("LogLevel is not set in configuration")
				}
				if conf.LogLevel != "error" && conf.LogLevel != "warning" && conf.LogLevel != "info" && conf.LogLevel != "debug" {
					t.Error("LogLevel set to unsupported value")
				}
				if conf.Onboarding.ServiceURL == "" {
					t.Error("ServiceURL is not set in configuration")
				}
				if conf.Onboarding.Enabled != true && conf.Onboarding.Enabled != false {
					t.Error("OnboardingEnabled set to unsupported value")
				}
				if conf.Onboarding.HeartbeatInterval == 0 {
					t.Error("OnboardingHeartbeatInterval set to unsupported value")
				}
			}
		}
	})
}
