// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

func createTempFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func TestOpenNotLink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	f, err := utils.OpenNoLinks(filePath)
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

	read, err := utils.ReadFileNoLinks(f.Name())
	require.NoError(t, err)
	require.Equal(t, data, read)
}

func TestReadNonExistingFile(t *testing.T) {
	read, err := utils.ReadFileNoLinks("")
	require.Error(t, err)
	require.Nil(t, read)
}

func TestCreateNotLink(t *testing.T) {
	f, err := utils.CreateNoLinks("/tmp/regularFileTest", 0600)
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

	f, err := utils.OpenNoLinks(symlinkPath)
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

	f, err := utils.OpenNoLinks(hardlinkPath)
	if assert.Error(t, err) {
		require.Regexp(t, hardlinkPath+" is a hardlink", err.Error())
	}
	require.Nil(t, f)
}
