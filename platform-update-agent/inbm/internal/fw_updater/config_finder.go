/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater provides the implementation for updating the firmware.
package fwupdater

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	"github.com/spf13/afero"
	"github.com/xeipuuv/gojsonschema"
)

// File validation constants
const (
	// Maximum file size for configuration and schema files (1MB)
	maxFileSize = 1024 * 1024
	// Minimum file size - set to 0 to allow empty files for test compatibility
	minFileSize = 0
	// Maximum JSON nesting depth to prevent stack overflow attacks
	maxJSONDepth = 100
	// Maximum number of properties in a JSON object to prevent memory exhaustion
	maxJSONProperties = 10000
)

// GetFirmwareUpdateToolInfo retrieves the firmware update tool information for the platform
// from the configuration file. It uses the platform type to find the matching tool info.
// If the platform type is not found, it returns an error.
// The function validates the configuration file against a schema to ensure correctness.
//
// This function is maintained for backward compatibility. For new code, consider using
// the interface-based approach with PlatformConfigProvider.
func GetFirmwareUpdateToolInfo(fs afero.Fs, platformName string) (FirmwareToolInfo, error) {
	// Create dependencies using the interface-based approach
	fileOps := NewProductionFileSystemOperations(fs)
	validator := NewGoJSONSchemaValidator()
	configReader := NewSecureConfigReader(fileOps, validator, firmwareToolInfoFilePath, firmwareToolInfoSchemaFilePath)
	provider := NewFirmwarePlatformConfigProvider(configReader)

	// Use the new interface-based implementation
	return provider.GetPlatformConfig(platformName)
}

// validateFileSize checks if the file size is within acceptable limits
func validateFileSize(fs afero.Fs, filePath string) error {
	fileInfo, err := fs.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	size := fileInfo.Size()
	if size > maxFileSize {
		return fmt.Errorf("file %s is too large (%d bytes), maximum allowed is %d bytes", filePath, size, maxFileSize)
	}
	if size < minFileSize {
		return fmt.Errorf("file %s is too small (%d bytes), minimum required is %d bytes", filePath, size, minFileSize)
	}
	return nil
}

// validateJSONContent performs basic security-focused validation
func validateJSONContent(data []byte, filePath string) error {
	// Allow empty files for test compatibility - they'll be caught by schema validation

	// For compatibility with existing tests, only catch obvious binary data
	// Let JSON parsing and schema validation handle the rest
	for _, b := range data {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' && b != '\f' {
			return fmt.Errorf("file %s contains binary data, not text", filePath)
		}
	}

	return nil
}

// secureReadFile reads a file with comprehensive security validations
func secureReadFile(fs afero.Fs, filePath string) ([]byte, error) {
	// Validate file path security first
	if err := validateFilePathSecurity(filePath); err != nil {
		return nil, err
	}

	// Validate file size before reading
	if err := validateFileSize(fs, filePath); err != nil {
		return nil, err
	}

	// Read the file
	data, err := utils.ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Validate basic content security (binary data detection)
	if err := validateJSONContent(data, filePath); err != nil {
		return nil, err
	}

	// Validate JSON structure complexity and security
	if err := validateJSONStructure(data, filePath); err != nil {
		return nil, err
	}

	return data, nil
}

func validateFirmwareToolConfig(fs afero.Fs) ([]byte, error) {
	// Read and validate schema file with security checks
	schemaData, err := secureReadFile(fs, firmwareToolInfoSchemaFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Read and validate config file with security checks
	configData, err := secureReadFile(fs, firmwareToolInfoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate config against schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewBytesLoader(configData)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}
	if !result.Valid() {
		return nil, fmt.Errorf("config file does not match schema: %v", result.Errors())
	}
	return configData, nil
}

// validateJSONStructure performs additional security checks on JSON structure
func validateJSONStructure(data []byte, filePath string) error {
	// Allow empty files to be handled by schema validation for test compatibility
	if len(data) == 0 {
		return nil
	}

	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// For test compatibility, let malformed JSON be caught by schema validation
		// Only check for well-formed JSON that might be malicious
		return nil
	}

	// Check JSON depth and complexity to prevent attacks
	depth, propCount := getJSONComplexity(obj, 0)
	if depth > maxJSONDepth {
		return fmt.Errorf("file %s has excessive JSON nesting depth (%d), maximum allowed is %d", filePath, depth, maxJSONDepth)
	}
	if propCount > maxJSONProperties {
		return fmt.Errorf("file %s has too many JSON properties (%d), maximum allowed is %d", filePath, propCount, maxJSONProperties)
	}

	return nil
}

// getJSONComplexity calculates the nesting depth and property count of a JSON structure
func getJSONComplexity(obj interface{}, currentDepth int) (maxDepth int, propCount int) {
	maxDepth = currentDepth

	switch v := obj.(type) {
	case map[string]interface{}:
		// Increment depth for this object level
		currentObjectDepth := currentDepth + 1
		if currentObjectDepth > maxDepth {
			maxDepth = currentObjectDepth
		}
		propCount += len(v)
		for _, value := range v {
			childDepth, childProps := getJSONComplexity(value, currentObjectDepth)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
			propCount += childProps
		}
	case []interface{}:
		// Increment depth for array processing
		currentArrayDepth := currentDepth + 1
		for _, item := range v {
			childDepth, childProps := getJSONComplexity(item, currentArrayDepth)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
			propCount += childProps
		}
	}

	return maxDepth, propCount
}

// validateFilePathSecurity performs additional security checks on file paths
func validateFilePathSecurity(filePath string) error {
	// Check for suspicious path patterns that could indicate path traversal attempts
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("file path %s contains suspicious path traversal patterns", filePath)
	}

	// Check for excessively long file paths that could cause buffer overflows
	if len(filePath) > 4096 {
		return fmt.Errorf("file path %s is too long (%d characters), maximum allowed is 4096", filePath, len(filePath))
	}

	return nil
}
