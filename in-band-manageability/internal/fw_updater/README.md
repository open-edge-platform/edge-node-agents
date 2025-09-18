<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Interface-Based Dependency Injection for Firmware Configuration

This package now supports a modern interface-based approach for handling firmware configuration, which eliminates reliance on the file system's state and enables comprehensive mocking for unit tests.

## Overview

The traditional approach required actual files on the filesystem and was difficult to test reliably. The new interface-based approach uses dependency injection with well-defined interfaces, allowing for easy mocking and more robust testing.

## Key Components

### Interfaces

- **`FileSystemOperations`**: Abstracts file system operations (Open, Stat, ReadFile)
- **`SchemaValidator`**: Abstracts JSON schema validation
- **`ConfigReader`**: Abstracts configuration file reading and validation
- **`PlatformConfigProvider`**: Abstracts platform-specific configuration retrieval

### Implementations

- **`AferoFileSystemOperations`**: Basic afero-based file operations (for testing)
- **`ProductionFileSystemOperations`**: Secure file operations using utils.ReadFile (for production)
- **`GoJSONSchemaValidator`**: JSON schema validation using gojsonschema
- **`SecureConfigReader`**: Secure configuration reading with validation
- **`FirmwarePlatformConfigProvider`**: Platform configuration retrieval

### Mock Implementations

- **`MockFileSystemOperations`**: Full-featured mock for file system operations
- **`MockSchemaValidator`**: Mock for schema validation
- **`MockConfigReader`**: Mock for configuration reading
- **`MockPlatformConfigProvider`**: Mock for platform configuration

## Usage Examples

### Production Usage

```go
// Create a standard config loader
loader := NewConfigLoader()

// Load configuration for a platform
config, err := loader.LoadPlatformConfig("Alder Lake Client Platform")
if err != nil {
    return err
}
```

### Testing with Mocks

```go
// Create mock provider
mockProvider := NewMockPlatformConfigProvider()
testConfig := FirmwareToolInfo{
    Name:       "Test Platform",
    BiosVendor: "Test Vendor",
}
mockProvider.SetPlatformConfig("Test Platform", testConfig)

// Create loader with mock
loader := NewConfigLoaderWithProvider(mockProvider)

// Test without touching filesystem
config, err := loader.LoadPlatformConfig("Test Platform")
assert.NoError(t, err)
assert.Equal(t, testConfig, config)
```

### Testing Error Scenarios

```go
// Test permission denied
mockFS := NewMockFileSystemOperations()
mockFS.SimulatePermissionError()

configReader := NewSecureConfigReader(mockFS, validator, configPath, schemaPath)
provider := NewFirmwarePlatformConfigProvider(configReader)

_, err := provider.GetPlatformConfig("Any Platform")
assert.Error(t, err)
assert.Contains(t, err.Error(), "permission denied")
```

### Testing with In-Memory Filesystem

```go
factory := NewConfigLoaderFactory()
loader := factory.CreateMemoryLoader(configData, schemaData)

config, err := loader.LoadPlatformConfig("Test Platform")
assert.NoError(t, err)
```

## Benefits

### For Testing

1. **No File Dependencies**: Tests run without creating actual files
2. **Predictable Behavior**: Mocks provide consistent, controllable responses
3. **Error Simulation**: Easy to test permission errors, file locking, etc.
4. **Fast Execution**: In-memory operations are much faster than disk I/O
5. **Parallel Safety**: No file contention between parallel tests
6. **Clean Environment**: No temporary file cleanup needed

### For Production

1. **Security**: Production implementation uses secure utils.ReadFile
2. **Backward Compatibility**: Existing APIs continue to work unchanged
3. **Flexibility**: Easy to swap implementations for different environments
4. **Testability**: Production code can be tested with mocked dependencies

## Migration Guide

### Existing Code

Existing code continues to work without changes:

```go
// This still works exactly as before
config, err := GetFirmwareUpdateToolInfo(fs, platformName)
```

### New Test Code

New tests should use the interface-based approach:

```go
// Old approach (harder to test)
func TestOldWay(t *testing.T) {
    fs := afero.NewMemMapFs()
    // Write files to filesystem...
    config, err := GetFirmwareUpdateToolInfo(fs, "platform")
    // ...
}

// New approach (easy to test)
func TestNewWay(t *testing.T) {
    mockProvider := NewMockPlatformConfigProvider()
    mockProvider.SetPlatformConfig("platform", expectedConfig)
    
    loader := NewConfigLoaderWithProvider(mockProvider)
    config, err := loader.LoadPlatformConfig("platform")
    // ...
}
```

## Error Handling

The interface-based approach provides comprehensive error handling for:

- **File Permission Errors**: EACCES, permission denied
- **File Locking Errors**: EAGAIN, resource temporarily unavailable
- **Device Errors**: EBUSY, EINTR, EROFS
- **Validation Errors**: Schema validation failures
- **Configuration Errors**: Malformed JSON, missing platforms

## Best Practices

1. **Use Mocks in Unit Tests**: Avoid real file system dependencies
2. **Test Error Scenarios**: Use mocks to simulate various error conditions
3. **Validate Mock Interactions**: Check that mocks are called correctly
4. **Use Factory Pattern**: For complex dependency setups
5. **Maintain Backward Compatibility**: Keep existing APIs working

## File Structure

```
internal/fw_updater/
├── interfaces.go          # Interface definitions
├── implementations.go     # Production implementations
├── mocks.go              # Mock implementations for testing
├── config_loader.go      # High-level loader with factory methods
├── config_loader_test.go # Comprehensive tests using mocks
├── config_interface_test.go # Interface-specific tests
├── examples.go           # Usage examples and documentation
├── config_finder.go      # Original implementation (backward compatible)
└── config_finder_test.go # Original tests (still working)
```

This structure separates concerns clearly while maintaining backward compatibility and providing comprehensive testing capabilities.
