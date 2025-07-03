package utils

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func writeTestConfig(t *testing.T, fs afero.Fs, path string, m map[string]interface{}) {
	data, err := json.MarshalIndent(m, "", "  ")
	assert.NoError(t, err)
	err = WriteFile(fs, path, data, 0644)
	assert.NoError(t, err)
}

func writeTestSchema(t *testing.T, fs afero.Fs, path string) {
	// Minimal schema: allow any object
	err := WriteFile(fs, path, []byte(`{"type":"object"}`), 0644)
	assert.NoError(t, err)
}

func TestLoadConfigCommand_ValidAndInvalid(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	// Valid config
	valid := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"maxCacheSize": 10,
		},
	}
	validFile, err := CreateTempFile(fs, "/tmp", "valid_*.json")
	assert.NoError(t, err)
	validFile.Close()
	defer os.Remove(validFile.Name())
	writeTestConfig(t, fs, validFile.Name(), valid)

	op := &ConfigOperation{}
	err = op.LoadConfigCommand(validFile.Name(), "")
	assert.NoError(t, err)

	// Invalid config (not JSON)
	invalidFile, err := CreateTempFile(fs, "/tmp", "invalid_*.json")
	assert.NoError(t, err)
	invalidFile.Close()
	defer os.Remove(invalidFile.Name())
	err = WriteFile(fs, invalidFile.Name(), []byte("{notjson"), 0644)
	assert.NoError(t, err)
	err = op.LoadConfigCommand(invalidFile.Name(), "")
	assert.Error(t, err)
}

func TestSetConfigCommand(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	initial := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"maxCacheSize": 10,
		},
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	// Valid set
	err = op.SetConfigCommand("os_updater.maxCacheSize:42")
	assert.NoError(t, err)
	val, _, err := op.GetConfigCommand("os_updater.maxCacheSize")
	assert.NoError(t, err)
	assert.Equal(t, "42", val)

	// Invalid format
	err = op.SetConfigCommand("invalidformat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestGetConfigCommand(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()
	initial := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"maxCacheSize":           99,
			"proceedWithoutRollback": true,
		},
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	// Existing key
	val, _, err := op.GetConfigCommand("os_updater.maxCacheSize")
	assert.NoError(t, err)
	assert.Equal(t, "99", val)

	// Non-existing key
	_, _, err = op.GetConfigCommand("os_updater.notfound")
	assert.Error(t, err)
}

func TestAppendAndRemoveConfigCommand(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	initial := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"trustedRepositories": []interface{}{"https://abc.com/"},
		},
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	// Append allowed
	err = op.AppendConfigCommand("os_updater.trustedRepositories:https://def.com/")
	assert.NoError(t, err)
	val, _, err := op.GetConfigCommand("os_updater.trustedRepositories")
	assert.NoError(t, err)
	assert.Contains(t, val, "https://def.com/")

	// Remove allowed
	err = op.RemoveConfigCommand("os_updater.trustedRepositories:https://abc.com/")
	assert.NoError(t, err)
	val, _, err = op.GetConfigCommand("os_updater.trustedRepositories")
	assert.NoError(t, err)
	assert.NotContains(t, val, "https://abc.com/")

	// Append not allowed
	err = op.AppendConfigCommand("os_updater.notAllowed:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "append not supported")

	// Remove not allowed
	err = op.RemoveConfigCommand("os_updater.notAllowed:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove not supported")
}

func TestSetJSONPath_InvalidPath(t *testing.T) {
	m := map[string]interface{}{}
	// Try to set a value with an invalid path (e.g., empty string)
	err := setJSONPath(m, "", 123)
	assert.Error(t, err)
}

func TestGetJSONPath_InvalidPath(t *testing.T) {
	m := map[string]interface{}{}
	_, err := getJSONPath(m, "")
	assert.Error(t, err)
}

func TestValidateConfigMapWithSchema_InvalidJSON(t *testing.T) {
	fs := afero.NewOsFs()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	// Pass a map with a channel (not JSON serializable)
	m := map[string]interface{}{"bad": make(chan int)}
	err = validateConfigMapWithSchema(m)
	assert.Error(t, err)
}

func TestSetConfigCommand_SchemaValidationFail(t *testing.T) {
	fs := afero.NewOsFs()

	// Use a schema that requires a specific property
	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	// Schema requires "mustExist"
	err = WriteFile(fs, schemaFilePath, []byte(`{"type":"object","required":["mustExist"]}`), 0644)
	assert.NoError(t, err)

	// Initial config is empty
	writeTestConfig(t, fs, configFilePath, map[string]interface{}{})
	op := &ConfigOperation{}

	// Setting a key that doesn't satisfy schema
	err = op.SetConfigCommand("foo:bar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config validation failed")
}

