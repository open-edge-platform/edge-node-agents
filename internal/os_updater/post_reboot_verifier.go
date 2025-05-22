/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	"fmt"
	"log"
	"os"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/intel/intel-inb-manageability/internal/os_updater/emt"
	"github.com/intel/intel-inb-manageability/internal/os_updater/ubuntu"
	"github.com/spf13/afero"
)

// VerifyUpdateAfterReboot verifies the update after a reboot.
// It checks if the dispatcher state file exists and compares the previous and current versions.
// If the versions are different, it commits the update; otherwise, it reverts to the previous image.
func VerifyUpdateAfterReboot(fs afero.Fs) error {
	// Check if state file exist.
	fileInfo, err := os.Stat(utils.StateFilePath)
	if err == nil {
		if fileInfo.Size() == 0 {
			log.Println("State file is empty. Skip post update verification.")
			return nil
		}

		log.Println("Perform post update verification.")
		osType, err := DetectOS()
		if err != nil {
			return fmt.Errorf("error detecting OS: %w", err)
		}

		state, err := utils.ReadStateFile(fs, utils.StateFilePath)
		if err != nil {
			return fmt.Errorf("error reading state file: %w", err)
		}

		if osType == "EMT" {
			err := emt.VerifyUpdateAfterReboot(fs, state)
			if err != nil {
				return err
			}
		} else if osType == "Ubuntu" {
			err := ubuntu.VerifyUpdateAfterReboot(fs, state)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Unsupported OS type: %s", osType)
		}
		log.Println("Post update verification completed.")
	} else {
		log.Println("No dispatcher state file. Skip post update verification.")
	}

	return nil
}
