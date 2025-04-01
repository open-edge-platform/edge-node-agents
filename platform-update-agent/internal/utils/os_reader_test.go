// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealFileReader_ReadFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "testfile.txt")

	content := []byte("Hello, Intel Edge!")

	err := os.WriteFile(tempFilePath, content, 0644)
	require.NoError(t, err, "Failed to write to temp file")

	reader := &RealFileReader{}

	readContent, err := reader.ReadFile(tempFilePath)
	require.NoError(t, err, "ReadFile returned an unexpected error")

	assert.Equal(t, content, readContent, "Content mismatch")
}

func TestRealFileReader_ReadFile_NonExistent(t *testing.T) {
	reader := &RealFileReader{}

	nonExistentFilePath := "/path/to/nonexistent/file.txt"

	readContent, err := reader.ReadFile(nonExistentFilePath)

	assert.Error(t, err, "Expected an error when reading a non-existent file")

	if pathErr, ok := err.(*os.PathError); ok {
		assert.Equal(t, nonExistentFilePath, pathErr.Path, "Error path mismatch")
		assert.True(t, os.IsNotExist(err), "Expected a file not exist error")
	} else {
		t.Errorf("Expected a *os.PathError, but got: %T", err)
	}

	assert.Nil(t, readContent, "Expected no content to be read")
}

func TestRealFileReader_ReadFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	tempFilePath := filepath.Join(tempDir, "empty.txt")

	file, err := os.Create(tempFilePath)
	require.NoError(t, err, "Failed to create empty file")
	file.Close()

	reader := &RealFileReader{}

	readContent, err := reader.ReadFile(tempFilePath)
	require.NoError(t, err, "ReadFile returned an unexpected error")

	assert.Empty(t, readContent, "Expected empty content")
}
