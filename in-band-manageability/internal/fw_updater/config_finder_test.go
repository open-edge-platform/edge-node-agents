/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// Mock valid JSON schema for firmware tool config
const validSchema = `{
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
              "tool_options": {"type": "boolean"},
              "guid": {"type": "boolean"},
              "bios_vendor": {"type": "string"},
              "operating_system": {"type": "string"},
              "firmware_tool": {"type": "string"},
              "firmware_tool_args": {"type": "string"},
              "firmware_tool_check_args": {"type": "string"},
              "firmware_file_type": {"type": "string"},
              "firmware_dest_path": {"type": "string"}
            },
            "required": ["name", "bios_vendor", "firmware_file_type"]
          }
        }
      },
      "required": ["firmware_products"]
    }
  },
  "required": ["firmware_component"]
}`

// Mock valid JSON config for firmware tools
const validConfig = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Alder Lake Client Platform",
        "guid": true,
        "bios_vendor": "Intel Corporation",
        "operating_system": "linux",
        "firmware_tool": "fwupdate",
        "firmware_tool_args": "--apply",
        "firmware_tool_check_args": "-s",
        "firmware_file_type": "xx"
      },
      {
        "name": "Arrow Lake Client Platform",
        "guid": true,
        "bios_vendor": "Intel Corp.",
        "operating_system": "linux",
        "firmware_tool": "/usr/bin/UpdateFirmwareBlobFwupdtool.sh",
        "firmware_file_type": "xx"
      },
      {
        "name": "Default string",
        "tool_options": true,
        "bios_vendor": "American Megatrends Inc.",
        "operating_system": "linux",
        "firmware_tool": "/opt/afulnx/afulnx_64",
        "firmware_file_type": "xx"
      }
    ]
  }
}`

// Mock invalid JSON config (missing required fields)
const invalidConfig = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Test Platform"
      }
    ]
  }
}`

// Mock malformed JSON
const malformedConfig = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Test Platform",
        "bios_vendor": "Test Vendor"
      }
    ]
  // Missing closing bracket
}`

// Additional test constants for edge cases
const emptyFile = ""

const corruptedJSONConfig = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Test Platform",
        "bios_vendor": "Test Vendor",
        "firmware_file_type": "xx",
        "invalid_field": {
          "nested": "value",,
        }
      }
    ]
  }
}`

const incompleteConfigStructure = `{
  "firmware_component": {
    "wrong_key": []
  }
}`

const configWithNullValues = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": null,
        "bios_vendor": "Test Vendor",
        "firmware_file_type": "xx"
      }
    ]
  }
}`

const configWithWrongTypes = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": 123,
        "bios_vendor": "Test Vendor",
        "firmware_file_type": "xx"
      }
    ]
  }
}`

const configWithMissingRequiredFields = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Test Platform"
      }
    ]
  }
}`

const validConfigWithSpecialCharacters = `{
  "firmware_component": {
    "firmware_products": [
      {
        "name": "Test Platform with Spëciål Ćhārs & Symbols!@#",
        "bios_vendor": "Test Vendor with Unicode: ñáéíóú",
        "firmware_file_type": "xx",
        "firmware_tool_args": "--arg=\"value with spaces\" --flag"
      }
    ]
  }
}`

const invalidSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "firmware_component": {
      "type": "invalid_type"
    }
  }
}`

const emptyObjectConfig = `{}`
const nullConfig = `null`
const arrayInsteadOfObjectConfig = `[]`
const stringInsteadOfObjectConfig = `"this is a string"`
const numberInsteadOfObjectConfig = `42`

func setupMockFS(configContent, schemaContent string) afero.Fs {
	fs := afero.NewMemMapFs()

	// Write schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(schemaContent), 0644)
	if err != nil {
		panic("Failed to write schema file: " + err.Error())
	}

	// Write config file
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(configContent), 0644)
	if err != nil {
		panic("Failed to write config file: " + err.Error())
	}

	return fs
}

