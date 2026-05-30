/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"testing"
	"time"

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

func TestIsTrustedRepository(t *testing.T) {
	config := &Configurations{
		OSUpdater: OSUpdaterConfig{
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
		OSUpdater: OSUpdaterConfig{
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
		OSUpdater: OSUpdaterConfig{
			ProceedWithoutRollback: false,
		},
	}

	// Call IsProceedWithoutRollback
	result := IsProceedWithoutRollback(config)

	// Assertions
	assert.False(t, result, "Expected rollback to be disallowed")
}

func TestGetInstallIdleTimeout(t *testing.T) {
	t.Run("empty value disabled", func(t *testing.T) {
		cfg := &Configurations{OSUpdater: OSUpdaterConfig{}}
		idleTimeout, err := GetInstallIdleTimeout(cfg)
		assert.NoError(t, err)
		assert.Equal(t, 60*time.Minute, idleTimeout)
	})

	t.Run("valid duration", func(t *testing.T) {
		cfg := &Configurations{OSUpdater: OSUpdaterConfig{InstallIdleTimeoutSeconds: 2700}}
		idleTimeout, err := GetInstallIdleTimeout(cfg)
		assert.NoError(t, err)
		assert.Equal(t, 45*time.Minute, idleTimeout)
	})

	t.Run("invalid duration", func(t *testing.T) {
		cfg := &Configurations{OSUpdater: OSUpdaterConfig{InstallIdleTimeoutSeconds: -1}}
		_, err := GetInstallIdleTimeout(cfg)
		assert.ErrorContains(t, err, "invalid os_updater.installIdleTimeoutSeconds")
	})

	t.Run("negative duration", func(t *testing.T) {
		cfg := &Configurations{OSUpdater: OSUpdaterConfig{InstallIdleTimeoutSeconds: -1}}
		_, err := GetInstallIdleTimeout(cfg)
		assert.ErrorContains(t, err, "must be >= 0")
	})
}
