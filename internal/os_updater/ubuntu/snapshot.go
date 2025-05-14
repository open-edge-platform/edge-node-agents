/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"golang.org/x/sys/unix"
)

// Snapshotter is the concrete implementation of the Updater interface
// for the Ubuntu OS.
type Snapshotter struct {
	CommandExecutor utils.Executor
	IsBTRFSFileSystemFunc func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error)
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
		isInstalled, err := isSnapperInstalled(u.CommandExecutor)
		if err != nil {
			return fmt.Errorf("snapper installation check failed: %w", err)
		}
		if !isInstalled {
			return fmt.Errorf("snapper is not installed")
		}

		// Clear the dispatcher state file before writing it.
		if err := utils.ClearStateFile(u.CommandExecutor); err != nil {
			return fmt.Errorf("failed to clear dispatcher state file: %w", err)
		}

		err = ensureSnapperConfig(u.CommandExecutor, "rootConfig")
		if err != nil {
			return fmt.Errorf("failed to ensure snapper config exists: %w", err)
		}

		// Create a snapshot using snapper
		snapshotCmd := []string{
			"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update",
		}
		stdout, stderr, err := u.CommandExecutor.Execute(snapshotCmd)
		if err != nil {
			return fmt.Errorf("error executing command: %s, stderr: %s, err: %w", stdout, stderr, err)
		}
		
		// Log a warning if stderr is non-empty but the command succeeded
		if string(stderr) != "" {
			log.Printf("Warning: snapshot command produced stderr: %s", stderr)
		}
		log.Println("Snapshot created successfully.")
	} else {
		// TODO: Check if ProceedWithRollback flag is set to false.
		// If it set to false, send error and do not proceed.
		log.Println("No snapshot taken as the file system is not BTRFS.")
	}
	return nil
}

// isSnapperInstalled checks if the snapper package is installed on the system.
func isSnapperInstalled(cmdExecutor utils.Executor) (bool, error) {
	findSnapper := []string{
		"which", "snapper",
	}
	stdout, _, err := cmdExecutor.Execute(findSnapper)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Println("Snapper is not installed.")
			return false, nil // snapper is not installed
		}
		return false, fmt.Errorf("snapper is not installed")
	}

	if string(stdout) == "" {
		log.Println("Snapper is not installed.")
		return false, nil
	}

	log.Println("Snapper is installed.")
	return true, nil // snapper is installed
}

func ensureSnapperConfig(cmdExecutor utils.Executor, configName string) error {
    log.Println("Ensuring snapper config exists.")
	checkConfigCmd := []string{"snapper", "-c", configName, "list-configs"}
    stdout, stderr, err := cmdExecutor.Execute(checkConfigCmd)
    if err != nil {
        return fmt.Errorf("failed to check snapper config: %s, stderr: %s, err: %w", stdout, stderr, err)
    }

    if !strings.Contains(string(stdout), configName) {
        createConfigCmd := []string{"snapper", "-c", configName, "create-config", "/"}
        _, stderr, err := cmdExecutor.Execute(createConfigCmd)
        if err != nil {
            return fmt.Errorf("failed to create snapper config: stderr: %s, err: %w", stderr, err)
        }
    }

	log.Println("Snapper config exists or was created successfully.")
    return nil
}
