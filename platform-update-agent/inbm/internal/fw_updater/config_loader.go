/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater provides the implementation for updating the firmware with dependency injection support.
package fwupdater

import (
	"github.com/spf13/afero"
)

// ConfigLoader provides a high-level interface for loading firmware configurations
// This demonstrates how to use the interface-based approach in production code
type ConfigLoader struct {
	provider PlatformConfigProvider
}

// NewConfigLoader creates a new ConfigLoader instance with real implementations
func NewConfigLoader() *ConfigLoader {
	// Create the dependency chain with real implementations
	fs := afero.NewOsFs()                            // Use the real filesystem
	fileOps := NewProductionFileSystemOperations(fs) // Use secure production version
	validator := NewGoJSONSchemaValidator()
	configReader := NewSecureConfigReader(fileOps, validator, firmwareToolInfoFilePath, firmwareToolInfoSchemaFilePath)
	provider := NewFirmwarePlatformConfigProvider(configReader)

	return &ConfigLoader{
		provider: provider,
	}
}

// NewConfigLoaderWithProvider creates a new ConfigLoader with a custom provider
// This allows for dependency injection in tests and other scenarios
func NewConfigLoaderWithProvider(provider PlatformConfigProvider) *ConfigLoader {
	return &ConfigLoader{
		provider: provider,
	}
}

// LoadPlatformConfig loads the configuration for a specific platform
func (c *ConfigLoader) LoadPlatformConfig(platformName string) (FirmwareToolInfo, error) {
	return c.provider.GetPlatformConfig(platformName)
}

// GetProvider returns the underlying platform config provider
// This can be useful for advanced use cases
func (c *ConfigLoader) GetProvider() PlatformConfigProvider {
	return c.provider
}

// ConfigLoaderFactory provides factory methods for creating ConfigLoader instances
// This demonstrates various configuration patterns
type ConfigLoaderFactory struct{}

// NewConfigLoaderFactory creates a new ConfigLoaderFactory
func NewConfigLoaderFactory() *ConfigLoaderFactory {
	return &ConfigLoaderFactory{}
}

// CreateStandardLoader creates a ConfigLoader with standard production settings
func (f *ConfigLoaderFactory) CreateStandardLoader() *ConfigLoader {
	return NewConfigLoader()
}

// CreateTestLoader creates a ConfigLoader suitable for testing
func (f *ConfigLoaderFactory) CreateTestLoader(mockProvider PlatformConfigProvider) *ConfigLoader {
	return NewConfigLoaderWithProvider(mockProvider)
}

// CreateCustomLoader creates a ConfigLoader with custom filesystem and validation
func (f *ConfigLoaderFactory) CreateCustomLoader(fs afero.Fs, configPath, schemaPath string) *ConfigLoader {
	fileOps := NewAferoFileSystemOperations(fs)
	validator := NewGoJSONSchemaValidator()
	configReader := NewSecureConfigReader(fileOps, validator, configPath, schemaPath)
	provider := NewFirmwarePlatformConfigProvider(configReader)

	return &ConfigLoader{
		provider: provider,
	}
}

// CreateMemoryLoader creates a ConfigLoader that operates entirely in memory
// This is useful for testing and scenarios where files are provided as byte arrays
func (f *ConfigLoaderFactory) CreateMemoryLoader(configData, schemaData []byte) *ConfigLoader {
	// Create an in-memory filesystem
	fs := afero.NewMemMapFs()

	// Write the provided data to the in-memory filesystem
	err := afero.WriteFile(fs, firmwareToolInfoFilePath, configData, 0644)
	if err != nil {
		return nil
	}
	err = afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, schemaData, 0644)
	if err != nil {
		return nil
	}

	return f.CreateCustomLoader(fs, firmwareToolInfoFilePath, firmwareToolInfoSchemaFilePath)
}
