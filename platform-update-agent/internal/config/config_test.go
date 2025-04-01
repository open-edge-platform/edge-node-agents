// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const logLevel = "logLevel"
const metadataPath = "./build/sample/metadata"
const accessTokenPath = "/etc/edge-node/platform-update-agent/access_token" // #nosec G101
const insecureSkipVerify = true
const releaseServiceFQDN = "https://test-release-service-fqdn.com"

const inbcLogPath = "path/to/inbc/logs"

// helper function that will create a temporary YAML with the provided parameters for testing purposes
func createConfigFile(t *testing.T, guid, logLevel, updateServiceURL, metadataPath, inbcLogPath string, insecureSkipVerify bool, accessTokenPath, rsFQDN string) string { //nolint:unparam
	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)
	defer f.Close()

	newConfig := config.Config{
		Version:          "v0.0.0",
		GUID:             guid,
		LogLevel:         logLevel,
		UpdateServiceURL: updateServiceURL,
		MetadataPath:     metadataPath,
		JWT: config.JWT{
			AccessTokenPath: accessTokenPath,
		},
		INBCLogsPath:       inbcLogPath,
		ReleaseServiceFQDN: rsFQDN,
	}

	file, err := yaml.Marshal(newConfig)
	require.Nil(t, err)

	_, err = f.Write(file)
	require.Nil(t, err)

	err = f.Close()
	require.Nil(t, err)
	return f.Name()
}

// helper function that will create a temporary YAML with the provided parameters for testing purposes
func createConfigFileWithNewFields(t *testing.T, guid, logLevel, updateServiceURL, metadataPath, inbcLogPath string, insecureSkipVerify bool, accessTokenPath string, immediateDownloadWindow, downloadWindow time.Duration, rsFQDN string) string {
	f, err := os.CreateTemp("", "test_config")
	require.Nil(t, err)
	defer f.Close()

	newConfig := config.Config{
		Version:          "v0.0.0",
		GUID:             guid,
		LogLevel:         logLevel,
		UpdateServiceURL: updateServiceURL,
		MetadataPath:     metadataPath,
		JWT: config.JWT{
			AccessTokenPath: accessTokenPath,
		},
		INBCLogsPath:            inbcLogPath,
		ImmediateDownloadWindow: immediateDownloadWindow,
		DownloadWindow:          downloadWindow,
		ReleaseServiceFQDN:      rsFQDN,
	}

	file, err := yaml.Marshal(newConfig)
	require.Nil(t, err)

	_, err = f.Write(file)
	require.Nil(t, err)

	err = f.Close()
	require.Nil(t, err)
	return f.Name()
}

func Test_Config_CustomValuesForNewFields(t *testing.T) {
	guid := "6B29FC40-CA47-AAAA-B31D-00DD010662DA"
	updateServiceUrl := "https:/www.sampleUpdate.com:8080"

	customImmediateWindow := 45 * time.Minute
	customDownloadWindow := 3 * time.Hour

	fileName := createConfigFileWithNewFields(t, guid, logLevel, updateServiceUrl, metadataPath, inbcLogPath, insecureSkipVerify, accessTokenPath, customImmediateWindow, customDownloadWindow, releaseServiceFQDN)

	defer os.Remove(fileName)

	cfg, err := config.New(fileName)

	require.Nil(t, err)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, updateServiceUrl, cfg.UpdateServiceURL)
	assert.Equal(t, metadataPath, cfg.MetadataPath)
	assert.Equal(t, accessTokenPath, cfg.JWT.AccessTokenPath)
	assert.Equal(t, inbcLogPath, cfg.INBCLogsPath)
	assert.Equal(t, releaseServiceFQDN, cfg.ReleaseServiceFQDN)

	// Verify custom values are set
	assert.Equal(t, customImmediateWindow, cfg.ImmediateDownloadWindow, "ImmediateDownloadWindow should be set to custom value")
	assert.Equal(t, customDownloadWindow, cfg.DownloadWindow, "DownloadWindow should be set to custom value")
}

func Test_Config_InvalidNewFields(t *testing.T) {
	guid := "123e4567-e89b-12d3-a456-426614174000"
	updateServiceUrl := "https://updateservice.example.com"

	// Set invalid negative durations
	invalidImmediateWindow := -5 * time.Minute
	invalidDownloadWindow := -1 * time.Hour

	fileName := createConfigFileWithNewFields(t, guid, logLevel, updateServiceUrl, metadataPath, inbcLogPath, insecureSkipVerify, accessTokenPath, invalidImmediateWindow, invalidDownloadWindow, releaseServiceFQDN)

	defer os.Remove(fileName)

	cfg, err := config.New(fileName)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "immediateDownloadWindow cannot be negative", err.Error())
}

