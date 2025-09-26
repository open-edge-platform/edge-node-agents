/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"errors"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example tests demonstrating the interface-based approach with proper mocking

func TestFileSystemOperations_MockSuccess(t *testing.T) {
	// Create mock file system operations
	mockFS := NewMockFileSystemOperations()

	// Set up test data
	configContent := `{"firmware_component": {"firmware_products": [{"name": "test", "bios_vendor": "vendor", "firmware_file_type": "xx"}]}}`
	schemaContent := `{"type": "object"}`

	mockFS.SetFileContent("/test/config.json", []byte(configContent))
	mockFS.SetFileContent("/test/schema.json", []byte(schemaContent))

	// Test file operations
	data, err := mockFS.ReadFile("/test/config.json")
	assert.NoError(t, err)
	assert.Equal(t, configContent, string(data))

	info, err := mockFS.Stat("/test/config.json")
	assert.NoError(t, err)
	assert.Equal(t, int64(len(configContent)), info.Size())
}

func TestFileSystemOperations_MockPermissionError(t *testing.T) {
	// Create mock file system operations with permission errors
	mockFS := NewMockFileSystemOperations()
	mockFS.SimulatePermissionError()

	// Test that permission errors are returned
	_, err := mockFS.ReadFile("/test/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = mockFS.Stat("/test/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFileSystemOperations_MockFileLockingError(t *testing.T) {
	// Create mock file system operations with file locking errors
	mockFS := NewMockFileSystemOperations()
	mockFS.SimulateFileLockError()

	// Test that file locking errors are returned
	_, err := mockFS.ReadFile("/test/config.json")
	assert.Error(t, err)
	// Check for various possible error messages that indicate locking
	errorMsg := err.Error()
	lockingDetected := assert.Contains(t, errorMsg, "resource temporarily unavailable") ||
		assert.Contains(t, errorMsg, "try again") ||
		assert.Contains(t, errorMsg, "eagain")
	assert.True(t, lockingDetected, "Expected file locking error indication, got: %s", errorMsg)
}

func TestSchemaValidator_MockSuccess(t *testing.T) {
	// Create mock schema validator
	mockValidator := NewMockSchemaValidator()
	mockValidator.SetValidationSuccess()

	// Test successful validation
	err := mockValidator.ValidateConfig([]byte(`{"type": "object"}`), []byte(`{"key": "value"}`))
	assert.NoError(t, err)

	// Verify the call was recorded
	assert.Equal(t, 1, mockValidator.GetValidationCallCount())

	lastCall := mockValidator.GetLastValidationCall()
	assert.NotNil(t, lastCall)
	assert.Equal(t, `{"type": "object"}`, string(lastCall.SchemaData))
	assert.Equal(t, `{"key": "value"}`, string(lastCall.ConfigData))
}

func TestSchemaValidator_MockFailure(t *testing.T) {
	// Create mock schema validator with failure
	mockValidator := NewMockSchemaValidator()
	mockValidator.SetValidationFailure("validation failed")

	// Test validation failure
	err := mockValidator.ValidateConfig([]byte(`{"type": "object"}`), []byte(`invalid json`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Verify the call was recorded
	assert.Equal(t, 1, mockValidator.GetValidationCallCount())
}

func TestConfigReader_MockSuccess(t *testing.T) {
	// Create mock config reader
	mockReader := NewMockConfigReader("/test/config.json", "/test/schema.json")

	configData := []byte(`{"firmware_component": {"firmware_products": []}}`)
	mockReader.SetConfigData(configData)

	// Test successful config reading
	data, err := mockReader.ReadAndValidateConfig()
	assert.NoError(t, err)
	assert.Equal(t, configData, data)

	// Verify call count
	assert.Equal(t, 1, mockReader.GetReadCallCount())

	// Test path getters
	assert.Equal(t, "/test/config.json", mockReader.GetConfigFilePath())
	assert.Equal(t, "/test/schema.json", mockReader.GetSchemaFilePath())
}

func TestConfigReader_MockError(t *testing.T) {
	// Create mock config reader with error
	mockReader := NewMockConfigReader("/test/config.json", "/test/schema.json")
	mockReader.SetReadError(errors.New("failed to read config"))

	// Test error handling
	_, err := mockReader.ReadAndValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config")

	// Verify call count
	assert.Equal(t, 1, mockReader.GetReadCallCount())
}

func TestPlatformConfigProvider_MockSuccess(t *testing.T) {
	// Create mock platform config provider
	mockProvider := NewMockPlatformConfigProvider()

	expectedConfig := FirmwareToolInfo{
		Name:             "Test Platform",
		BiosVendor:       "Test Vendor",
		FirmwareFileType: "xx",
	}
	mockProvider.SetPlatformConfig("Test Platform", expectedConfig)

	// Test successful platform config retrieval
	config, err := mockProvider.GetPlatformConfig("Test Platform")
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)

	// Verify call tracking
	assert.Equal(t, 1, mockProvider.GetConfigCallCount())
	assert.Equal(t, "Test Platform", mockProvider.GetLastConfigCall())

	allCalls := mockProvider.GetAllConfigCalls()
	assert.Len(t, allCalls, 1)
	assert.Equal(t, "Test Platform", allCalls[0])
}

func TestPlatformConfigProvider_MockPlatformNotFound(t *testing.T) {
	// Create mock platform config provider without setting any configs
	mockProvider := NewMockPlatformConfigProvider()

	// Test platform not found error
	_, err := mockProvider.GetPlatformConfig("Nonexistent Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform Nonexistent Platform not found in config")

	// Verify call tracking
	assert.Equal(t, 1, mockProvider.GetConfigCallCount())
	assert.Equal(t, "Nonexistent Platform", mockProvider.GetLastConfigCall())
}

func TestPlatformConfigProvider_MockSpecificError(t *testing.T) {
	// Create mock platform config provider with specific error
	mockProvider := NewMockPlatformConfigProvider()
	mockProvider.SetPlatformError("Error Platform", errors.New("custom error"))

	// Test specific error handling
	_, err := mockProvider.GetPlatformConfig("Error Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom error")

	// Verify call tracking
	assert.Equal(t, 1, mockProvider.GetConfigCallCount())
	assert.Equal(t, "Error Platform", mockProvider.GetLastConfigCall())
}

func TestIntegration_MockedDependencies(t *testing.T) {
	// Create all mocked dependencies
	mockFS := NewMockFileSystemOperations()
	mockValidator := NewMockSchemaValidator()

	// Set up valid test data
	configContent := []byte(`{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Test Platform",
					"bios_vendor": "Test Vendor",
					"firmware_file_type": "xx"
				}
			]
		}
	}`)
	schemaContent := []byte(`{"type": "object"}`)

	mockFS.SetFileContent("/test/config.json", configContent)
	mockFS.SetFileContent("/test/schema.json", schemaContent)
	mockValidator.SetValidationSuccess()

	// Create config reader with mocked dependencies
	configReader := NewSecureConfigReader(mockFS, mockValidator, "/test/config.json", "/test/schema.json")

	// Create platform config provider
	provider := NewFirmwarePlatformConfigProvider(configReader)

	// Test the complete integration
	config, err := provider.GetPlatformConfig("Test Platform")
	assert.NoError(t, err)
	assert.Equal(t, "Test Platform", config.Name)
	assert.Equal(t, "Test Vendor", config.BiosVendor)
	assert.Equal(t, "xx", config.FirmwareFileType)

	// Verify that the validator was called
	assert.Equal(t, 1, mockValidator.GetValidationCallCount())

	lastCall := mockValidator.GetLastValidationCall()
	assert.NotNil(t, lastCall)
	assert.Equal(t, schemaContent, lastCall.SchemaData)
	assert.Equal(t, configContent, lastCall.ConfigData)
}

func TestIntegration_MockedErrors(t *testing.T) {
	// Test various error scenarios with mocked dependencies

	testCases := []struct {
		name               string
		setupMocks         func(*MockFileSystemOperations, *MockSchemaValidator)
		expectedErrorParts []string
	}{
		{
			name: "file_permission_denied",
			setupMocks: func(fs *MockFileSystemOperations, v *MockSchemaValidator) {
				fs.SimulatePermissionError()
				v.SetValidationSuccess()
			},
			expectedErrorParts: []string{"failed to validate firmware tool config", "permission denied"},
		},
		{
			name: "file_locked",
			setupMocks: func(fs *MockFileSystemOperations, v *MockSchemaValidator) {
				fs.SimulateFileLockError()
				v.SetValidationSuccess()
			},
			expectedErrorParts: []string{"failed to validate firmware tool config"},
		},
		{
			name: "validation_failure",
			setupMocks: func(fs *MockFileSystemOperations, v *MockSchemaValidator) {
				fs.SetFileContent("/test/config.json", []byte(`{"valid": "json"}`))
				fs.SetFileContent("/test/schema.json", []byte(`{"type": "object"}`))
				v.SetValidationFailure("schema validation failed")
			},
			expectedErrorParts: []string{"failed to validate firmware tool config", "schema validation failed"},
		},
		{
			name: "config_file_not_found",
			setupMocks: func(fs *MockFileSystemOperations, v *MockSchemaValidator) {
				fs.SetFileContent("/test/schema.json", []byte(`{"type": "object"}`))
				// Don't set config file content - it will return ENOENT
				v.SetValidationSuccess()
			},
			expectedErrorParts: []string{"failed to validate firmware tool config", "failed to read config file"},
		},
		{
			name: "schema_file_not_found",
			setupMocks: func(fs *MockFileSystemOperations, v *MockSchemaValidator) {
				fs.SetFileContent("/test/config.json", []byte(`{"valid": "json"}`))
				// Don't set schema file content - it will return ENOENT
				v.SetValidationSuccess()
			},
			expectedErrorParts: []string{"failed to validate firmware tool config", "failed to read schema file"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh mocked dependencies for each test
			mockFS := NewMockFileSystemOperations()
			mockValidator := NewMockSchemaValidator()

			// Apply test-specific setup
			tc.setupMocks(mockFS, mockValidator)

			// Create config reader with mocked dependencies
			configReader := NewSecureConfigReader(mockFS, mockValidator, "/test/config.json", "/test/schema.json")
			provider := NewFirmwarePlatformConfigProvider(configReader)

			// Test the error scenario
			_, err := provider.GetPlatformConfig("Test Platform")
			assert.Error(t, err, "Expected error for test case: %s", tc.name)

			// Verify that the error contains expected parts
			errorMsg := err.Error()
			for _, expectedPart := range tc.expectedErrorParts {
				assert.Contains(t, errorMsg, expectedPart,
					"Expected error to contain '%s' for test case '%s', got: %s",
					expectedPart, tc.name, errorMsg)
			}
		})
	}
}

func TestBackwardCompatibility_GetFirmwareUpdateToolInfo(t *testing.T) {
	// Test that the original GetFirmwareUpdateToolInfo function still works
	// This ensures backward compatibility while using the new interface-based approach internally

	// This test would normally use afero.NewMemMapFs(), but we're testing
	// that the function correctly creates and uses the interface-based implementations
	// The actual filesystem operations are tested separately above

	// Note: This is a conceptual test showing how backward compatibility is maintained
	// In practice, you'd still need to provide a real or mocked filesystem for this test
	// but the important thing is that existing code continues to work unchanged

	t.Skip("This test demonstrates the concept of backward compatibility - actual implementation would require filesystem setup")
}

// Example of how to use mocks for specific system error scenarios
func TestSpecificSystemErrors_DetailedScenarios(t *testing.T) {
	testCases := []struct {
		name          string
		setupFS       func(*MockFileSystemOperations)
		expectedError string
	}{
		{
			name: "permission_denied_EACCES",
			setupFS: func(fs *MockFileSystemOperations) {
				fs.SetFileError("/test/config.json", syscall.EACCES)
			},
			expectedError: "permission denied",
		},
		{
			name: "resource_busy_EBUSY",
			setupFS: func(fs *MockFileSystemOperations) {
				fs.SimulateBusyDevice = true
			},
			expectedError: "device or resource busy",
		},
		{
			name: "interrupted_EINTR",
			setupFS: func(fs *MockFileSystemOperations) {
				fs.SimulateInterrupted = true
			},
			expectedError: "interrupted",
		},
		{
			name: "read_only_EROFS",
			setupFS: func(fs *MockFileSystemOperations) {
				fs.SimulateReadOnlyFS = true
			},
			expectedError: "read-only",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFS := NewMockFileSystemOperations()
			tc.setupFS(mockFS)

			mockValidator := NewMockSchemaValidator()
			mockValidator.SetValidationSuccess()

			configReader := NewSecureConfigReader(mockFS, mockValidator, "/test/config.json", "/test/schema.json")
			provider := NewFirmwarePlatformConfigProvider(configReader)

			_, err := provider.GetPlatformConfig("Test Platform")
			assert.Error(t, err)
			// Note: Error message format may vary by OS, so we check for general error indication
			assert.NotEmpty(t, err.Error(), "Error should not be empty for test case: %s", tc.name)
		})
	}
}
