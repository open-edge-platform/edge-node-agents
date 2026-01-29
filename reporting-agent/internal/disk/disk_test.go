// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package disk

import (
	"os"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/testutils"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

func TestGetDiskDataSuccess(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/mock_disks.json")
	require.NoError(t, err)
	testutils.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, testData, nil)

	out, err := GetDiskData(testutils.TestCmdExecutor)
	require.NoError(t, err)

	expected := make([]model.Disk, 0, 3)
	disk1Res := model.Disk{
		Name:   "nvme0n1p1",
		Vendor: "",
		Model:  "",
		Size:   1127219200,
	}
	disk2Res := model.Disk{
		Name:          "sda",
		Vendor:        "DELL    ",
		Model:         "PERC H730P Mini",
		Size:          240057409536,
		ChildrenCount: 8,
	}
	disk3Res := model.Disk{
		Name:   "nvme0n1",
		Vendor: "",
		Model:  "INTEL SSDPEDMD800G4",
		Size:   800166076416,
	}
	expected = append(expected, disk1Res, disk2Res, disk3Res)
	require.Equal(t, expected, out)
}

func TestGetDiskDataLsblkCommandFailure(t *testing.T) {
	testutils.ClearMockOutputs()
	testutils.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, nil, os.ErrPermission)

	_, err := GetDiskData(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from lsblk command")
}

func TestGetDiskDataUnmarshalFailed(t *testing.T) {
	testutils.ClearMockOutputs()
	testutils.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, []byte("not a json"), nil)

	_, err := GetDiskData(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to unmarshal data")
}