func TestAppendConfigCommand_InvalidFormat(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()
	writeTestConfig(t, fs, configFilePath, map[string]interface{}{}) // Write empty config

	op := &ConfigOperation{}
	err = op.AppendConfigCommand("missingcolon")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestRemoveConfigCommand_InvalidFormat(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()
	writeTestConfig(t, fs, configFilePath, map[string]interface{}{}) // Write empty config

	op := &ConfigOperation{}
	err = op.RemoveConfigCommand("missingcolon")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestIsAppendRemovePathAllowed_CaseInsensitive(t *testing.T) {
	assert.False(t, isAppendRemovePathAllowed("OS_UPDATER.TRUSTEDREPOSITORIES"))
}

func TestValidateConfigMapWithSchema_InvalidSchemaFile(t *testing.T) {
	// Use a non-existent schema file
	schemaFilePath = "/tmp/does_not_exist_schema.json"
	m := map[string]interface{}{"foo": "bar"}
	err := validateConfigMapWithSchema(m)
	assert.Error(t, err)
}

func TestSetConfigCommand_NoPartialUpdateOnSchemaFail(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	// Schema only allows "foo"
	err = WriteFile(fs, schemaFilePath, []byte(`{"type":"object","properties":{"foo":{"type":"string"}},"additionalProperties":false}`), 0644)
	assert.NoError(t, err)

	// Initial config
	writeTestConfig(t, fs, configFilePath, map[string]interface{}{"foo": "bar"})
	op := &ConfigOperation{}

	// Try to set one valid and one invalid key
	err = op.SetConfigCommand("foo:baz;bar:bad")
	assert.Error(t, err)
	// Config should remain unchanged
	data, err := ReadFile(fs, configFilePath)
	assert.NoError(t, err)
	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "bar", m["foo"])
	assert.Nil(t, m["bar"])
}

func TestAppendConfigCommand_NoPartialUpdateOnPathFail(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	initial := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"trustedRepositories": []interface{}{"https://abc.com/"},
		},
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	// Try to append to one allowed and one not allowed path
	err = op.AppendConfigCommand("os_updater.trustedRepositories:https://def.com/;os_updater.notAllowed:foo")
	assert.Error(t, err)
	// Config should remain unchanged
	val, _, err := op.GetConfigCommand("os_updater.trustedRepositories")
	assert.NoError(t, err)
	assert.NotContains(t, val, "https://def.com/")
}

func TestRemoveConfigCommand_NoPartialUpdateOnPathFail(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()

	tmpSchema, err := CreateTempFile(fs, "/tmp", "test_schema_*.json")
	assert.NoError(t, err)
	tmpSchema.Close()
	defer os.Remove(tmpSchema.Name())
	schemaFilePath = tmpSchema.Name()
	writeTestSchema(t, fs, schemaFilePath)

	initial := map[string]interface{}{
		"os_updater": map[string]interface{}{
			"trustedRepositories": []interface{}{"https://abc.com/", "https://def.com/"},
		},
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	// Try to remove from one allowed and one not allowed path
	err = op.RemoveConfigCommand("os_updater.trustedRepositories:https://abc.com/;os_updater.notAllowed:foo")
	assert.Error(t, err)
	// Config should remain unchanged
	val, _, err := op.GetConfigCommand("os_updater.trustedRepositories")
	assert.NoError(t, err)
	assert.Contains(t, val, "https://abc.com/")
	assert.Contains(t, val, "https://def.com/")
}

func TestGetConfigCommand_EmptyOrWhitespacePath(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()
	writeTestConfig(t, fs, configFilePath, map[string]interface{}{"foo": "bar"})
	op := &ConfigOperation{}

	_, _, err = op.GetConfigCommand("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")

	_, _, err = op.GetConfigCommand("   ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestGetConfigCommand_MultiKeyPartialResult(t *testing.T) {
	fs := afero.NewOsFs()

	tmpConfig, err := CreateTempFile(fs, "/tmp", "test_config_*.json")
	assert.NoError(t, err)
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())
	configFilePath = tmpConfig.Name()
	initial := map[string]interface{}{
		"foo": "bar",
		"baz": 42,
	}
	writeTestConfig(t, fs, configFilePath, initial)
	op := &ConfigOperation{}

	val, errStr, err := op.GetConfigCommand("foo;notfound;baz")
	assert.NoError(t, err)
	assert.Equal(t, "bar;;42", val)
	assert.Contains(t, errStr, "notfound")
}
