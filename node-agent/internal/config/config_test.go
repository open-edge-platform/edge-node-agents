// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
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
const testMetricsEnabled = true
const testMetricsEndpoint = "unix:///run/agent/metrics.sock"
const testMetricsHeartbeatInterval = 10 * time.Second

// Disabling lint locally - G101: Potential hardcoded credentials (gosec)
const testAuthAccessTokenPath = "/etc/intel_edge_node/tokens/node-agent"  // #nosec G101
const testAuthClientCredsPath = "/etc/intel_edge_node/client-credentials" // #nosec G101

var testAuthTokenClients = []string{"one", "two", "three"}

var testStatusClients = []string{"one", "two", "three"}

var testStatusOutboundClients = []string{"one", "two", "three"}

var testNetworkStatusInterval = 6 * testOnboardingHeartbeatInterval

var testStatusNetworkEndpoints = []config.NetworkEndpoint{{Name: "one", URL: "http://one.com"}}

// Function to create a test configuration file with injected configurations
func createConfigFile(t *testing.T, testGUID string, onboardingServiceURL string, logLevel string, rsTokenURL string,
	accessTokenURL string, accessTokenPath string, heartbeatInterval time.Duration) string {

	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)

	c := config.NodeAgentConfig{
		Version:  "v0.1.0",
		LogLevel: logLevel,
		GUID:     testGUID,
		Onboarding: config.ConfigOnboarding{
			Enabled:           testOnboardingEnabled,
			ServiceURL:        onboardingServiceURL,
			HeartbeatInterval: heartbeatInterval,
		},
		Status: config.ConfigStatus{
			Endpoint:         testStatusEndpoint,
			ServiceClients:   testStatusClients,
			OutboundClients:  testStatusOutboundClients,
			NetworkEndpoints: testStatusNetworkEndpoints,
		},
		Auth: config.ConfigAuth{
			AccessTokenURL:  accessTokenURL,
			RsTokenURL:      rsTokenURL,
			AccessTokenPath: accessTokenPath,
			ClientCredsPath: testAuthClientCredsPath,
			TokenClients:    testAuthTokenClients,
		},
		Metrics: config.ConfigMetrics{
			Enabled:  testMetricsEnabled,
			Endpoint: testMetricsEndpoint,
			Interval: testMetricsHeartbeatInterval,
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

func getExpectedConfig(logLevel string, heartbeatInterval time.Duration, networkInterval time.Duration) config.NodeAgentConfig {
	return config.NodeAgentConfig{
		Version:  "v0.1.0",
		LogLevel: logLevel,
		GUID:     testGUID,
		Onboarding: config.ConfigOnboarding{
			Enabled:           testOnboardingEnabled,
			ServiceURL:        testOnboardingServiceURL,
			HeartbeatInterval: heartbeatInterval,
		},
		Status: config.ConfigStatus{
			Endpoint:              testStatusEndpoint,
			ServiceClients:        testStatusClients,
			OutboundClients:       testStatusOutboundClients,
			NetworkStatusInterval: networkInterval,
			NetworkEndpoints:      testStatusNetworkEndpoints,
		},
		Auth: config.ConfigAuth{
			AccessTokenURL:  testAuthAccessTokenURL,
			AccessTokenPath: testAuthAccessTokenPath,
			RsTokenURL:      testAuthRsTokenURL,
			ClientCredsPath: testAuthClientCredsPath,
			TokenClients:    testAuthTokenClients,
		},
		Metrics: config.ConfigMetrics{
			Enabled:  testMetricsEnabled,
			Endpoint: testMetricsEndpoint,
			Interval: testMetricsHeartbeatInterval,
		},
	}
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

// Verify an existing configuration file read
func TestConfigFileExists(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	expectedCfg := getExpectedConfig(testLogLevel, testOnboardingHeartbeatInterval, testNetworkStatusInterval)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, expectedCfg, *cfg)
}

// Expect error if critical configuration not present
func TestConfigFileNoOnboardingServiceURL(t *testing.T) {
	fileName := createConfigFile(t, testGUID, "", testLogLevel, testAuthRsTokenURL, testAuthAccessTokenURL,
		testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigFileNoGuid(t *testing.T) {
	fileName := createConfigFile(t, "", testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigFileNoRsTokenURL(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, "", testAuthAccessTokenURL,
		testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigFileNoAccessTokenURL(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		"", testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigFileNoAccessTokenPath(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, "", 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// No error expected and configuration file returned
func TestConfigFileNoLogLevel(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, "", testAuthRsTokenURL, testAuthAccessTokenURL,
		testAuthAccessTokenPath, 0*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	expectedCfg := getExpectedConfig("", testOnboardingHeartbeatInterval, testNetworkStatusInterval)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, expectedCfg, *cfg)
}

func TestConfigFileOnboardingIntervalGreaterThanZero(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, testAuthAccessTokenPath, 5*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	expectedCfg := getExpectedConfig(testLogLevel, 5*time.Second, 30*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, expectedCfg, *cfg)
}

func TestConfigFileOnboardingIntervalGreaterThanOneMinute(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, testAuthAccessTokenPath, 15*time.Second)
	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	expectedCfg := getExpectedConfig(testLogLevel, 15*time.Second, testNetworkStatusInterval)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, expectedCfg, *cfg)
}

func TestConfigFileSymlinkFile(t *testing.T) {
	fileName := createConfigFile(t, testGUID, testOnboardingServiceURL, testLogLevel, testAuthRsTokenURL,
		testAuthAccessTokenURL, testAuthAccessTokenPath, 0*time.Second)
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
