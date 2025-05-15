// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package disk_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/disk"
)

func Test_GetDisks(t *testing.T) {
	out, err := disk.GetDiskList(testCmdExecutorSuccessLSBLK)
	expected := []*disk.Disk{}
	disk1Res := &disk.Disk{
		SerialNum: "unknown",
		Name:      "nvme0n1p1",
		Vendor:    "unknown",
		Model:     "unknown",
		Size:      1127219200,
		Wwid:      "unknown",
	}
	disk2Res := &disk.Disk{
		SerialNum: "002bb496324e7da81d0018d730708741",
		Name:      "sda",
		Vendor:    "DELL    ",
		Model:     "PERC H730P Mini",
		Size:      399431958528,
		Wwid:      "0x5000c5008e0b3b1d",
	}
	disk3Res := &disk.Disk{
		SerialNum: "CVFT521000J6800CGN",
		Name:      "nvme0n1",
		Vendor:    "unknown",
		Model:     "INTEL SSDPEDMD800G4",
		Size:      800166076416,
		Wwid:      "eui.01000000010000005cd2e43cf16e5451",
	}
	expected = append(expected, disk1Res, disk2Res, disk3Res)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetDisksUnmarshalFailed(t *testing.T) {
	out, err := disk.GetDiskList(testCmdExecutorFailedUnmarshal)
	assert.Equal(t, []*disk.Disk{}, out)
	assert.Error(t, err)
}

func Test_GetDisksCommandFailed(t *testing.T) {
	out, err := disk.GetDiskList(testCmdExecutorCommandFailed)
	assert.Equal(t, []*disk.Disk{}, out)
	assert.Error(t, err)
}

func testCmdExecutorSuccessLSBLK(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestDisksListExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailedUnmarshal(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestDisksListExecutionUnmarshalFail", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorCommandFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestDisksListExecutionCommandFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestDisksListExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_disks.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestDisksListExecutionUnmarshalFail(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", string("not a json"))
	os.Exit(0)
}

func TestDisksListExecutionCommandFailed(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}
