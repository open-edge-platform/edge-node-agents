// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package disk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

func TestGetDiskDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/mock_disks.json")
	require.NoError(t, err)
	testutil.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, testData, nil)

	out, err := GetDiskData(testutil.TestCmdExecutor)
	require.NoError(t, err)

	expected := []model.Disk{}
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
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, nil, os.ErrPermission)

	_, err := GetDiskData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from lsblk command")
}

func TestGetDiskDataUnmarshalFailed(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("lsblk", []string{"-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree"}, []byte("not a json"), nil)

	_, err := GetDiskData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to unmarshal data")
}
