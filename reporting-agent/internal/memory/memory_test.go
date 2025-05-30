// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

func TestGetMemoryDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_real")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(32768), mem.Summary.TotalSizeMB)
	require.Equal(t, "DIMM", mem.Summary.CommonFormFactor)
	require.Equal(t, "DRAM", mem.Summary.CommonType)
	require.Len(t, mem.Devices, 2)
	for _, dev := range mem.Devices {
		require.NotEmpty(t, dev.FormFactor)
		require.NotEmpty(t, dev.Size)
		require.NotEmpty(t, dev.Type)
		require.NotEmpty(t, dev.Speed)
		require.NotEmpty(t, dev.Manufacturer)
	}
}

func TestGetMemoryDataCommandFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, nil, os.ErrPermission)

	_, err := GetMemoryData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from dmidecode command")
}

func TestGetMemoryDataIgnoresUnrelatedLines(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_dummy_lines")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(4*1024), mem.Summary.TotalSizeMB)
	require.Equal(t, "DDR3", mem.Summary.CommonType)
	require.Equal(t, "DIMM", mem.Summary.CommonFormFactor)

	require.Len(t, mem.Devices, 1)
	require.Equal(t, "DIMM", mem.Devices[0].FormFactor)
	require.Equal(t, "4 GB", mem.Devices[0].Size)
	require.Equal(t, "DDR3", mem.Devices[0].Type)
	require.Equal(t, "1600 MT/s", mem.Devices[0].Speed)
	require.Equal(t, "Kingston", mem.Devices[0].Manufacturer)
}

func TestGetMemoryDataHandlesNoModuleInstalled(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_no_modules_installed")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(0), mem.Summary.TotalSizeMB)
	require.Empty(t, mem.Summary.CommonType)
	require.Empty(t, mem.Summary.CommonFormFactor)
	require.Empty(t, mem.Devices)
}

func TestGetMemoryDataHandlesUnknownSize(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_unknown_size")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(0), mem.Summary.TotalSizeMB)
	require.Empty(t, mem.Summary.CommonType)
	require.Empty(t, mem.Summary.CommonFormFactor)
	require.Empty(t, mem.Devices)
}

func TestGetMemoryDataDifferentTypes(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_memory_different_types")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(16*1024), mem.Summary.TotalSizeMB)
	require.Empty(t, mem.Summary.CommonType)
	require.Equal(t, "DIMM", mem.Summary.CommonFormFactor)
	require.Len(t, mem.Devices, 2)
}

func TestGetMemoryDataDifferentFormFactors(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/dmidecode_different_form_factors")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, testData, nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(16*1024), mem.Summary.TotalSizeMB)
	require.Equal(t, "DDR4", mem.Summary.CommonType)
	require.Empty(t, mem.Summary.CommonFormFactor)
	require.Len(t, mem.Devices, 2)
}

func TestGetMemoryDataEmptyOutput(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "--type", "memory"}, []byte(""), nil)

	mem, err := GetMemoryData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, model.Memory{Devices: []model.MemoryDevice{}}, mem)
}
