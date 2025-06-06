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

// TestReadFileTrimmedSuccess verifies that ReadFileTrimmed returns trimmed content for a valid file.
func TestReadFileTrimmedSuccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	content := "   hello world  \n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0640), "Should write test file")

	result, err := ReadFileTrimmed(file)
	require.NoError(t, err, "ReadFileTrimmed should not return error for valid file")
	require.Equal(t, "hello world", result, "ReadFileTrimmed should trim whitespace")
}

// TestReadFileTrimmedEmptyFile checks that ReadFileTrimmed returns an empty string for an empty file.
func TestReadFileTrimmedEmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(file, []byte("   \n\t "), 0640), "Should write empty test file")

	result, err := ReadFileTrimmed(file)
	require.NoError(t, err, "ReadFileTrimmed should not return error for empty file")
	require.Empty(t, result, "ReadFileTrimmed should return empty string for whitespace-only file")
}

// TestReadFileTrimmedFileNotExist checks that ReadFileTrimmed returns an error for a non-existent file.
func TestReadFileTrimmedFileNotExist(t *testing.T) {
	_, err := ReadFileTrimmed("/non/existing/file/path")
	require.Error(t, err, "ReadFileTrimmed should return error for non-existent file")
}

// TestReadFileTrimmedNoPermission checks that ReadFileTrimmed returns an error for a file without read permission.
func TestReadFileTrimmedNoPermission(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "noperm.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0640), "Should write file with 0640 permissions")
	require.NoError(t, os.Chmod(file, 0000), "Should chmod file to 0000")
	defer func() { require.NoError(t, os.Chmod(file, 0640), "Should chmod file back to 0640") }() // Clean up permissions for temp dir deletion

	_, err := ReadFileTrimmed(file)
	require.Error(t, err, "ReadFileTrimmed should return error for file without read permission")
}

func TestOpenNotLink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	f, err := OpenNoLinks(filePath)
	require.NoError(t, err)
	require.NotNil(t, f)

	err = f.Close()
	require.NoError(t, err)
}

func TestReadFileNoLinks(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	defer f.Close()

	data := []byte("test \n 0xBADC0FFE")
	_, err = f.Write(data)
	require.NoError(t, err)

	read, err := ReadFileNoLinks(f.Name())
	require.NoError(t, err)
	require.Equal(t, data, read)
}

func TestReadNonExistingFile(t *testing.T) {
	read, err := ReadFileNoLinks("")
	require.Error(t, err)
	require.Nil(t, read)
}

func TestCreateNotLink(t *testing.T) {
	f, err := CreateNoLinks("/tmp/regularFileTest", 0600)
	require.NoError(t, err)
	defer os.Remove(f.Name())

	err = f.Close()
	require.NoError(t, err)
}

func TestOpenSymlink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	symlinkPath := filePath + "-symlink"
	err := os.Symlink(filePath, symlinkPath)
	defer os.Remove(symlinkPath)
	require.NoError(t, err)

	f, err := OpenNoLinks(symlinkPath)
	if assert.Error(t, err) {
		require.Regexp(t, "too many levels of symbolic links", err.Error())
	}
	require.Nil(t, f)
}

func TestOpenHardlink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	hardlinkPath := filePath + "-hardlink"
	err := os.Link(filePath, hardlinkPath)
	defer os.Remove(hardlinkPath)
	require.NoError(t, err)

	f, err := OpenNoLinks(hardlinkPath)
	if assert.Error(t, err) {
		require.Regexp(t, hardlinkPath+" is a hardlink", err.Error())
	}
	require.Nil(t, f)
}

func createTempFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
