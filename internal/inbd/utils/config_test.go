/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		configContent := `
        {
            "os_updater": {
                "trustedRepositories": ["https://example.com/repo1", "https://example.com/repo2"]
            }
        }`

		err := afero.WriteFile(fs, "config.json", []byte(configContent), 0644)
		assert.NoError(t, err)

		config, err := LoadConfig(fs, "config.json")
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, 2, len(config.OSUpdater.TrustedRepositories))
		assert.Equal(t, "https://example.com/repo1", config.OSUpdater.TrustedRepositories[0])
		assert.Equal(t, "https://example.com/repo2", config.OSUpdater.TrustedRepositories[1])
	})

	t.Run("file not found", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		config, err := LoadConfig(fs, "nonexistent.json")
		assert.Error(t, err)
		assert.Nil(t, config)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		configContent := `
        {
            "os_updater": {
                "trustedRepositories": ["https://example.com/repo1", "https://example.com/repo2
            }
        }`
		err := afero.WriteFile(fs, "config.json", []byte(configContent), 0644)
		assert.NoError(t, err)

		config, err := LoadConfig(fs, "config.json")
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

func TestLoadConfig_IntelManageabilityConf(t *testing.T) {
	fs := afero.NewMemMapFs()
	configContent := `
{
    "os_updater": {
        "trustedRepositories": [],
        "proceedWithoutRollback": true
    },
    "luks":{
        "volumePath": "/var/intel-manageability/secret.img",
        "mapperName": "intel-manageability-secret",
        "mountPoint": "/etc/intel-manageability/secret",
        "passwordLength": 20,
        "size": 32,
        "useTPM": true,
        "user": "root",
        "group": "root"
    }
}`
	err := afero.WriteFile(fs, "intel_manageability.conf", []byte(configContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(fs, "intel_manageability.conf")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 0, len(config.OSUpdater.TrustedRepositories))
	assert.True(t, config.OSUpdater.ProceedWithoutRollback)
	assert.Equal(t, "/var/intel-manageability/secret.img", config.LUKS.VolumePath)
	assert.Equal(t, "intel-manageability-secret", config.LUKS.MapperName)
	assert.Equal(t, "/etc/intel-manageability/secret", config.LUKS.MountPoint)
	assert.Equal(t, 20, config.LUKS.PasswordLength)
	assert.Equal(t, 32, config.LUKS.Size)
	assert.True(t, config.LUKS.UseTPM)
	assert.Equal(t, "root", config.LUKS.User)
	assert.Equal(t, "root", config.LUKS.Group)
}

func TestIsTrustedRepository(t *testing.T) {
	config := &Configurations{
		OSUpdater: struct {
			TrustedRepositories    []string `json:"trustedRepositories"`
			ProceedWithoutRollback bool     `json:"proceedWithoutRollback"`
		}{
			TrustedRepositories: []string{
				"https://example.com/repo1",
				"https://example.com/repo2"},
		},
	}

	t.Run("trusted repository", func(t *testing.T) {
		url := "https://example.com/repo1/some/path"
		result := IsTrustedRepository(url, config)
		assert.True(t, result)
	})

	t.Run("untrusted repository", func(t *testing.T) {
		url := "https://untrusted.com/repo"
		result := IsTrustedRepository(url, config)
		assert.False(t, result)
	})

	t.Run("empty URL", func(t *testing.T) {
		url := ""
		result := IsTrustedRepository(url, config)
		assert.False(t, result)
	})
}

func TestIsProceedWithoutRollback_True(t *testing.T) {
	// Create a configuration where ProceedWithoutRollback is true
	config := &Configurations{
		OSUpdater: struct {
			TrustedRepositories    []string `json:"trustedRepositories"`
			ProceedWithoutRollback bool     `json:"proceedWithoutRollback"`
		}{
			ProceedWithoutRollback: true,
		},
	}

	// Call IsProceedWithoutRollback
	result := IsProceedWithoutRollback(config)

	// Assertions
	assert.True(t, result, "Expected rollback to be allowed")
}

func TestIsProceedWithoutRollback_False(t *testing.T) {
	// Create a configuration where ProceedWithoutRollback is false
	config := &Configurations{
		OSUpdater: struct {
			TrustedRepositories    []string `json:"trustedRepositories"`
			ProceedWithoutRollback bool     `json:"proceedWithoutRollback"`
		}{
			ProceedWithoutRollback: false,
		},
	}

	// Call IsProceedWithoutRollback
	result := IsProceedWithoutRollback(config)

	// Assertions
	assert.False(t, result, "Expected rollback to be disallowed")
}
