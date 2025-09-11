/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"fmt"
	"log"
	"os/exec"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// VerifyUpdateAfterReboot verifies the update after a reboot.
// It checks if the state file exists and compares the previous and current versions.
// If the versions are different, it commits the update; otherwise, it reverts to the previous image.
func VerifyUpdateAfterReboot(fs afero.Fs, state utils.INBDState) error {
	previousVersion := state.TiberVersion
	log.Printf("Previous image version: %v", previousVersion)

	currentVersion, err := GetImageBuildDate(fs)
	if err != nil {
		return fmt.Errorf("error getting image build date: %w", err)
	}

	// Remove state file before rebooting.
	err = utils.RemoveFile(fs, utils.StateFilePath)
	if err != nil {
		log.Printf("[Warning] Error removing dispatcher state file: %v", err)
	}

	// Compare the versions
	if currentVersion != previousVersion {
		log.Printf("Update Success. Previous image: %v, Current image: %v", previousVersion, currentVersion)
		emtUpdater := NewUpdater(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), &pb.UpdateSystemSoftwareRequest{})
		err = emtUpdater.commitUpdate()
		if err != nil {
			return fmt.Errorf("error committing update: %w", err)
		}

		// Write status to the log file.
		writeUpdateStatus(fs, SUCCESS, "", "")

		log.Println("SUCCESSFUL INSTALL: Overall SOTA update successful.  System has been properly updated.")

		writeGranularLog(fs, SUCCESS, "")
	} else {
		log.Println("Update failed. Reverting to previous image.")
		// Write the status to the log file.
		writeUpdateStatus(fs, FAIL, "", "Update failed. Versions are the same.")
		writeGranularLog(fs, FAIL, FAILURE_REASON_BOOTLOADER)

		log.Println("Rebooting...")
		// Reboot the system without commit.
		emtRebooter := NewRebooter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), &pb.UpdateSystemSoftwareRequest{})
		err = emtRebooter.Reboot()
		if err != nil {
			return fmt.Errorf("error rebooting system: %w", err)
		}
	}

	return nil
}
