/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

// MockFileSystemOperations provides a mock implementation for testing
type MockFileSystemOperations struct {
	// Data stores file contents by path
	Data map[string][]byte
	// FileInfos stores file information by path
	FileInfos map[string]os.FileInfo
	// Errors maps file paths to errors that should be returned
	Errors map[string]error
	// SimulatePermissionDenied enables permission denied simulation
	SimulatePermissionDenied bool
	// SimulateFileLocked enables file locking simulation
	SimulateFileLocked bool
	// SimulateBusyDevice enables device busy simulation
	SimulateBusyDevice bool
	// SimulateInterrupted enables interrupted system call simulation
	SimulateInterrupted bool
	// SimulateReadOnlyFS enables read-only filesystem simulation
	SimulateReadOnlyFS bool
}

// NewMockFileSystemOperations creates a new MockFileSystemOperations instance
func NewMockFileSystemOperations() *MockFileSystemOperations {
	return &MockFileSystemOperations{
		Data:      make(map[string][]byte),
		FileInfos: make(map[string]os.FileInfo),
		Errors:    make(map[string]error),
	}
}

// SetFileContent sets the content for a file path
func (m *MockFileSystemOperations) SetFileContent(path string, content []byte) {
	m.Data[path] = content
	m.FileInfos[path] = &mockFileInfo{
		name: path,
		size: int64(len(content)),
		mode: 0644,
	}
}

// SetFileError sets an error to be returned for a specific file path
func (m *MockFileSystemOperations) SetFileError(path string, err error) {
	m.Errors[path] = err
}

// SimulatePermissionError enables permission denied errors for all operations
func (m *MockFileSystemOperations) SimulatePermissionError() {
	m.SimulatePermissionDenied = true
}

// SimulateFileLockError enables file locking errors for all operations
func (m *MockFileSystemOperations) SimulateFileLockError() {
	m.SimulateFileLocked = true
}

// Open opens a file for reading (mock implementation)
func (m *MockFileSystemOperations) Open(name string) (afero.File, error) {
	return nil, m.checkErrors(name, "open")
}

// Stat returns file information (mock implementation)
func (m *MockFileSystemOperations) Stat(name string) (os.FileInfo, error) {
	if err := m.checkErrors(name, "stat"); err != nil {
		return nil, err
	}

	if info, exists := m.FileInfos[name]; exists {
		return info, nil
	}

	return nil, &os.PathError{Op: "stat", Path: name, Err: syscall.ENOENT}
}

// ReadFile reads the entire content of a file (mock implementation)
func (m *MockFileSystemOperations) ReadFile(filename string) ([]byte, error) {
	if err := m.checkErrors(filename, "read"); err != nil {
		return nil, err
	}

	if data, exists := m.Data[filename]; exists {
		return data, nil
	}

	return nil, &os.PathError{Op: "open", Path: filename, Err: syscall.ENOENT}
}