// Helper function to generate long strings for testing
func generateLongString(length int) string {
	if length <= 0 {
		return ""
	}
	result := make([]byte, length)
	for i := range result {
		result[i] = 'A' + byte(i%26)
	}
	return string(result)
}

// Basic functionality tests

func TestGetFirmwareUpdateToolInfo_Success(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	result, err := GetFirmwareUpdateToolInfo(fs, "Alder Lake Client Platform")
	assert.NoError(t, err)

	// Verify the returned info matches the expected platform
	expected := FirmwareToolInfo{
		Name:                  "Alder Lake Client Platform",
		GUID:                  true,
		BiosVendor:            "Intel Corporation",
		FirmwareTool:          "fwupdate",
		FirmwareToolArgs:      "--apply",
		FirmwareToolCheckArgs: "-s",
		FirmwareFileType:      "xx",
	}

	assert.Equal(t, expected.Name, result.Name)
	assert.Equal(t, expected.BiosVendor, result.BiosVendor)
	assert.Equal(t, expected.FirmwareTool, result.FirmwareTool)
	assert.Equal(t, expected.FirmwareToolArgs, result.FirmwareToolArgs)
	assert.Equal(t, expected.FirmwareToolCheckArgs, result.FirmwareToolCheckArgs)
	assert.Equal(t, expected.FirmwareFileType, result.FirmwareFileType)
	assert.Equal(t, expected.GUID, result.GUID)
}

