/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"os"

	"github.com/spf13/afero"
)

// FileSystemOperations defines the interface for file system operations
// This allows for easy mocking and dependency injection in tests
type FileSystemOperations interface {
	// Open opens a file for reading
	Open(name string) (afero.File, error)
	// Stat returns file information
	Stat(name string) (os.FileInfo, error)
	// ReadFile reads the entire content of a file
	ReadFile(filename string) ([]byte, error)
}

// SchemaValidator defines the interface for JSON schema validation
// This allows for mocking schema validation logic in tests
type SchemaValidator interface {
	// ValidateConfig validates configuration data against a schema
	ValidateConfig(schemaData, configData []byte) error
}

// ConfigReader defines the interface for configuration file reading and validation
// This provides a high-level abstraction for configuration operations
type ConfigReader interface {
	// ReadAndValidateConfig reads and validates the firmware configuration
	ReadAndValidateConfig() ([]byte, error)
	// GetConfigFilePath returns the path to the configuration file
	GetConfigFilePath() string
	// GetSchemaFilePath returns the path to the schema file
	GetSchemaFilePath() string
}

// PlatformConfigProvider defines the interface for retrieving platform-specific configuration
type PlatformConfigProvider interface {
	// GetPlatformConfig returns the firmware tool configuration for a specific platform
	GetPlatformConfig(platformName string) (FirmwareToolInfo, error)
}
