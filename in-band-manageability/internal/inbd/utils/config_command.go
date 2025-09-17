/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

var (
	ConfigurationAppendRemovePathsList = []string{"sotaSW", "trustedRepositories"}
	configFilePath                     = "/etc/intel_manageability.conf"
	schemaFilePath                     = "/usr/share/inbd_schema.json"
)

type ConfigOperation struct {
	mu sync.RWMutex
}

// LoadConfigCommand copies the file at uri to /etc/intel_manageability.conf
func (c *ConfigOperation) LoadConfigCommand(uri, signature, hashAlgorithm string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Default to sha384 if not provided
	finalHashAlgorithm := "sha384"
	if hashAlgorithm != "" {
		switch strings.ToLower(hashAlgorithm) {
		case "sha256", "sha384", "sha512":
			finalHashAlgorithm = strings.ToLower(hashAlgorithm)
		default:
			return fmt.Errorf("invalid hash algorithm: %s (must be 'sha256', 'sha384', or 'sha512')", hashAlgorithm)
		}
	}

	// Verify signature if provided
	if signature != "" {
		if err := VerifySignature(signature, uri, ParseHashAlgorithm(finalHashAlgorithm)); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	fs := afero.Afero{Fs: afero.NewOsFs()}

	// Determine if the input is a tar file
	isTar := strings.HasSuffix(strings.ToLower(uri), ".tar")

	var input []byte

	if isTar {
		// Extract intel_manageability.conf from tar to a temp file
		tarFile, err := fs.Open(uri)
		if err != nil {
			return fmt.Errorf("failed to open tar file: %w", err)
		}
		defer tarFile.Close()

		tr := tar.NewReader(tarFile)
		found := false
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read tar file: %w", err)
			}
			if filepath.Base(hdr.Name) == "intel_manageability.conf" {
				input, err = io.ReadAll(tr)
				if err != nil {
					return fmt.Errorf("failed to read config from tar: %w", err)
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("intel_manageability.conf not found in tar archive")
		}
	} else {
		var err error
		input, err = ReadFile(fs.Fs, uri)
		if err != nil {
			return fmt.Errorf("failed to read new config: %w", err)
		}
	}

	// Validate input against schema before writing
	tmpFile, err := CreateTempFile(fs.Fs, "", "inbd_config_load_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file for schema validation: %w", err)
	}
	defer func() {
		if err := RemoveFile(fs.Fs, tmpFile.Name()); err != nil {
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", tmpFile.Name(), err)
		}
	}()
	defer tmpFile.Close()

	if _, err := tmpFile.Write(input); err != nil {
		return fmt.Errorf("failed to write temp config for schema validation: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp config for schema validation: %w", err)
	}

	valid, err := IsValidJSON(fs, schemaFilePath, tmpFile.Name())
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}
	if !valid {
		return errors.New("configuration file is not valid according to schema")
	}

	// Only write if valid
	if err := WriteFile(fs.Fs, configFilePath, input, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// GetConfigCommand returns the value for a key in /etc/intel_manageability.conf
func (c *ConfigOperation) GetConfigCommand(key string) (string, string, error) {
	c.mu.RLock() // Use read lock instead of exclusive lock
	defer c.mu.RUnlock()

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "path is required", errors.New("path is required")
	}

	fs := afero.Afero{Fs: afero.NewOsFs()}
	data, err := ReadFile(fs.Fs, configFilePath)
	if err != nil {
		return "", fmt.Sprintf("failed to read config: %v", err), err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Sprintf("invalid config JSON: %v", err), err
	}

	keys := strings.Split(key, ";")
	var results []string
	var errorsFound []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			results = append(results, "")
			continue
		}
		val, err := getJSONPath(m, k)
		if err != nil {
			errorsFound = append(errorsFound, fmt.Sprintf("%s: %v", k, err))
			results = append(results, "")
		} else {
			results = append(results, fmt.Sprintf("%v", val))
		}
	}
	errorStr := ""
	if len(errorsFound) > 0 {
		errorStr = strings.Join(errorsFound, "; ")
	}
	if len(errorsFound) == len(keys) {
		// All failed
		return "", errorStr, errors.New(errorStr)
	}
	// Partial or full success
	return strings.Join(results, ";"), errorStr, nil
}

