// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"os"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VerifyUpdate_KernelOnlyUpdate_NoINBCLogs(t *testing.T) {
	// Create a temporary metadata file
	metaFile, err := os.CreateTemp("/tmp", "metadata-kernel-test-")
	require.NoError(t, err)
	defer os.Remove(metaFile.Name())
	defer metaFile.Close()

	// Set up metadata
	metadata.MetaPath = metaFile.Name()
	err = metadata.InitMetadata()
	require.NoError(t, err)

	// Set up a kernel update scenario
	err = metadata.SetMetaUpdateInProgress(metadata.OS)
	require.NoError(t, err)

	updateSource := &pb.UpdateSource{
		KernelCommand: "intel_iommu=on",
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	require.NoError(t, err)

	// Create UpdateController
	controller := &UpdateController{}

	// Test VerifyUpdate with non-existent INBC log files (typical for kernel updates)
	status, granularLog, time, err := controller.VerifyUpdate("/non/existent/inbc.log", "/non/existent/granular.log")

	// Assert the results
	assert.NoError(t, err)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, status)
	assert.Equal(t, "Kernel command line parameters updated successfully", granularLog)
	assert.Equal(t, "", time)
}

func Test_VerifyUpdate_KernelOnlyUpdate_EmptyINBCLogs(t *testing.T) {
	// Create a temporary metadata file
	metaFile, err := os.CreateTemp("/tmp", "metadata-kernel-test-")
	require.NoError(t, err)
	defer os.Remove(metaFile.Name())
	defer metaFile.Close()

	// Create empty INBC log files
	inbcLogFile, err := os.CreateTemp("/tmp", "inbc-log-")
	require.NoError(t, err)
	defer os.Remove(inbcLogFile.Name())
	defer inbcLogFile.Close()

	granularLogFile, err := os.CreateTemp("/tmp", "granular-log-")
	require.NoError(t, err)
	defer os.Remove(granularLogFile.Name())
	defer granularLogFile.Close()

	// Set up metadata
	metadata.MetaPath = metaFile.Name()
	err = metadata.InitMetadata()
	require.NoError(t, err)

	// Set up a kernel update scenario
	err = metadata.SetMetaUpdateInProgress(metadata.OS)
	require.NoError(t, err)

	updateSource := &pb.UpdateSource{
		KernelCommand: "intel_iommu=on mitigations=off",
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	require.NoError(t, err)

	// Create UpdateController
	controller := &UpdateController{}

	// Test VerifyUpdate with empty INBC log files (typical for kernel updates)
	status, granularLog, time, err := controller.VerifyUpdate(inbcLogFile.Name(), granularLogFile.Name())

	// Assert the results
	assert.NoError(t, err)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, status)
	assert.Equal(t, "Kernel command line parameters updated successfully", granularLog)
	assert.Equal(t, "", time)
}

func Test_VerifyUpdate_NonKernelUpdate_ShouldFail(t *testing.T) {
	// Create a temporary metadata file
	metaFile, err := os.CreateTemp("/tmp", "metadata-non-kernel-test-")
	require.NoError(t, err)
	defer os.Remove(metaFile.Name())
	defer metaFile.Close()

	// Set up metadata
	metadata.MetaPath = metaFile.Name()
	err = metadata.InitMetadata()
	require.NoError(t, err)

	// Set up a non-kernel update scenario (INBM update)
	err = metadata.SetMetaUpdateInProgress(metadata.INBM)
	require.NoError(t, err)

	// Create UpdateController
	controller := &UpdateController{}

	// Test VerifyUpdate with non-existent INBC log files for non-kernel updates
	status, granularLog, time, err := controller.VerifyUpdate("/non/existent/inbc.log", "/non/existent/granular.log")

	// Assert the results - should fail for non-kernel updates
	assert.Error(t, err)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, "", granularLog)
	assert.Equal(t, "", time)
	assert.Contains(t, err.Error(), "reading INBC logs failed")
}

func Test_VerifyUpdate_KernelUpdateWithoutKernelCommand_ShouldFail(t *testing.T) {
	// Create a temporary metadata file
	metaFile, err := os.CreateTemp("/tmp", "metadata-kernel-no-cmd-test-")
	require.NoError(t, err)
	defer os.Remove(metaFile.Name())
	defer metaFile.Close()

	// Set up metadata
	metadata.MetaPath = metaFile.Name()
	err = metadata.InitMetadata()
	require.NoError(t, err)

	// Set up a kernel update scenario but without kernel command
	err = metadata.SetMetaUpdateInProgress(metadata.OS)
	require.NoError(t, err)

	updateSource := &pb.UpdateSource{
		KernelCommand: "", // Empty kernel command
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	require.NoError(t, err)

	// Create UpdateController
	controller := &UpdateController{}

	// Test VerifyUpdate with non-existent INBC log files
	status, granularLog, time, err := controller.VerifyUpdate("/non/existent/inbc.log", "/non/existent/granular.log")

	// Assert the results - should fail since no kernel command was provided
	assert.Error(t, err)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, "", granularLog)
	assert.Equal(t, "", time)
	assert.Contains(t, err.Error(), "reading INBC logs failed")
}
