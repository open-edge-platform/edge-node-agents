/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ossource provides functionality to update the OS source.
package ossource

import (
	"fmt"
	"os"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
)

// Update updates the Ubuntu apt sources list with the new contents.
func Update(newContents []string) error {
	// Might need this up to be an incoming parameter for testing
	fs := afero.NewOsFs()

	// Backup the original sources list
	if err := utils.CopyFile(fs, ubuntuAptSourcesList, ubuntuAptSourcesListBackup); err != nil {
		return fmt.Errorf("failed to backup sources list: %w", err)
	}

	file, err := utils.OpenFile(fs, ubuntuAptSourcesList, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	for _, line := range newContents {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}
