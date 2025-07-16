/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestConfigLoader_StandardLoader tests the standard production loader
func TestConfigLoader_StandardLoader(t *testing.T) {
	// This test demonstrates how the standard loader would be used in production
	// Note: This test is conceptual since it would require actual files
	t.Skip("Skipping standard loader test - requires actual filesystem setup")

	// In a real scenario:
	// loader := NewConfigLoader()
	// config, err := loader.LoadPlatformConfig("Some Platform")
	// assert.NoError(t, err)
	// assert.Equal(t, "Some Platform", config.Name)
}

// TestConfigLoader_WithMockProvider demonstrates dependency injection for testing
func TestConfigLoader_WithMockProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := NewMockPlatformConfigProvider()

	// Set up expected behavior
	expectedConfig := FirmwareToolInfo{
		Name:             "Test Platform",
		BiosVendor:       "Test Vendor",
		FirmwareFileType: "bin",
		FirmwareTool:     "/usr/bin/test-tool",
	}
	mockProvider.SetPlatformConfig("Test Platform", expectedConfig)

	// Create loader with mock provider
	loader := NewConfigLoaderWithProvider(mockProvider)

	// Test the functionality
	config, err := loader.LoadPlatformConfig("Test Platform")
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)

	// Verify the mock was called correctly
	assert.Equal(t, 1, mockProvider.GetConfigCallCount())
	assert.Equal(t, "Test Platform", mockProvider.GetLastConfigCall())
}

// TestConfigLoader_MockProviderError tests error handling with mocked dependencies
func TestConfigLoader_MockProviderError(t *testing.T) {
	// Create a mock provider with error
	mockProvider := NewMockPlatformConfigProvider()
	mockProvider.SetPlatformError("Error Platform", errors.New("simulated error"))

	// Create loader with mock provider
	loader := NewConfigLoaderWithProvider(mockProvider)

	// Test error handling
	_, err := loader.LoadPlatformConfig("Error Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")

	// Verify the mock tracked the call
	assert.Equal(t, 1, mockProvider.GetConfigCallCount())
}

// TestConfigLoader_PlatformNotFound tests platform not found scenario
func TestConfigLoader_PlatformNotFound(t *testing.T) {
	// Create a mock provider without any configurations
	mockProvider := NewMockPlatformConfigProvider()

	// Create loader with mock provider
	loader := NewConfigLoaderWithProvider(mockProvider)

	// Test platform not found
	_, err := loader.LoadPlatformConfig("Nonexistent Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform Nonexistent Platform not found")
}

// TestConfigLoaderFactory_CreateStandardLoader tests the factory pattern
func TestConfigLoaderFactory_CreateStandardLoader(t *testing.T) {
	factory := NewConfigLoaderFactory()

	// This would create a standard loader - conceptual test
	loader := factory.CreateStandardLoader()
	assert.NotNil(t, loader)
	assert.NotNil(t, loader.GetProvider())
}

// TestConfigLoaderFactory_CreateTestLoader tests factory method for test scenarios
func TestConfigLoaderFactory_CreateTestLoader(t *testing.T) {
	factory := NewConfigLoaderFactory()

	// Create mock provider
	mockProvider := NewMockPlatformConfigProvider()
	expectedConfig := FirmwareToolInfo{
		Name:       "Factory Test Platform",
		BiosVendor: "Factory Vendor",
	}
	mockProvider.SetPlatformConfig("Factory Test Platform", expectedConfig)

	// Create test loader
	loader := factory.CreateTestLoader(mockProvider)

	// Test functionality
	config, err := loader.LoadPlatformConfig("Factory Test Platform")
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}