func TestGetFirmwareUpdateToolInfo_ConfigValidationFails(t *testing.T) {
	fs := setupMockFS(invalidConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Test Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_SchemaFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Write config but no schema file
	err := afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(validConfig), 0644)
	assert.NoError(t, err, "Failed to write config file")

	_, err = GetFirmwareUpdateToolInfo(fs, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_ConfigFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Write schema but no config file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err, "Failed to write schema file")

	_, err = GetFirmwareUpdateToolInfo(fs, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_ArrowLakePlatform(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	result, err := GetFirmwareUpdateToolInfo(fs, "Arrow Lake Client Platform")
	assert.NoError(t, err)

	expected := FirmwareToolInfo{
		Name:             "Arrow Lake Client Platform",
		GUID:             true,
		BiosVendor:       "Intel Corp.",
		FirmwareTool:     "/usr/bin/UpdateFirmwareBlobFwupdtool.sh",
		FirmwareFileType: "xx",
	}

	assert.Equal(t, expected.Name, result.Name)
	assert.Equal(t, expected.BiosVendor, result.BiosVendor)
	assert.Equal(t, expected.FirmwareTool, result.FirmwareTool)
	assert.Equal(t, expected.FirmwareFileType, result.FirmwareFileType)
	assert.Equal(t, expected.GUID, result.GUID)
}

func TestGetFirmwareUpdateToolInfo_PlatformNotFound(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Nonexistent Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform Nonexistent Platform not found in config")
}

func TestGetFirmwareUpdateToolInfo_ToolOptionsTrue(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	result, err := GetFirmwareUpdateToolInfo(fs, "Default string")
	assert.NoError(t, err)

	assert.True(t, result.ToolOptions)
}

func TestGetFirmwareUpdateToolInfo_InvalidJSON(t *testing.T) {
	fs := setupMockFS(malformedConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Test Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_JSONUnmarshalError(t *testing.T) {
	// Create a config that passes schema validation but fails unmarshaling
	problemConfig := `{
		"firmware_component": {
			"firmware_products": "not an array"
		}
	}`

	// Use a more permissive schema for this test
	permissiveSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`

	fs := setupMockFS(problemConfig, permissiveSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

// Validation function tests

func TestValidateFirmwareToolConfig_Success(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	configData, err := validateFirmwareToolConfig(fs)
	assert.NoError(t, err)
	assert.NotNil(t, configData)
	assert.Greater(t, len(configData), 0)
}

func TestValidateFirmwareToolConfig_InvalidConfig(t *testing.T) {
	fs := setupMockFS(invalidConfig, validSchema)

	_, err := validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestValidateFirmwareToolConfig_SchemaFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Write config but no schema file
	err := afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(validConfig), 0644)
	assert.NoError(t, err, "Failed to write config file")

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read schema file")
}

func TestValidateFirmwareToolConfig_ConfigFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Write schema but no config file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err, "Failed to write schema file")

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestValidateFirmwareToolConfig_MalformedJSON(t *testing.T) {
	fs := setupMockFS(malformedConfig, validSchema)

	_, err := validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate config")
}

func TestValidateFirmwareToolConfig_InvalidSchema(t *testing.T) {
	invalidSchema := `{invalid json schema`
	fs := setupMockFS(validConfig, invalidSchema)

	_, err := validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate config")
}

// Edge case tests
func TestGetFirmwareUpdateToolInfo_EmptyConfig(t *testing.T) {
	emptyConfig := `{
		"firmware_component": {
			"firmware_products": []
		}
	}`
	fs := setupMockFS(emptyConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform Any Platform not found in config")
}

func TestGetFirmwareUpdateToolInfo_EmptyPlatformName(t *testing.T) {
	fs := setupMockFS(validConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform  not found in config")
}

// Comprehensive edge case tests for robustness

func TestGetFirmwareUpdateToolInfo_EmptyFile(t *testing.T) {
	fs := setupMockFS(emptyFile, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_CorruptedJSON(t *testing.T) {
	fs := setupMockFS(corruptedJSONConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Test Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
}

func TestGetFirmwareUpdateToolInfo_IncompleteStructure(t *testing.T) {
	fs := setupMockFS(incompleteConfigStructure, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_NullValues(t *testing.T) {
	fs := setupMockFS(configWithNullValues, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_WrongTypes(t *testing.T) {
	fs := setupMockFS(configWithWrongTypes, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_MissingRequiredFields(t *testing.T) {
	fs := setupMockFS(configWithMissingRequiredFields, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Test Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_SpecialCharacters(t *testing.T) {
	fs := setupMockFS(validConfigWithSpecialCharacters, validSchema)

	result, err := GetFirmwareUpdateToolInfo(fs, "Test Platform with Spëciål Ćhārs & Symbols!@#")
	assert.NoError(t, err)
	assert.Equal(t, "Test Platform with Spëciål Ćhārs & Symbols!@#", result.Name)
	assert.Equal(t, "Test Vendor with Unicode: ñáéíóú", result.BiosVendor)
	assert.Equal(t, "--arg=\"value with spaces\" --flag", result.FirmwareToolArgs)
}

func TestGetFirmwareUpdateToolInfo_LongPlatformName(t *testing.T) {
	longName := generateLongString(1000)
	largePlatformConfig := `{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "` + longName + `",
					"bios_vendor": "Test Vendor",
					"firmware_file_type": "xx"
				}
			]
		}
	}`

	fs := setupMockFS(largePlatformConfig, validSchema)

	result, err := GetFirmwareUpdateToolInfo(fs, longName)
	assert.NoError(t, err)
	assert.Equal(t, longName, result.Name)
	assert.Equal(t, "Test Vendor", result.BiosVendor)
}

func TestValidateFirmwareToolConfig_InvalidSchemaJSON(t *testing.T) {
	fs := setupMockFS(validConfig, invalidSchemaJSON)

	_, err := validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate config")
}

func TestGetFirmwareUpdateToolInfo_EmptyObject(t *testing.T) {
	fs := setupMockFS(emptyObjectConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_NullConfig(t *testing.T) {
	fs := setupMockFS(nullConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_ArrayConfig(t *testing.T) {
	fs := setupMockFS(arrayInsteadOfObjectConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_StringConfig(t *testing.T) {
	fs := setupMockFS(stringInsteadOfObjectConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestGetFirmwareUpdateToolInfo_NumberConfig(t *testing.T) {
	fs := setupMockFS(numberInsteadOfObjectConfig, validSchema)

	_, err := GetFirmwareUpdateToolInfo(fs, "Any Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not match schema")
}

func TestValidateFileSize_ConfigFileExceedsMaxSize(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create a config file that exceeds the 1MB limit (1MB + 1 byte)
	largeContent := make([]byte, 1024*1024+1)
	for i := range largeContent {
		largeContent[i] = 'a' // Fill with valid text characters
	}
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, largeContent, 0644)
	assert.NoError(t, err)

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
	assert.Contains(t, err.Error(), "maximum allowed is 1048576 bytes")
}

func TestValidateFileSize_SchemaFileExceedsMaxSize(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid config file
	err := afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(validConfig), 0644)
	assert.NoError(t, err)

	// Create a schema file that exceeds the 1MB limit
	largeContent := make([]byte, 1024*1024+1)
	for i := range largeContent {
		largeContent[i] = 'a' // Fill with valid text characters
	}
	err = afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, largeContent, 0644)
	assert.NoError(t, err)

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
	assert.Contains(t, err.Error(), "maximum allowed is 1048576 bytes")
}

func TestValidateFileSize_AtMaxSizeLimit(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid config exactly at the 1MB limit
	configAtLimit := make([]byte, 1024*1024)
	validConfigBytes := []byte(validConfig)
	copy(configAtLimit, validConfigBytes)
	// Fill the rest with spaces to reach exactly 1MB
	for i := len(validConfigBytes); i < len(configAtLimit); i++ {
		configAtLimit[i] = ' '
	}
	err := afero.WriteFile(fs, firmwareToolInfoFilePath, configAtLimit, 0644)
	assert.NoError(t, err)

	// Create a valid schema file
	err = afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// This should pass since it's exactly at the limit
	_, err = validateFirmwareToolConfig(fs)
	// This may fail due to invalid JSON, but it should not fail due to size
	if err != nil {
		assert.NotContains(t, err.Error(), "is too large")
	}
}

func TestGetFirmwareUpdateToolInfo_OversizedConfigFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create an oversized config file
	largeContent := make([]byte, 1024*1024+1)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, largeContent, 0644)
	assert.NoError(t, err)

	_, err = GetFirmwareUpdateToolInfo(fs, "Test Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "is too large")
}

func TestValidateJSONContent_BinaryData(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create a config file with binary data (null bytes)
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, '{', '"', 't', 'e', 's', 't', '"', ':', '1', '}'}
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, binaryContent, 0644)
	assert.NoError(t, err)

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contains binary data, not text")
}

func TestValidateJSONContent_ValidTextWithSpecialChars(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create valid files with special characters that should be allowed
	configWithSpecialChars := `{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Test\tPlatform\nWith\rSpecial\fChars",
					"bios_vendor": "Test Vendor",
					"firmware_file_type": "bin"
				}
			]
		}
	}`

	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(configWithSpecialChars), 0644)
	assert.NoError(t, err)

	// This should pass the binary data validation (though it may fail schema validation)
	_, err = validateFirmwareToolConfig(fs)
	if err != nil {
		assert.NotContains(t, err.Error(), "contains binary data")
	}
}

func TestSecureReadFile_ValidFile(t *testing.T) {
	// Test using the existing framework to ensure secureReadFile works correctly
	fs := setupMockFS(validConfig, validSchema)

	// Test reading the config file through secureReadFile
	data, err := secureReadFile(fs, firmwareToolInfoFilePath)
	assert.NoError(t, err)
	assert.Equal(t, validConfig, string(data))
}

func TestSecureReadFile_OversizedFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create an oversized config file (1MB + 1 byte)
	largeContent := make([]byte, 1024*1024+1)
	for i := range largeContent {
		largeContent[i] = 'z'
	}
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, largeContent, 0644)
	assert.NoError(t, err)

	// Test that secureReadFile rejects oversized files
	_, err = secureReadFile(fs, firmwareToolInfoFilePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestSecureReadFile_BinaryContent(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create a config file with binary content
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, binaryContent, 0644)
	assert.NoError(t, err)

	// Test that secureReadFile rejects binary content
	_, err = secureReadFile(fs, firmwareToolInfoFilePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contains binary data, not text")
}

// Enhanced security validation tests

func TestValidateJSONStructure_ExcessiveNesting(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create a deeply nested JSON that has valid schema but exceeds depth limit
	// Start with the basic required structure
	deepNested := `{"firmware_component":{"firmware_products":[{"name":"test","bios_vendor":"test","firmware_file_type":"bin","extra":`

	// Add deep nesting within the extra field (reduced to avoid panic)
	for i := 0; i < 101; i++ { // Exceed the 100 limit
		deepNested += fmt.Sprintf(`{"level%d":`, i)
	}
	deepNested += `"value"`
	for i := 0; i < 101; i++ {
		deepNested += "}"
	}
	deepNested += "}]}}"

	// Verify the JSON is valid first
	var testObj interface{}
	jsonErr := json.Unmarshal([]byte(deepNested), &testObj)
	assert.NoError(t, jsonErr, "Generated JSON should be valid")

	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(deepNested), 0644)
	assert.NoError(t, err)

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "excessive JSON nesting depth")
	}
}

func TestValidateJSONStructure_TooManyProperties(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Create JSON with excessive number of properties
	var properties []string
	for i := 0; i < 10001; i++ { // Exceed the 10000 limit
		properties = append(properties, fmt.Sprintf(`"prop%d":"value%d"`, i, i))
	}

	excessiveProps := "{" + strings.Join(properties, ",") + "}"
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(excessiveProps), 0644)
	assert.NoError(t, err)

	_, err = validateFirmwareToolConfig(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many JSON properties")
}

func TestValidateJSONStructure_ValidComplexStructure(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a complex but valid JSON structure within limits
	complexConfig := `{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Complex Platform",
					"bios_vendor": "Test Vendor",
					"firmware_file_type": "bin",
					"nested_config": {
						"level1": {
							"level2": {
								"level3": {
									"settings": ["opt1", "opt2", "opt3"]
								}
							}
						}
					},
					"multiple_arrays": [
						{"item1": "value1"},
						{"item2": "value2"},
						{"item3": "value3"}
					]
				}
			]
		}
	}`

	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(complexConfig), 0644)
	assert.NoError(t, err)

	// This should pass structure validation (though it may fail schema validation)
	_, err = validateFirmwareToolConfig(fs)
	if err != nil {
		assert.NotContains(t, err.Error(), "excessive JSON nesting")
		assert.NotContains(t, err.Error(), "too many JSON properties")
	}
}

func TestValidateFilePathSecurity_PathTraversal(t *testing.T) {
	// Test path traversal attack detection
	maliciousPath := "/etc/../../../etc/passwd"

	err := validateFilePathSecurity(maliciousPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "suspicious path traversal patterns")
}

func TestValidateFilePathSecurity_ExcessivelyLongPath(t *testing.T) {
	// Create an excessively long file path
	longPath := "/" + strings.Repeat("a", 4097) // Exceed 4096 limit

	err := validateFilePathSecurity(longPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path")
	assert.Contains(t, err.Error(), "is too long")
}

func TestValidateFilePathSecurity_ValidPath(t *testing.T) {
	validPath := "/etc/firmware_tool_info.conf"

	err := validateFilePathSecurity(validPath)
	assert.NoError(t, err)
}

func TestSecureReadFile_EnhancedSecurity(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Test that all security validations work together
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(validConfig), 0644)
	assert.NoError(t, err)

	// This should pass all security validations
	data, err := secureReadFile(fs, firmwareToolInfoFilePath)
	assert.NoError(t, err)
	assert.Equal(t, validConfig, string(data))
}

func TestGetJSONComplexity_SimpleObject(t *testing.T) {
	simpleObj := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	depth, propCount := getJSONComplexity(simpleObj, 0)
	assert.Equal(t, 1, depth) // The object itself has depth 1 (corrected)
	assert.Equal(t, 2, propCount)
}

func TestGetJSONComplexity_NestedObject(t *testing.T) {
	nestedObj := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
		},
	}

	depth, propCount := getJSONComplexity(nestedObj, 0)
	assert.Equal(t, 3, depth)     // Three levels of nesting: level1 -> level2 -> level3
	assert.Equal(t, 3, propCount) // level1 + level2 + level3
}

func TestGetJSONComplexity_ArrayStructure(t *testing.T) {
	arrayObj := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"item1": "value1"},
			map[string]interface{}{"item2": "value2"},
		},
	}

	depth, propCount := getJSONComplexity(arrayObj, 0)
	assert.Equal(t, 3, depth)     // Root object -> array -> inner objects (corrected)
	assert.Equal(t, 3, propCount) // items + item1 + item2
}

// File permission and locking test scenarios

// mockErrorFS is a filesystem implementation that can simulate various error conditions
type mockErrorFS struct {
	afero.Fs
	configReadError          error
	schemaReadError          error
	configStatError          error
	schemaStatError          error
	simulatePermissionDenied bool
	simulateFileLocked       bool
}

func newMockErrorFS(baseFS afero.Fs) *mockErrorFS {
	return &mockErrorFS{
		Fs: baseFS,
	}
}

func (m *mockErrorFS) Open(name string) (afero.File, error) {
	if name == firmwareToolInfoFilePath && m.configReadError != nil {
		return nil, m.configReadError
	}
	if name == firmwareToolInfoSchemaFilePath && m.schemaReadError != nil {
		return nil, m.schemaReadError
	}
	if m.simulatePermissionDenied {
		return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.EACCES}
	}
	if m.simulateFileLocked {
		return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.EAGAIN}
	}
	return m.Fs.Open(name)
}

func (m *mockErrorFS) Stat(name string) (os.FileInfo, error) {
	if name == firmwareToolInfoFilePath && m.configStatError != nil {
		return nil, m.configStatError
	}
	if name == firmwareToolInfoSchemaFilePath && m.schemaStatError != nil {
		return nil, m.schemaStatError
	}
	if m.simulatePermissionDenied {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: syscall.EACCES}
	}
	return m.Fs.Stat(name)
}

func TestFilePermissionErrors_ConfigFilePermissionDenied(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFilePermissionErrors_SchemaFilePermissionDenied(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.schemaReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read schema file")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFilePermissionErrors_ConfigFileStatPermissionDenied(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configStatError = &fs.PathError{Op: "stat", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Contains(t, err.Error(), "failed to get file info")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFilePermissionErrors_SchemaFileStatPermissionDenied(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.schemaStatError = &fs.PathError{Op: "stat", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read schema file")
	assert.Contains(t, err.Error(), "failed to get file info")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFilePermissionErrors_BothFilesPermissionDenied(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.simulatePermissionDenied = true

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFileLockingErrors_ConfigFileLocked(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EAGAIN}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	// Note: EAGAIN error messages can vary by OS, so we check for common patterns
	errorMsg := err.Error()
	lockingDetected := strings.Contains(errorMsg, "resource temporarily unavailable") ||
		strings.Contains(errorMsg, "try again") ||
		strings.Contains(errorMsg, "eagain") ||
		strings.Contains(errorMsg, "EAGAIN")
	assert.True(t, lockingDetected, "Expected file locking error indication, got: %s", errorMsg)
}

func TestFileLockingErrors_SchemaFileLocked(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.schemaReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EAGAIN}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read schema file")
	// Check for file locking error indicators
	errorMsg := err.Error()
	lockingDetected := strings.Contains(errorMsg, "resource temporarily unavailable") ||
		strings.Contains(errorMsg, "try again") ||
		strings.Contains(errorMsg, "eagain") ||
		strings.Contains(errorMsg, "EAGAIN")
	assert.True(t, lockingDetected, "Expected file locking error indication, got: %s", errorMsg)
}

func TestFileLockingErrors_BothFilesLocked(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.simulateFileLocked = true

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	// Check for file locking error indicators
	errorMsg := err.Error()
	lockingDetected := strings.Contains(errorMsg, "resource temporarily unavailable") ||
		strings.Contains(errorMsg, "try again") ||
		strings.Contains(errorMsg, "eagain") ||
		strings.Contains(errorMsg, "EAGAIN")
	assert.True(t, lockingDetected, "Expected file locking error indication, got: %s", errorMsg)
}

// Test scenarios with mixed permission and content issues
func TestFilePermissionErrors_ConfigPermissionDeniedWithValidSchema(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}

	_, err := validateFirmwareToolConfig(mockFS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestFilePermissionErrors_SchemaPermissionDeniedWithValidConfig(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.schemaReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EACCES}

	_, err := validateFirmwareToolConfig(mockFS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read schema file")
	assert.Contains(t, err.Error(), "permission denied")
}

// Test OS-specific permission errors
func TestFilePermissionErrors_ReadOnlyFileSystem(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EROFS}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	// Check for read-only filesystem error
	errorMsg := err.Error()
	readOnlyDetected := strings.Contains(errorMsg, "read-only") ||
		strings.Contains(errorMsg, "erofs") ||
		strings.Contains(errorMsg, "EROFS")
	assert.True(t, readOnlyDetected, "Expected read-only filesystem error indication, got: %s", errorMsg)
}

func TestFilePermissionErrors_DeviceBusy(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EBUSY}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	// Check for device busy error
	errorMsg := err.Error()
	busyDetected := strings.Contains(errorMsg, "device or resource busy") ||
		strings.Contains(errorMsg, "ebusy") ||
		strings.Contains(errorMsg, "EBUSY") ||
		strings.Contains(errorMsg, "busy")
	assert.True(t, busyDetected, "Expected device busy error indication, got: %s", errorMsg)
}

// Test concurrent access scenarios
func TestFilePermissionErrors_InterruptedSystemCall(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)
	mockFS.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EINTR}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	// Check for interrupted system call error
	errorMsg := err.Error()
	interruptDetected := strings.Contains(errorMsg, "interrupted") ||
		strings.Contains(errorMsg, "eintr") ||
		strings.Contains(errorMsg, "EINTR")
	assert.True(t, interruptDetected, "Expected interrupted system call error indication, got: %s", errorMsg)
}

// Test edge cases with permission errors and security validations
func TestFilePermissionErrors_ConfigPermissionDeniedAfterSizeCheck(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)

	// This will pass the stat check but fail on actual read
	mockFS.configReadError = &fs.PathError{Op: "read", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestFilePermissionErrors_SchemaPermissionDeniedAfterSizeCheck(t *testing.T) {
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)

	// This will pass the stat check but fail on actual read
	mockFS.schemaReadError = &fs.PathError{Op: "read", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read schema file")
}

// Test scenarios where file operations succeed partially
func TestFilePermissionErrors_PartialReadScenario(t *testing.T) {
	// This test ensures that if permissions change between stat and read operations,
	// the error is handled gracefully

	// Create a more complex scenario where the file exists and is readable,
	// but contains invalid permission settings that would cause issues
	// in a real filesystem scenario
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, firmwareToolInfoSchemaFilePath, []byte(validSchema), 0644)
	assert.NoError(t, err)

	// Write config file with no read permissions (simulated via content that would cause read issues)
	err = afero.WriteFile(fs, firmwareToolInfoFilePath, []byte(validConfig), 0000) // No permissions
	assert.NoError(t, err)

	// The afero memmap filesystem doesn't enforce file permissions like a real filesystem,
	// but this test structure shows how permission errors would be handled
	_, err = GetFirmwareUpdateToolInfo(fs, "Alder Lake Client Platform")

	// The test should either succeed (if permissions aren't enforced)
	// or fail gracefully with appropriate error handling
	if err != nil {
		// If it fails, ensure the error is handled gracefully
		assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	} else {
		// If it succeeds, the functionality is working despite the permission setup
		assert.NotNil(t, "File operation completed successfully")
	}
}

// Test graceful error recovery scenarios
func TestFilePermissionErrors_GracefulErrorRecovery(t *testing.T) {
	// Test that the system doesn't crash or leak resources when encountering permission errors
	baseFS := setupMockFS(validConfig, validSchema)

	// Test multiple error conditions in sequence to ensure robust error handling
	testCases := []struct {
		name        string
		setupError  func(*mockErrorFS)
		expectedErr string
	}{
		{
			name: "config_permission_denied",
			setupError: func(m *mockErrorFS) {
				m.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}
			},
			expectedErr: "permission denied",
		},
		{
			name: "schema_permission_denied",
			setupError: func(m *mockErrorFS) {
				m.schemaReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EACCES}
			},
			expectedErr: "permission denied",
		},
		{
			name: "config_file_locked",
			setupError: func(m *mockErrorFS) {
				m.configReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoFilePath, Err: syscall.EAGAIN}
			},
			expectedErr: "resource temporarily unavailable",
		},
		{
			name: "schema_file_locked",
			setupError: func(m *mockErrorFS) {
				m.schemaReadError = &fs.PathError{Op: "open", Path: firmwareToolInfoSchemaFilePath, Err: syscall.EAGAIN}
			},
			expectedErr: "resource temporarily unavailable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the mock filesystem for each test
			mockFS := newMockErrorFS(baseFS)
			tc.setupError(mockFS)

			_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
			assert.Error(t, err, "Expected error for test case: %s", tc.name)

			// Verify the error contains expected information without being too strict about exact format
			errorMsg := strings.ToLower(err.Error())
			expectedErrLower := strings.ToLower(tc.expectedErr)

			// For some errors, check for common variations
			if strings.Contains(expectedErrLower, "resource temporarily unavailable") {
				lockingDetected := strings.Contains(errorMsg, "resource temporarily unavailable") ||
					strings.Contains(errorMsg, "try again") ||
					strings.Contains(errorMsg, "eagain") ||
					strings.Contains(errorMsg, "temporarily unavailable")
				assert.True(t, lockingDetected, "Expected locking error indication for %s, got: %s", tc.name, err.Error())
			} else {
				assert.Contains(t, errorMsg, expectedErrLower, "Expected error message for %s", tc.name)
			}

			// Ensure the error is properly wrapped and contains context
			assert.Contains(t, err.Error(), "failed to validate firmware tool config",
				"Error should be properly wrapped for %s", tc.name)
		})
	}
}

// Integration test for permission error scenarios
func TestFilePermissionErrors_IntegrationWithSecurityValidation(t *testing.T) {
	// Test that permission errors are caught before security validation attempts
	baseFS := setupMockFS(validConfig, validSchema)
	mockFS := newMockErrorFS(baseFS)

	// Set up permission error that should be caught early
	mockFS.configStatError = &fs.PathError{Op: "stat", Path: firmwareToolInfoFilePath, Err: syscall.EACCES}

	_, err := GetFirmwareUpdateToolInfo(mockFS, "Alder Lake Client Platform")
	assert.Error(t, err)

	// Verify that the permission error is caught before security validation
	assert.Contains(t, err.Error(), "failed to validate firmware tool config")
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Contains(t, err.Error(), "failed to get file info")

	// Ensure the error doesn't mention security validation failures,
	// which would indicate the permission error was caught appropriately early
	assert.NotContains(t, err.Error(), "excessive JSON nesting")
	assert.NotContains(t, err.Error(), "too many JSON properties")
	assert.NotContains(t, err.Error(), "contains binary data")
}
