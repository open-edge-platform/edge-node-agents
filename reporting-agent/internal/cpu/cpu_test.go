// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cpu

import (
	"os"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/testutils"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

func TestGetCPUDataSuccess(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_real")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, "x86_64", cpu.Architecture)
	require.Equal(t, "GenuineIntel", cpu.Vendor)
	require.Equal(t, "6", cpu.Family)
	require.Equal(t, "13th Gen Intel(R) Core(TM) i9-13950HX", cpu.ModelName)
	require.Equal(t, "183", cpu.Model)
	require.Equal(t, "1", cpu.Stepping)
	require.Equal(t, uint64(12), cpu.SocketCount)
	require.Equal(t, uint64(12), cpu.ThreadCount)
	require.Equal(t, uint64(12), cpu.CoreCount) // 12 * 1
	require.Equal(t, "full", cpu.Virtualization)
	require.Equal(t, "VMware", cpu.Hypervisor)
}

func TestGetCPUDataCommandFailure(t *testing.T) {
	testutils.ClearMockOutputs()
	testutils.SetMockOutput("lscpu", nil, nil, os.ErrPermission)

	_, err := GetCPUData(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from lscpu command")
}

func TestGetCPUDataWithSpacesAndEmptyLines(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_spaces_and_empty_lines")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, "x86_64", cpu.Architecture)
	require.Equal(t, "GenuineIntel", cpu.Vendor)
	require.Equal(t, "6", cpu.Family)
	require.Equal(t, "Intel(R) Xeon(R) CPU", cpu.ModelName)
	require.Equal(t, "42", cpu.Model)
	require.Equal(t, "7", cpu.Stepping)
	require.Equal(t, uint64(2), cpu.SocketCount)
	require.Equal(t, uint64(2), cpu.ThreadCount)
	require.Equal(t, uint64(16), cpu.CoreCount) // 2 * 8
}

func TestGetCPUDataIgnoresUnrelatedLines(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_unrelated_lines")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, "x86_64", cpu.Architecture)
	require.Equal(t, uint64(16), cpu.ThreadCount)
	require.Equal(t, "Intel(R) Xeon(R) Platinum", cpu.ModelName)
}

func TestGetCPUDataHandlesParseUintError(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_not_a_numbers")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, "x86_64", cpu.Architecture)
	require.Equal(t, uint64(0), cpu.ThreadCount)
	require.Equal(t, uint64(0), cpu.SocketCount)
	require.Equal(t, uint64(0), cpu.CoreCount)
	require.Equal(t, "Intel(R) Xeon(R)", cpu.ModelName)
}

func TestGetCPUDataPartialFields(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_partial_data")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, "x86_64", cpu.Architecture)
	require.Equal(t, uint64(4), cpu.ThreadCount)
	require.Empty(t, cpu.ModelName)
	require.Equal(t, uint64(0), cpu.CoreCount)
}

func TestGetCPUDataZeroSocketsOrCores(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_zero_sockets_or_cores")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(0), cpu.SocketCount)
	require.Equal(t, uint64(0), cpu.CoreCount) // 0 * 8 = 0
}

func TestGetCPUDataZeroCoresPerSocket(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lscpu_with_zero_cores_per_socket")
	require.NoError(t, err)
	testutils.SetMockOutput("lscpu", nil, testData, nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cpu.SocketCount)
	require.Equal(t, uint64(0), cpu.CoreCount)
}

func TestGetCPUDataEmptyOutput(t *testing.T) {
	testutils.ClearMockOutputs()
	testutils.SetMockOutput("lscpu", nil, []byte(""), nil)

	cpu, err := GetCPUData(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, model.CPU{}, cpu)
}
