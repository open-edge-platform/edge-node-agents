/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// Snapshotter is the concrete implementation of the Updater interface
// for the Ubuntu OS.
type Snapshotter struct {
	CommandExecutor         common.Executor
	IsBTRFSFileSystemFunc   func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error)
	IsSnapperInstalledFunc  func(cmdExecutor common.Executor) (bool, error)
	EnsureSnapperConfigFunc func(cmdExecutor common.Executor, configName string) error
	ClearStateFileFunc      func(cmdExecutor common.Executor, stateFilePath string) error
	WriteToStateFileFunc    func(fs afero.Fs, stateFilePath string, content string) error
	Fs                      afero.Fs
}

// Snapshot method for Ubuntu
func (u *Snapshotter) Snapshot() error {
	// Check if the file system is BTRFS
	isBtrfs, err := u.IsBTRFSFileSystemFunc("/", unix.Statfs)
	if err != nil {
		return fmt.Errorf("failed to check if file system is BTRFS: %w", err)
	}
	if isBtrfs {
		log.Println("OS is Ubuntu and FileSystem is BTRFS.  Take a snapshot.")

		// Check if snapper is installed
		isInstalled, err := u.IsSnapperInstalledFunc(u.CommandExecutor)
		if err != nil {
			return fmt.Errorf("snapper installation check failed: %w", err)
		}
		if !isInstalled {
			return fmt.Errorf("snapper is not installed")
		}

		if err := u.ClearStateFileFunc(u.CommandExecutor, utils.StateFilePath); err != nil {
			return fmt.Errorf("failed to clear dispatcher state file: %w", err)
		}

		err = u.EnsureSnapperConfigFunc(u.CommandExecutor, "rootConfig")
		if err != nil {
			return fmt.Errorf("failed to ensure snapper config exists: %w", err)
		}

		// Create a snapshot using snapper
		snapshotCmd := []string{
			common.SnapperCmd, "-c", "rootConfig", "create", "-p", "--description", "sota_update",
		}
		stdout, stderr, err := u.CommandExecutor.Execute(snapshotCmd)
		if err != nil {
			return fmt.Errorf("error executing command: %s, stderr: %s, err: %w", stdout, stderr, err)
		}

		// Log a warning if stderr is non-empty but the command succeeded
		if string(stderr) != "" {
			log.Printf("Warning: snapshot command produced stderr: %s", stderr)
		}
		log.Println("Snapshot created successfully. SnapshotID: ", string(stdout))

		// Ensure stdout is not blank and is an integer
		snapshotID := strings.TrimSpace(string(stdout))
		if snapshotID == "" {
			return fmt.Errorf("snapshot ID is blank")
		}
		snapshotNumber, err := strconv.Atoi(snapshotID)
		if err != nil {
			return fmt.Errorf("snapshot ID is not a valid integer: %s", snapshotID)
		}

		state := utils.INBDState{
			RestartReason:  "sota",
			SnapshotNumber: snapshotNumber,
		}

		stateJSON, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to serialize state to JSON: %w", err)
		}
		log.Printf("State JSON: %s", string(stateJSON))
		err = u.WriteToStateFileFunc(u.Fs, utils.StateFilePath, string(stateJSON))
		if err != nil {
			return fmt.Errorf("failed to write to state file: %w", err)
		}
		log.Printf("Snapshot ID: %s", snapshotID)
	} else {
		// TODO: Check if ProceedWithRollback flag is set to false.
		// If it set to false, send error and do not proceed.
		log.Println("No snapshot taken as the file system is not BTRFS.")
	}
	return nil
}

// UndoChange reverts the changes made after the snapshot version.
func UndoChange(cmdExecutor common.Executor, snapshotNumber int) error {
	log.Println("Undoing changes made after the snapshot version.")

	if snapshotNumber == 0 {
		log.Println("Update System Software rollback skipped.  Snapshot number is 0.")
		return nil
	}

	undoChangeRange := strconv.Itoa(snapshotNumber) + "..0"
	revertCmd := []string{
		common.SnapperCmd, "-c", "rootConfig", "undochange", undoChangeRange,
	}
	stdout, stderr, err := cmdExecutor.Execute(revertCmd)
	if err != nil {
		return fmt.Errorf("error executing command: %s, stderr: %s, err: %w", stdout, stderr, err)
	}

	if string(stderr) != "" {
		log.Printf("Warning: undochange command produced stderr: %s", stderr)
	}
	log.Println("UndoChange completed successfully.")
	return nil
}

// DeleteSnapshot deletes a snapshot version
func DeleteSnapshot(cmdExecutor common.Executor, snapshotNumber int) error {
	log.Println("Deleting the snapshot version.")

	if snapshotNumber == 0 {
		log.Println("Snapshot number is 0 (dummy snapshot); no need to delete.")
		return nil
	}

	deleteCmd := []string{
		common.SnapperCmd, "-c", "rootConfig", "delete", strconv.Itoa(snapshotNumber),
	}
	stdout, stderr, err := cmdExecutor.Execute(deleteCmd)
	if err != nil {
		return fmt.Errorf("error executing command: %s, stderr: %s, err: %w", stdout, stderr, err)
	}

	if string(stderr) != "" {
		log.Printf("Warning: delete snapshot command produced stderr: %s", stderr)
	}
	log.Println("DeleteSnapshot completed successfully.")
	return nil
}

// IsSnapperInstalled checks if the snapper package is installed on the system.
func IsSnapperInstalled(cmdExecutor common.Executor) (bool, error) {
	checkSnapperCmd := []string{common.SnapperCmd, "--version"}
	stdout, _, err := cmdExecutor.Execute(checkSnapperCmd)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Println("Snapper is not installed.")
			return false, nil // snapper is not installed
		}
		return false, fmt.Errorf("snapper is not installed")
	}

	if strings.TrimSpace(string(stdout)) == "" {
		log.Println("Snapper is not installed.")
		return false, nil
	}

	log.Println("Snapper is installed.")
	return true, nil // snapper is installed
}

// EnsureSnapperConfig checks if the snapper config exists and creates it if not.
func EnsureSnapperConfig(cmdExecutor common.Executor, configName string) error {
	log.Println("Ensuring snapper config exists.")
	checkConfigCmd := []string{common.SnapperCmd, "-c", configName, "list-configs"}
	stdout, stderr, err := cmdExecutor.Execute(checkConfigCmd)
	if err != nil {
		return fmt.Errorf("failed to check snapper config: %s, stderr: %s, err: %w", stdout, stderr, err)
	}

	if !strings.Contains(string(stdout), configName) {
		createConfigCmd := []string{common.SnapperCmd, "-c", configName, "create-config", "/"}
		_, stderr, err := cmdExecutor.Execute(createConfigCmd)
		if err != nil {
			return fmt.Errorf("failed to create snapper config: stderr: %s, err: %w", stderr, err)
		}
	}

	log.Println("Snapper config exists or was created successfully.")
	return nil
}
