/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package emt

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
)

// SnapshotMockExecutor is a specific mock implementation for snapshot tests
type SnapshotMockExecutor struct {
	shouldFail bool
	errorMsg   string
}

func (m *SnapshotMockExecutor) Execute(args []string) ([]byte, []byte, error) {
	if m.shouldFail {
		return []byte(""), []byte(m.errorMsg), errors.New(m.errorMsg)
	}
	return []byte(""), []byte(""), nil
}

func TestNewSnapshotter(t *testing.T) {
	mockExecutor := &SnapshotMockExecutor{}
	request := &pb.UpdateSystemSoftwareRequest{
		Url: "http://example.com/update.tar",
	}

	snapshotter := NewSnapshotter(mockExecutor, request)

	assert.NotNil(t, snapshotter)
	assert.Equal(t, mockExecutor, snapshotter.commandExecutor)
	assert.NotNil(t, snapshotter.fs)
	assert.Equal(t, utils.StateFilePath, snapshotter.stateFilePath)
}

func TestSnapshotter_Snapshot_Success(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with valid content
	imageIDContent := `IMAGE_BUILD_DATE=2025-01-15
OTHER_FIELD=some_value
ANOTHER_FIELD=another_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Create mock executor that succeeds
	mockExecutor := &SnapshotMockExecutor{shouldFail: false}

	// Create snapshotter with memory filesystem and custom state file path
	stateFilePath := "/tmp/test_state_file"
	snapshotter := NewSnapshotterWithConfig(mockExecutor, fs, stateFilePath)

	// Execute snapshot
	err = snapshotter.Snapshot()
	assert.NoError(t, err)

	// Verify the state file was created with correct content
	stateContent, err := afero.ReadFile(fs, stateFilePath)
	assert.NoError(t, err)

	var state utils.INBDState
	err = json.Unmarshal(stateContent, &state)
	assert.NoError(t, err)
	assert.Equal(t, "sota", state.RestartReason)
	assert.Equal(t, "2025-01-15", state.TiberVersion)
}

func TestSnapshotter_Snapshot_ClearStateFileError(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create mock executor that fails
	mockExecutor := &SnapshotMockExecutor{
		shouldFail: true,
		errorMsg:   "command failed",
	}

	// Create snapshotter with memory filesystem and custom state file path
	stateFilePath := "/tmp/test_state_file"
	snapshotter := NewSnapshotterWithConfig(mockExecutor, fs, stateFilePath)

	// Execute snapshot
	err := snapshotter.Snapshot()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear dispatcher state file")
}

func TestSnapshotter_Snapshot_GetImageBuildDateError(t *testing.T) {
	// Create a memory filesystem without the image-id file
	fs := afero.NewMemMapFs()

	// Create mock executor that succeeds (but image-id file doesn't exist)
	mockExecutor := &SnapshotMockExecutor{shouldFail: false}

	// Create snapshotter with memory filesystem and custom state file path
	stateFilePath := "/tmp/test_state_file"
	snapshotter := NewSnapshotterWithConfig(mockExecutor, fs, stateFilePath)

	// Execute snapshot
	err := snapshotter.Snapshot()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get image build date")
}

func TestSnapshotter_Snapshot_EmptyBuildDate(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file without IMAGE_BUILD_DATE
	imageIDContent := `OTHER_FIELD=some_value
ANOTHER_FIELD=another_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Create mock executor that succeeds
	mockExecutor := &SnapshotMockExecutor{shouldFail: false}

	// Create snapshotter with memory filesystem and custom state file path
	stateFilePath := "/tmp/test_state_file"
	snapshotter := NewSnapshotterWithConfig(mockExecutor, fs, stateFilePath)

	// Execute snapshot
	err = snapshotter.Snapshot()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get image build date")
}

func TestGetImageBuildDate_Success(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with valid content
	imageIDContent := `# This is a comment
IMAGE_BUILD_DATE=2025-01-15T10:30:00Z
OTHER_FIELD=some_value
ANOTHER_FIELD=another_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Equal(t, "2025-01-15T10:30:00Z", buildDate)
}

func TestGetImageBuildDate_FileNotFound(t *testing.T) {
	// Create a memory filesystem without the image-id file
	fs := afero.NewMemMapFs()

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.Error(t, err)
	assert.Empty(t, buildDate)
}

func TestGetImageBuildDate_NoImageBuildDate(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file without IMAGE_BUILD_DATE
	imageIDContent := `# This is a comment
OTHER_FIELD=some_value
ANOTHER_FIELD=another_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Empty(t, buildDate)
}

func TestGetImageBuildDate_ImageBuildDateAtBeginning(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with IMAGE_BUILD_DATE at the beginning
	imageIDContent := `IMAGE_BUILD_DATE=2025-01-15
OTHER_FIELD=some_value
ANOTHER_FIELD=another_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Equal(t, "2025-01-15", buildDate)
}

func TestGetImageBuildDate_ImageBuildDateAtEnd(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with IMAGE_BUILD_DATE at the end
	imageIDContent := `OTHER_FIELD=some_value
ANOTHER_FIELD=another_value
IMAGE_BUILD_DATE=2025-12-31`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Equal(t, "2025-12-31", buildDate)
}

func TestGetImageBuildDate_MultipleImageBuildDates(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with multiple IMAGE_BUILD_DATE entries (should return first one)
	imageIDContent := `IMAGE_BUILD_DATE=2025-01-15
OTHER_FIELD=some_value
IMAGE_BUILD_DATE=2025-12-31`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Equal(t, "2025-01-15", buildDate) // Should return the first occurrence
}

func TestGetImageBuildDate_EmptyFile(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create an empty image-id file
	err := afero.WriteFile(fs, "/etc/image-id", []byte(""), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Empty(t, buildDate)
}

func TestGetImageBuildDate_OnlyWhitespace(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with only whitespace
	imageIDContent := `   
	
	
	`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Empty(t, buildDate)
}

func TestGetImageBuildDate_ImageBuildDateWithEquals(t *testing.T) {
	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Create the image-id file with IMAGE_BUILD_DATE containing equals signs in the value
	imageIDContent := `IMAGE_BUILD_DATE=version=2025-01-15=final
OTHER_FIELD=some_value`
	err := afero.WriteFile(fs, "/etc/image-id", []byte(imageIDContent), 0644)
	assert.NoError(t, err)

	// Call GetImageBuildDate
	buildDate, err := GetImageBuildDate(fs)
	assert.NoError(t, err)
	assert.Equal(t, "version=2025-01-15=final", buildDate) // Should handle multiple equals signs
}