// SetConfigCommand expects keyValue(s) in "key:value;key2:value2" format.
func (c *ConfigOperation) SetConfigCommand(keyValues string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fs := afero.Afero{Fs: afero.NewOsFs()}
	data, err := ReadFile(fs.Fs, configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}

	pairs := strings.Split(keyValues, ";")
	for _, keyValue := range pairs {
		keyValue = strings.TrimSpace(keyValue)
		if keyValue == "" {
			continue
		}
		parts := strings.SplitN(keyValue, ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid format, expected key:value")
		}
		key, value := parts[0], parts[1]

		// Try to parse value as int, bool, or leave as string
		var v interface{} = value
		if i, err := strconv.Atoi(value); err == nil {
			v = i
		} else if b, err := strconv.ParseBool(value); err == nil {
			v = b
		}

		if err := setJSONPath(m, key, v); err != nil {
			return err
		}
	}

	if err := validateConfigMapWithSchema(m); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := WriteFile(fs.Fs, configFilePath, out, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// AppendConfigCommand expects keyValue(s) in "key:value;key2:value2" format.
func (c *ConfigOperation) AppendConfigCommand(keyValues string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fs := afero.Afero{Fs: afero.NewOsFs()}
	data, err := ReadFile(fs.Fs, configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}

	pairs := strings.Split(keyValues, ";")
	for _, keyValue := range pairs {
		keyValue = strings.TrimSpace(keyValue)
		if keyValue == "" {
			continue
		}
		parts := strings.SplitN(keyValue, ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid format, expected key:value")
		}
		key, value := parts[0], parts[1]
		if !isAppendRemovePathAllowed(key) {
			return errors.New("append not supported for this path")
		}
		arr, err := getJSONPath(m, key)
		if err != nil {
			return err
		}
		list, ok := arr.([]interface{})
		if !ok {
			return errors.New("target is not a list")
		}
		list = append(list, value)
		if err := setJSONPath(m, key, list); err != nil {
			return err
		}
	}

	if err := validateConfigMapWithSchema(m); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := WriteFile(fs.Fs, configFilePath, out, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// RemoveConfigCommand expects keyValue(s) in "key:value;key2:value2" format.
func (c *ConfigOperation) RemoveConfigCommand(keyValues string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fs := afero.Afero{Fs: afero.NewOsFs()}
	data, err := ReadFile(fs.Fs, configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}

	pairs := strings.Split(keyValues, ";")
	for _, keyValue := range pairs {
		keyValue = strings.TrimSpace(keyValue)
		if keyValue == "" {
			continue
		}
		parts := strings.SplitN(keyValue, ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid format, expected key:value")
		}
		key, value := parts[0], parts[1]
		if !isAppendRemovePathAllowed(key) {
			return errors.New("remove not supported for this path")
		}
		arr, err := getJSONPath(m, key)
		if err != nil {
			return err
		}
		list, ok := arr.([]interface{})
		if !ok {
			return errors.New("target is not a list")
		}
		newList := make([]interface{}, 0)
		for _, v := range list {
			if v != value {
				newList = append(newList, v)
			}
		}
		if err := setJSONPath(m, key, newList); err != nil {
			return err
		}
	}

	if err := validateConfigMapWithSchema(m); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := WriteFile(fs.Fs, configFilePath, out, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// --- Helpers ---

func isAppendRemovePathAllowed(path string) bool {
	parts := strings.Split(path, ".")
	last := strings.TrimSpace(parts[len(parts)-1])
	for _, allowed := range ConfigurationAppendRemovePathsList {
		if last == allowed {
			return true
		}
	}
	return false
}

func getJSONPath(m map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	var cur interface{} = m
	for _, p := range parts {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid path: " + path)
		}
		cur, ok = mm[p]
		if !ok {
			return nil, errors.New("path not found: " + path)
		}
	}
	return cur, nil
}

func setJSONPath(m map[string]interface{}, path string, value interface{}) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("invalid path: path is empty")
	}
	parts := strings.Split(path, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return nil
		}
		if _, ok := cur[p]; !ok {
			cur[p] = make(map[string]interface{})
		}
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return errors.New("invalid path: " + path)
		}
		cur = next
	}
	return nil
}

// Validate a config map against the schema (using a temp file)
func validateConfigMapWithSchema(m map[string]interface{}) error {
	fs := afero.Afero{Fs: afero.NewOsFs()}
	tmpFile, err := CreateTempFile(fs.Fs, "", "inbd_config_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file for schema validation: %w", err)
	}
	defer func() {
		if err := RemoveFile(fs.Fs, tmpFile.Name()); err != nil {
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", tmpFile.Name(), err)
		}
	}()
	defer tmpFile.Close()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("failed to encode config for schema validation: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp config for schema validation: %w", err)
	}

	valid, err := IsValidJSON(fs, schemaFilePath, tmpFile.Name())
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("configuration is not valid according to schema")
	}
	return nil
}
