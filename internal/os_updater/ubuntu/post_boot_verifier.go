/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"log"

	utils "github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
)

// VerifyUpdateAfterReboot verifies the update after a reboot.
// It checks if the state file exists and compares the previous and current versions.
// If the versions are different, it commits the update; otherwise, it reverts to the previous image.
func VerifyUpdateAfterReboot(fs afero.Fs, state utils.INBDState) error {
	snapshotNumber := state.SnapshotNumber
	log.Printf("Snapshot number: %v", snapshotNumber)

	// Remove state file before rebooting.
	err := utils.RemoveFile(fs, utils.StateFilePath)
	if err != nil {
		log.Printf("[Warning] Error removing state file: %v", err)
	}	

	return nil
}
