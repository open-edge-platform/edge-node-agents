/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	"github.com/stretchr/testify/assert"
)

type MockExecutor struct{}

func (m *MockExecutor) Execute(string, ...string) error {
	return nil
}

func TestClean_Success(t *testing.T) {
	// Create a temporary directory with files for testing
	tempDir := t.TempDir()
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	_ = os.WriteFile(file1, []byte("test"), fs.ModePerm)
	_ = os.WriteFile(file2, []byte("test"), fs.ModePerm)

	cleaner := NewCleaner(utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput), tempDir)

	err := cleaner.Clean()
	assert.NoError(t, err)

	// Verify files are deleted
	_, err = os.Stat(file1)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(file2)
	assert.True(t, os.IsNotExist(err))
}

func TestClean_NonExistentPath(t *testing.T) {
	nonExistentPath := "nonexistent/path"

	cleaner := NewCleaner(utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput), nonExistentPath)

	err := cleaner.Clean()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}
