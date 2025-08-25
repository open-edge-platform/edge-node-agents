/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	utils "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

var (
	imageIDPath = "/etc/image-id"
)

// Snapshotter is the concrete implementation of the IUpdater interface
// for the EMT OS.
type Snapshotter struct {
	commandExecutor utils.Executor
	fs              afero.Fs
	stateFilePath   string
}

// NewSnapshotter creates a new EMTSnapshotter.
func NewSnapshotter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) *Snapshotter {
	return &Snapshotter{
		commandExecutor: commandExecutor,
		fs:              afero.NewOsFs(),
		stateFilePath:   utils.StateFilePath,
	}
}

// NewSnapshotterWithConfig creates a new EMTSnapshotter with custom configuration.
// This is primarily for testing purposes.
func NewSnapshotterWithConfig(commandExecutor utils.Executor, fs afero.Fs, stateFilePath string) *Snapshotter {
	return &Snapshotter{
		commandExecutor: commandExecutor,
		fs:              fs,
		stateFilePath:   stateFilePath,
	}
}

// Snapshot creates a snapshot of the system.
func (t *Snapshotter) Snapshot() error {
	log.Println("Take a snapshot.")

	err := utils.ClearStateFile(t.commandExecutor, t.stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to clear dispatcher state file: %w", err)
	}

	buildDate, err := GetImageBuildDate(t.fs)
	if err != nil || buildDate == "" {
		return fmt.Errorf("failed to get image build date: %w", err)
	}
	// Create an instance of EMTState with the desired values
	state := utils.INBDState{
		RestartReason: "sota",
		TiberVersion:  buildDate,
	}
	// Convert the state to JSON
	jsonData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Write the JSON to the state file
	if err := utils.WriteToStateFile(t.fs, t.stateFilePath, string(jsonData)); err != nil {
		return fmt.Errorf("failed to write to state file: %w", err)
	}

	log.Println("Snapshot created successfully.")
	return nil
}

// GetImageBuildDate get the image build date.
func GetImageBuildDate(fs afero.Fs) (string, error) {
	// Open the file
	file, err := utils.Open(fs, imageIDPath)
	if err != nil {
		log.Println("Error opening file:", err)
		return "", err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate through each line
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains IMAGE_BUILD_DATE
		if strings.HasPrefix(line, "IMAGE_BUILD_DATE=") {
			// Extract the value after the first '=' sign
			imageBuildDate := strings.SplitN(line, "=", 2)[1]
			log.Println("IMAGE_BUILD_DATE:", imageBuildDate)
			return imageBuildDate, nil
		}
	}

	log.Println("IMAGE_BUILD_DATE not found.")
	return "", nil
}