// TestConfigLoaderFactory_CreateCustomLoader tests custom loader creation
func TestConfigLoaderFactory_CreateCustomLoader(t *testing.T) {
	factory := NewConfigLoaderFactory()

	// Create in-memory filesystem with test data
	fs := afero.NewMemMapFs()

	configData := []byte(`{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Custom Platform",
					"bios_vendor": "Custom Vendor",
					"firmware_file_type": "custom"
				}
			]
		}
	}`)

	schemaData := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"firmware_component": {
				"type": "object",
				"properties": {
					"firmware_products": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"bios_vendor": {"type": "string"},
								"firmware_file_type": {"type": "string"}
							},
							"required": ["name", "bios_vendor", "firmware_file_type"]
						}
					}
				},
				"required": ["firmware_products"]
			}
		},
		"required": ["firmware_component"]
	}`)

	// Write test data to filesystem
	configPath := "/test/custom_config.json"
	schemaPath := "/test/custom_schema.json"
	err := afero.WriteFile(fs, configPath, configData, 0644)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, schemaPath, schemaData, 0644)
	assert.NoError(t, err)

	// Create custom loader
	loader := factory.CreateCustomLoader(fs, configPath, schemaPath)

	// Test functionality
	config, err := loader.LoadPlatformConfig("Custom Platform")
	assert.NoError(t, err)
	assert.Equal(t, "Custom Platform", config.Name)
	assert.Equal(t, "Custom Vendor", config.BiosVendor)
	assert.Equal(t, "custom", config.FirmwareFileType)
}

// TestConfigLoaderFactory_CreateMemoryLoader tests the memory-based loader
func TestConfigLoaderFactory_CreateMemoryLoader(t *testing.T) {
	factory := NewConfigLoaderFactory()

	configData := []byte(`{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Memory Platform",
					"bios_vendor": "Memory Vendor",
					"firmware_file_type": "mem"
				}
			]
		}
	}`)

	schemaData := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"firmware_component": {
				"type": "object",
				"properties": {
					"firmware_products": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"bios_vendor": {"type": "string"},
								"firmware_file_type": {"type": "string"}
							},
							"required": ["name", "bios_vendor", "firmware_file_type"]
						}
					}
				},
				"required": ["firmware_products"]
			}
		},
		"required": ["firmware_component"]
	}`)

	// Create memory loader
	loader := factory.CreateMemoryLoader(configData, schemaData)

	// Test functionality
	config, err := loader.LoadPlatformConfig("Memory Platform")
	assert.NoError(t, err)
	assert.Equal(t, "Memory Platform", config.Name)
	assert.Equal(t, "Memory Vendor", config.BiosVendor)
	assert.Equal(t, "mem", config.FirmwareFileType)
}

// TestConfigLoaderFactory_CreateMemoryLoader_InvalidData tests error handling with invalid data
func TestConfigLoaderFactory_CreateMemoryLoader_InvalidData(t *testing.T) {
	factory := NewConfigLoaderFactory()

	// Invalid JSON data
	invalidConfigData := []byte(`{invalid json`)
	validSchemaData := []byte(`{"type": "object"}`)

	// Create memory loader with invalid data
	loader := factory.CreateMemoryLoader(invalidConfigData, validSchemaData)

	// Test that it fails gracefully
	_, err := loader.LoadPlatformConfig("Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

// TestConfigLoader_MultipleCallsToSameProvider tests that the provider is reused correctly
func TestConfigLoader_MultipleCallsToSameProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := NewMockPlatformConfigProvider()

	// Set up multiple platform configurations
	config1 := FirmwareToolInfo{Name: "Platform 1", BiosVendor: "Vendor 1"}
	config2 := FirmwareToolInfo{Name: "Platform 2", BiosVendor: "Vendor 2"}

	mockProvider.SetPlatformConfig("Platform 1", config1)
	mockProvider.SetPlatformConfig("Platform 2", config2)

	// Create loader
	loader := NewConfigLoaderWithProvider(mockProvider)

	// Make multiple calls
	result1, err1 := loader.LoadPlatformConfig("Platform 1")
	assert.NoError(t, err1)
	assert.Equal(t, config1, result1)

	result2, err2 := loader.LoadPlatformConfig("Platform 2")
	assert.NoError(t, err2)
	assert.Equal(t, config2, result2)

	// Verify all calls were tracked
	assert.Equal(t, 2, mockProvider.GetConfigCallCount())
	allCalls := mockProvider.GetAllConfigCalls()
	assert.Equal(t, []string{"Platform 1", "Platform 2"}, allCalls)
}

// TestConfigLoader_GetProvider tests the provider getter
func TestConfigLoader_GetProvider(t *testing.T) {
	mockProvider := NewMockPlatformConfigProvider()
	loader := NewConfigLoaderWithProvider(mockProvider)

	// Test that we get the same provider back
	retrievedProvider := loader.GetProvider()
	assert.Equal(t, mockProvider, retrievedProvider)
}

// Benchmark tests to ensure performance isn't degraded by the interface approach

func BenchmarkConfigLoader_LoadPlatformConfig(b *testing.B) {
	// Create a mock provider for benchmarking
	mockProvider := NewMockPlatformConfigProvider()
	config := FirmwareToolInfo{
		Name:       "Benchmark Platform",
		BiosVendor: "Benchmark Vendor",
	}
	mockProvider.SetPlatformConfig("Benchmark Platform", config)

	loader := NewConfigLoaderWithProvider(mockProvider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loader.LoadPlatformConfig("Benchmark Platform")
	}
}

func BenchmarkConfigLoaderFactory_CreateTestLoader(b *testing.B) {
	factory := NewConfigLoaderFactory()
	mockProvider := NewMockPlatformConfigProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.CreateTestLoader(mockProvider)
	}
}
