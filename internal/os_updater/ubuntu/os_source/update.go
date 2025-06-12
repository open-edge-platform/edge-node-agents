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

// Manager is an interface that defines the methods to add an application source.
type Manager interface {
	Update(sourceListFileName string, sources []string, gpgKeyURI string, gpgKeyName string) error
}

// Updater is a struct that implements the Manager interface.
type Updater struct {
	fs           afero.Fs
	openFileFunc func(afero.Fs, string, int, os.FileMode) (afero.File, error)
	copyFileFunc func(afero.Fs, string, string) error
}

// NewUpdater creates a new Updater.
func NewUpdater() *Updater {
	return &Updater{
		fs:           afero.NewOsFs(),
		openFileFunc: utils.OpenFile,
		copyFileFunc: utils.CopyFile,
	}
}

// Update updates the Ubuntu apt sources list with the new contents.
func (u *Updater) Update(newContents []string, aptSourcesListPath string) error {

	// Backup the original sources list
	backupPath := aptSourcesListPath + ".bak"
	if err := u.copyFileFunc(u.fs, aptSourcesListPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup sources list: %w", err)
	}

	file, err := u.openFileFunc(u.fs, aptSourcesListPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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
