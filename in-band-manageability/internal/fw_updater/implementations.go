/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
	"github.com/xeipuuv/gojsonschema"
)

// AferoFileSystemOperations provides a concrete implementation of FileSystemOperations using afero
type AferoFileSystemOperations struct {
	fs afero.Fs
}

// NewAferoFileSystemOperations creates a new AferoFileSystemOperations instance
func NewAferoFileSystemOperations(fs afero.Fs) *AferoFileSystemOperations {
	return &AferoFileSystemOperations{fs: fs}
}

// Open opens a file for reading
func (a *AferoFileSystemOperations) Open(name string) (afero.File, error) {
	return a.fs.Open(name)
}

// Stat returns file information
func (a *AferoFileSystemOperations) Stat(name string) (os.FileInfo, error) {
	return a.fs.Stat(name)
}

// ReadFile reads the entire content of a file using afero directly
// This bypasses the utils.ReadFile security restrictions for interface-based testing
func (a *AferoFileSystemOperations) ReadFile(filename string) ([]byte, error) {
	return afero.ReadFile(a.fs, filename)
}

// ProductionFileSystemOperations provides a secure implementation for production use
// It uses utils.ReadFile which enforces security restrictions on file paths
type ProductionFileSystemOperations struct {
	fs afero.Fs
}

// NewProductionFileSystemOperations creates a new ProductionFileSystemOperations instance
func NewProductionFileSystemOperations(fs afero.Fs) *ProductionFileSystemOperations {
	return &ProductionFileSystemOperations{fs: fs}
}

// Open opens a file for reading
func (p *ProductionFileSystemOperations) Open(name string) (afero.File, error) {
	return p.fs.Open(name)
}

// Stat returns file information
func (p *ProductionFileSystemOperations) Stat(name string) (os.FileInfo, error) {
	return p.fs.Stat(name)
}

// ReadFile reads the entire content of a file using utils.ReadFile with security restrictions
func (p *ProductionFileSystemOperations) ReadFile(filename string) ([]byte, error) {
	return utils.ReadFile(p.fs, filename)
}

// GoJSONSchemaValidator provides JSON schema validation using the gojsonschema library
type GoJSONSchemaValidator struct{}

// NewGoJSONSchemaValidator creates a new GoJSONSchemaValidator instance
func NewGoJSONSchemaValidator() *GoJSONSchemaValidator {
	return &GoJSONSchemaValidator{}
}

// ValidateConfig validates configuration data against a JSON schema
func (v *GoJSONSchemaValidator) ValidateConfig(schemaData, configData []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewBytesLoader(configData)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		return fmt.Errorf("config file does not match schema: %v", result.Errors())
	}
	return nil
}

// SecureConfigReader provides secure configuration file reading with validation
type SecureConfigReader struct {
	fileOps         FileSystemOperations
	schemaValidator SchemaValidator
	configPath      string
	schemaPath      string
}

// NewSecureConfigReader creates a new SecureConfigReader instance
func NewSecureConfigReader(fileOps FileSystemOperations, validator SchemaValidator, configPath, schemaPath string) *SecureConfigReader {
	return &SecureConfigReader{
		fileOps:         fileOps,
		schemaValidator: validator,
		configPath:      configPath,
		schemaPath:      schemaPath,
	}
}

// ReadAndValidateConfig reads and validates the firmware configuration
func (r *SecureConfigReader) ReadAndValidateConfig() ([]byte, error) {
	// Read and validate schema file with security checks
	schemaData, err := r.secureReadFile(r.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Read and validate config file with security checks
	configData, err := r.secureReadFile(r.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate config against schema
	if err := r.schemaValidator.ValidateConfig(schemaData, configData); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return configData, nil
}

// GetConfigFilePath returns the path to the configuration file
func (r *SecureConfigReader) GetConfigFilePath() string {
	return r.configPath
}

// GetSchemaFilePath returns the path to the schema file
func (r *SecureConfigReader) GetSchemaFilePath() string {
	return r.schemaPath
}

// secureReadFile reads a file with comprehensive security validations
func (r *SecureConfigReader) secureReadFile(filePath string) ([]byte, error) {
	// Validate file path security first
	if err := validateFilePathSecurity(filePath); err != nil {
		return nil, err
	}

	// Validate file size before reading
	if err := r.validateFileSize(filePath); err != nil {
		return nil, err
	}

	// Read the file
	data, err := r.fileOps.ReadFile(filePath)
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

// validateFileSize checks if the file size is within acceptable limits
func (r *SecureConfigReader) validateFileSize(filePath string) error {
	fileInfo, err := r.fileOps.Stat(filePath)
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

// FirmwarePlatformConfigProvider provides platform-specific firmware configuration
type FirmwarePlatformConfigProvider struct {
	configReader ConfigReader
}

// NewFirmwarePlatformConfigProvider creates a new FirmwarePlatformConfigProvider instance
func NewFirmwarePlatformConfigProvider(configReader ConfigReader) *FirmwarePlatformConfigProvider {
	return &FirmwarePlatformConfigProvider{configReader: configReader}
}

// GetPlatformConfig returns the firmware tool configuration for a specific platform
func (p *FirmwarePlatformConfigProvider) GetPlatformConfig(platformName string) (FirmwareToolInfo, error) {
	configData, err := p.configReader.ReadAndValidateConfig()
	if err != nil {
		return FirmwareToolInfo{}, fmt.Errorf("failed to validate firmware tool config: %w", err)
	}

	return extractPlatformConfig(configData, platformName)
}

// extractPlatformConfig extracts platform-specific configuration from JSON data
func extractPlatformConfig(configData []byte, platformName string) (FirmwareToolInfo, error) {
	// Define a structure to match the JSON structure
	var configRoot struct {
		FirmwareComponent struct {
			FirmwareProducts []FirmwareToolInfo `json:"firmware_products"`
		} `json:"firmware_component"`
	}

	if err := json.Unmarshal(configData, &configRoot); err != nil {
		return FirmwareToolInfo{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Find the matching platformName
	for _, info := range configRoot.FirmwareComponent.FirmwareProducts {
		if info.Name == platformName {
			return info, nil
		}
	}

	return FirmwareToolInfo{}, fmt.Errorf("platform %s not found in config. Please add firmware update configuration information and try again", platformName)
}