func Test_Config_AllVariablesShouldBeAssignedCorrectlyInStruct(t *testing.T) {

	guid := "6B29FC40-CA47-AAAA-B31D-00DD010662DA"
	updateServiceUrl := "https:/www.sampleUpdate.com:8080"

	fileName := createConfigFile(t, guid, logLevel, updateServiceUrl, metadataPath, inbcLogPath, insecureSkipVerify, accessTokenPath, releaseServiceFQDN)

	defer os.Remove(fileName)

	cfg, err := config.New(fileName)

	require.Nil(t, err)
	assert.Equal(t, guid, cfg.GUID)
	assert.Equal(t, logLevel, cfg.LogLevel)
	assert.Equal(t, updateServiceUrl, cfg.UpdateServiceURL)
	assert.Equal(t, metadataPath, cfg.MetadataPath)
	assert.Equal(t, accessTokenPath, cfg.JWT.AccessTokenPath)
	assert.Equal(t, releaseServiceFQDN, cfg.ReleaseServiceFQDN)

	assert.Equal(t, inbcLogPath, cfg.INBCLogsPath)

}

// test that creates a config with a file that does not exist and asserts that an error is returned

func Test_NewConfig_WhenFilePathIsInvalidNoConfigShouldBeReturned(t *testing.T) {

	config, err := config.New("./this/path/doesnt/exist")
	assert.NotNil(t, err)
	assert.Nil(t, config)
}

func Test_NewConfig_NoConfigReturnedWhenSymLinkIsInputtedAsFilePath(t *testing.T) {

	symlinkTempFile := "/tmp/symlink_temp.txt"
	file, err := os.CreateTemp("", "config_temp.txt")
	assert.NoError(t, err)
	defer file.Close()
	err = os.Symlink(file.Name(), symlinkTempFile)
	require.Nil(t, err)

	defer os.Remove(file.Name())
	defer os.Remove(symlinkTempFile)

	cfg, err := config.New(symlinkTempFile)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
}

// test that creates a config with invalid yaml syntax and asserts that an error is returned
func Test_NewConfig_WhenYAMLIsInvalidNoConfigShouldBeReturned(t *testing.T) {

	file, err := os.CreateTemp("", "config_temp")
	require.Nil(t, err)
	defer file.Close()

	defer os.Remove(file.Name())

	_, err = file.WriteString("this is invalid YAML layout")
	require.Nil(t, err)

	config, err := config.New(file.Name())
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// test that creates a config without an Update Service URL and asserts that the correct error is returned

func Test_NewConfig_WhenUpdateServiceURLIsInvalidErrorShouldBeReturned(t *testing.T) {

	fileName := createConfigFile(t, "sample-guid", logLevel, "", metadataPath, inbcLogPath, insecureSkipVerify, accessTokenPath, releaseServiceFQDN)
	defer os.Remove(fileName)

	config, err := config.New(fileName)
	assert.Nil(t, config)
	assert.NotNil(t, err)
	assert.Equal(t, "updateServiceURL is required", err.Error())

}

// test that creates a config without a GUID and asserts that the correct error is returned

func Test_NewConfig_WhenGUIDIsInvalidErrorShouldBeReturned(t *testing.T) {

	fileName := createConfigFile(t, "", logLevel, "sample-updateserviceURL", metadataPath, inbcLogPath, insecureSkipVerify, accessTokenPath, releaseServiceFQDN)
	defer os.Remove(fileName)

	config, err := config.New(fileName)
	assert.Nil(t, config)
	assert.NotNil(t, err)
	assert.Equal(t, "GUID is required", err.Error())

}

func Test_NewConfig_WhenTokenErrorShouldBeReturned(t *testing.T) {

	fileName := createConfigFile(t, "sample-guid", logLevel, "sample-updateserviceURL", metadataPath, inbcLogPath, insecureSkipVerify, "", releaseServiceFQDN)
	defer os.Remove(fileName)

	config, err := config.New(fileName)
	assert.Nil(t, config)
	assert.NotNil(t, err)
	assert.Equal(t, "JWT is required", err.Error())

}
