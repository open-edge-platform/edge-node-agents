// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ossource

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const mockAptSourcesList = "/tmp/apt/sources.list"

func TestUpdate_Success(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

	// Mock the original sources list
	originalContent := "deb http://archive.ubuntu.com/ubuntu focal main restricted"
	err := afero.WriteFile(fs, mockAptSourcesList, []byte(originalContent), 0644)
	assert.NoError(t, err)

	// Mock the new contents
	newContents := []string{
		"deb http://archive.ubuntu.com/ubuntu focal main restricted",
		"deb http://security.ubuntu.com/ubuntu focal-security main restricted",
	}

	// Create an Updater instance with default behavior
	updater := &Updater{
		fs: fs,
		openFileFunc: func(fs afero.Fs, name string, flag int, perm os.FileMode) (afero.File, error) {
			return fs.OpenFile(name, flag, perm)
		},
		copyFileFunc: func(fs afero.Fs, srcPath, destPath string) error {
			return afero.WriteFile(fs, destPath, []byte("backup content"), 0644)
		},
	}

	// Call the Update function
	err = updater.Update(newContents, mockAptSourcesList)
	assert.NoError(t, err)

	// Verify the updated sources list
	content, err := afero.ReadFile(fs, mockAptSourcesList)
	assert.NoError(t, err)
	expectedContent := "deb http://archive.ubuntu.com/ubuntu focal main restricted\n" +
		"deb http://security.ubuntu.com/ubuntu focal-security main restricted\n"
	assert.Equal(t, expectedContent, string(content))
}

func TestUpdate_BackupFailure(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

	// Mock the new contents
	newContents := []string{
		"deb http://archive.ubuntu.com/ubuntu focal main restricted",
	}

	// Create an Updater instance with a failing copyFileFunc
	updater := &Updater{
		fs: fs,
		openFileFunc: func(fs afero.Fs, name string, flag int, perm os.FileMode) (afero.File, error) {
			return fs.OpenFile(name, flag, perm)
		},
		copyFileFunc: func(fs afero.Fs, srcPath, destPath string) error {
			return fmt.Errorf("mock backup failure")
		},
	}

	// Call the Update function
	err := updater.Update(newContents, mockAptSourcesList)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup sources list")
}

func TestUpdate_WriteFailure(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

	// Mock the original sources list
	originalContent := "deb http://archive.ubuntu.com/ubuntu focal main restricted"
	err := afero.WriteFile(fs, mockAptSourcesList, []byte(originalContent), 0644)
	assert.NoError(t, err)

	// Mock the new contents
	newContents := []string{
		"deb http://archive.ubuntu.com/ubuntu focal main restricted",
	}

	// Create an Updater instance with a failing openFileFunc
	updater := &Updater{
		fs: fs,
		openFileFunc: func(fs afero.Fs, name string, flag int, perm os.FileMode) (afero.File, error) {
			return nil, fmt.Errorf("mock write failure")
		},
		copyFileFunc: func(fs afero.Fs, srcPath, destPath string) error {
			return nil
		},
	}

	// Call the Update function
	err = updater.Update(newContents, mockAptSourcesList)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}