// checkErrors checks for simulated errors and returns appropriate error types
func (m *MockFileSystemOperations) checkErrors(name, op string) error {
	// Check for specific file errors first
	if err, exists := m.Errors[name]; exists {
		return err
	}

	// Check for global error simulations
	if m.SimulatePermissionDenied {
		return &os.PathError{Op: op, Path: name, Err: syscall.EACCES}
	}

	if m.SimulateFileLocked {
		return &os.PathError{Op: op, Path: name, Err: syscall.EAGAIN}
	}

	if m.SimulateBusyDevice {
		return &os.PathError{Op: op, Path: name, Err: syscall.EBUSY}
	}

	if m.SimulateInterrupted {
		return &os.PathError{Op: op, Path: name, Err: syscall.EINTR}
	}

	if m.SimulateReadOnlyFS {
		return &os.PathError{Op: op, Path: name, Err: syscall.EROFS}
	}

	return nil
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name string
	size int64
	mode os.FileMode
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// MockSchemaValidator provides a mock implementation for testing
type MockSchemaValidator struct {
	// ValidationResult determines whether validation should pass or fail
	ValidationResult error
	// ValidationCalls tracks the calls made to ValidateConfig
	ValidationCalls []ValidationCall
}

// ValidationCall represents a call to ValidateConfig
type ValidationCall struct {
	SchemaData []byte
	ConfigData []byte
}

// NewMockSchemaValidator creates a new MockSchemaValidator instance
func NewMockSchemaValidator() *MockSchemaValidator {
	return &MockSchemaValidator{
		ValidationCalls: make([]ValidationCall, 0),
	}
}

// SetValidationResult sets the result that ValidateConfig should return
func (m *MockSchemaValidator) SetValidationResult(err error) {
	m.ValidationResult = err
}

// SetValidationSuccess sets validation to always succeed
func (m *MockSchemaValidator) SetValidationSuccess() {
	m.ValidationResult = nil
}

// SetValidationFailure sets validation to always fail with a specific error
func (m *MockSchemaValidator) SetValidationFailure(message string) {
	m.ValidationResult = errors.New(message)
}

// ValidateConfig validates configuration data against a schema (mock implementation)
func (m *MockSchemaValidator) ValidateConfig(schemaData, configData []byte) error {
	// Record the call
	m.ValidationCalls = append(m.ValidationCalls, ValidationCall{
		SchemaData: schemaData,
		ConfigData: configData,
	})

	return m.ValidationResult
}

// GetValidationCallCount returns the number of times ValidateConfig was called
func (m *MockSchemaValidator) GetValidationCallCount() int {
	return len(m.ValidationCalls)
}

// GetLastValidationCall returns the last call to ValidateConfig, or nil if no calls were made
func (m *MockSchemaValidator) GetLastValidationCall() *ValidationCall {
	if len(m.ValidationCalls) == 0 {
		return nil
	}
	return &m.ValidationCalls[len(m.ValidationCalls)-1]
}

// MockConfigReader provides a mock implementation for testing
type MockConfigReader struct {
	// ConfigData is the data that ReadAndValidateConfig should return
	ConfigData []byte
	// ReadError is the error that ReadAndValidateConfig should return
	ReadError error
	// ConfigPath is the path to the configuration file
	ConfigPath string
	// SchemaPath is the path to the schema file
	SchemaPath string
	// ReadCalls tracks the number of times ReadAndValidateConfig was called
	ReadCalls int
}

// NewMockConfigReader creates a new MockConfigReader instance
func NewMockConfigReader(configPath, schemaPath string) *MockConfigReader {
	return &MockConfigReader{
		ConfigPath: configPath,
		SchemaPath: schemaPath,
	}
}

// SetConfigData sets the configuration data to return
func (m *MockConfigReader) SetConfigData(data []byte) {
	m.ConfigData = data
}

// SetReadError sets the error to return from ReadAndValidateConfig
func (m *MockConfigReader) SetReadError(err error) {
	m.ReadError = err
}

// ReadAndValidateConfig reads and validates the firmware configuration (mock implementation)
func (m *MockConfigReader) ReadAndValidateConfig() ([]byte, error) {
	m.ReadCalls++
	if m.ReadError != nil {
		return nil, m.ReadError
	}
	return m.ConfigData, nil
}

// GetConfigFilePath returns the path to the configuration file
func (m *MockConfigReader) GetConfigFilePath() string {
	return m.ConfigPath
}

// GetSchemaFilePath returns the path to the schema file
func (m *MockConfigReader) GetSchemaFilePath() string {
	return m.SchemaPath
}

// GetReadCallCount returns the number of times ReadAndValidateConfig was called
func (m *MockConfigReader) GetReadCallCount() int {
	return m.ReadCalls
}

// MockPlatformConfigProvider provides a mock implementation for testing
type MockPlatformConfigProvider struct {
	// Configs maps platform names to firmware tool configurations
	Configs map[string]FirmwareToolInfo
	// Errors maps platform names to errors that should be returned
	Errors map[string]error
	// ConfigCalls tracks the calls made to GetPlatformConfig
	ConfigCalls []string
}

// NewMockPlatformConfigProvider creates a new MockPlatformConfigProvider instance
func NewMockPlatformConfigProvider() *MockPlatformConfigProvider {
	return &MockPlatformConfigProvider{
		Configs:     make(map[string]FirmwareToolInfo),
		Errors:      make(map[string]error),
		ConfigCalls: make([]string, 0),
	}
}

// SetPlatformConfig sets the configuration for a specific platform
func (m *MockPlatformConfigProvider) SetPlatformConfig(platformName string, config FirmwareToolInfo) {
	m.Configs[platformName] = config
}

// SetPlatformError sets an error to be returned for a specific platform
func (m *MockPlatformConfigProvider) SetPlatformError(platformName string, err error) {
	m.Errors[platformName] = err
}

// GetPlatformConfig returns the firmware tool configuration for a specific platform (mock implementation)
func (m *MockPlatformConfigProvider) GetPlatformConfig(platformName string) (FirmwareToolInfo, error) {
	// Record the call
	m.ConfigCalls = append(m.ConfigCalls, platformName)

	// Check for specific errors first
	if err, exists := m.Errors[platformName]; exists {
		return FirmwareToolInfo{}, err
	}

	// Return the configuration if it exists
	if config, exists := m.Configs[platformName]; exists {
		return config, nil
	}

	// Return platform not found error
	return FirmwareToolInfo{}, fmt.Errorf("platform %s not found in config", platformName)
}

// GetConfigCallCount returns the number of times GetPlatformConfig was called
func (m *MockPlatformConfigProvider) GetConfigCallCount() int {
	return len(m.ConfigCalls)
}

// GetLastConfigCall returns the last platform name passed to GetPlatformConfig
func (m *MockPlatformConfigProvider) GetLastConfigCall() string {
	if len(m.ConfigCalls) == 0 {
		return ""
	}
	return m.ConfigCalls[len(m.ConfigCalls)-1]
}

// GetAllConfigCalls returns all platform names passed to GetPlatformConfig
func (m *MockPlatformConfigProvider) GetAllConfigCalls() []string {
	return append([]string(nil), m.ConfigCalls...) // Return a copy
}
