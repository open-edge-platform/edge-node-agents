/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"log"
	"os/exec"

	utils "github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
)

// VerifyUpdateAfterReboot verifies the update after a reboot.
// It checks if the state file exists and compares the previous and current versions.
// If the versions are different, it commits the update; otherwise, it reverts to the previous image.
func VerifyUpdateAfterReboot(fs afero.Fs, state utils.INBDState) error {
	snapshotNumber := state.SnapshotNumber
	log.Printf("Snapshot number: %v", snapshotNumber)

	cmdExecutor := utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)

	// Network check
	if !CheckNetworkConnection(cmdExecutor) {
		log.Println("No network connection detected.  Reverting to previous snapshot.")
		if err := UndoChange(cmdExecutor, snapshotNumber); err != nil {
			log.Printf("Failed to revert to previous snapshot: %v", err)
			return err
		}
		if err := DeleteSnapshot(cmdExecutor, snapshotNumber); err != nil {
			log.Printf("Failed to delete snapshot: %v", err)
			return err
		}
		log.Println("Reverted to previous snapshot and deleted it.")
		return nil

	}
	log.Println("Network connection detected.  Proceeding with update verification.")

	// Remove state file after checks
	err := utils.RemoveFile(fs, utils.StateFilePath)
	if err != nil {
		log.Printf("[Warning] Error removing state file: %v", err)
	}	

	return nil
}
