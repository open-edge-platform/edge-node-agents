// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"os"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func createTempFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp("", "")
	assert.NoError(t, err)

	err = tmpFile.Close()
	assert.NoError(t, err)

	return tmpFile.Name()
}

func TestOpenNotLink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	f, err := utils.OpenNoLinks(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, f)

	err = f.Close()
	assert.NoError(t, err)
}

func TestReadFileNoLinks(t *testing.T) {
	f, err := os.CreateTemp("", "")
	assert.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())

	data := []byte("test \n 0xBADC0FFE")
	_, err = f.Write(data)
	assert.NoError(t, err)

	read, err := utils.ReadFileNoLinks(f.Name())
	assert.NoError(t, err)
	assert.Equal(t, data, read)
}

func TestReadNonExistingFile(t *testing.T) {
	read, err := utils.ReadFileNoLinks("")
	assert.Error(t, err)
	assert.Nil(t, read)
}

func TestCreateNotLink(t *testing.T) {
	f, err := utils.CreateNoLinks("/tmp/regularFileTest", 0600)
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	err = f.Close()
	assert.NoError(t, err)
}

func TestOpenSymlink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	symlinkPath := filePath + "-symlink"
	err := os.Symlink(filePath, symlinkPath)
	defer os.Remove(symlinkPath)
	assert.NoError(t, err)

	f, err := utils.OpenNoLinks(symlinkPath)
	if assert.Error(t, err) {
		assert.Regexp(t, "too many levels of symbolic links", err.Error())
	}
	assert.Nil(t, f)
}

func TestOpenHardlink(t *testing.T) {
	filePath := createTempFile(t)
	defer os.Remove(filePath)

	hardlinkPath := filePath + "-hardlink"
	err := os.Link(filePath, hardlinkPath)
	defer os.Remove(hardlinkPath)
	assert.NoError(t, err)

	f, err := utils.OpenNoLinks(hardlinkPath)
	if assert.Error(t, err) {
		assert.Regexp(t, hardlinkPath+" is a hardlink", err.Error())
	}
	assert.Nil(t, f)
}
