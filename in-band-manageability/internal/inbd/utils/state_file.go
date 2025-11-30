/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	"github.com/spf13/afero"
)

// StateFilePath is the path to the inbd state file.
// This file is used to store the state of the inbd process.
const StateFilePath = "/var/intel-manageability/inbd_state"

// ClearStateFile clears the inbd state file by truncating it to zero size.
func ClearStateFile(cmdExecutor common.Executor, stateFilePath string) error {
	log.Println("Clearing inbd state file.")

	// Clear the inbd state file before writing it.
	// We use truncate rather than remove here as some OSes like EMT require files that need to persist
	// between reboots to not be removed.
	stateFileTruncateCommand := []string{
		common.TruncateCmd, "-s", "0", stateFilePath,
	}

	if _, _, err := cmdExecutor.Execute(stateFileTruncateCommand); err != nil {
		return fmt.Errorf("failed to truncate inbd state file with command(%v)- %w", stateFileTruncateCommand, err)
	}
	return nil
}

// INBDState represents the JSON structure
type INBDState struct {
	RestartReason  string `json:"restart_reason"`
	SnapshotNumber int    `json:"snapshot_number"`
	TiberVersion   string `json:"tiber-version"`
	PackageList    string `json:"package_list,omitempty"`
}

// WriteToStateFile writes the content to the state file.
func WriteToStateFile(fs afero.Fs, filePath string, content string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", dir, err)
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Open the file for writing
	file, err := OpenFile(fs, filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// Write the content to the file
	_, err = file.WriteString(content)
	if err != nil {
		log.Println("Error writing file:", err)
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

// ReadStateFile reads the content from the state file.
// It returns the image version.
func ReadStateFile(fs afero.Fs, filePath string) (INBDState, error) {

	file, err := Open(fs, filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return INBDState{}, err
	}
	defer file.Close()

	// Read the file content
	fileContent, err := afero.ReadFile(fs, filePath)
	if err != nil {
		log.Println("Error reading file:", err)
		return INBDState{}, err
	}

	// Parse the JSON content
	var stateJSON INBDState
	err = json.Unmarshal(fileContent, &stateJSON)
	if err != nil {
		log.Println("Error parsing JSON:", err)
		return INBDState{}, fmt.Errorf("error parsing JSON: %w", err)
	}
	return stateJSON, nil
}
