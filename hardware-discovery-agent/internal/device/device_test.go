// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package device_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/device"
)

func Test_GetDeviceInfo(t *testing.T) {
	res, err := device.GetDeviceInfo(testSuccess)
	expected := device.AMTInfo{
		Version:     "16.1.27",
		BuildNumber: "2176",
		Sku:         "16392",
		Features:    "AMT Pro Corporate",
		Uuid:        "1234abcd-ef56-7890-abcd-123456ef7890",
		ControlMode: "activated in client control mode",
		DNSSuffix:   "test.com",
		RAS: device.RASInfo{
			NetworkStatus: "direct",
			RemoteStatus:  "not connected",
			RemoteTrigger: "user initiated",
			MPSHostname:   "",
		},
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoFailed(t *testing.T) {
	res, err := device.GetDeviceInfo(testFailure)
	var expected device.AMTInfo
	assert.Error(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoFailedUnmarshal(t *testing.T) {
	res, err := device.GetDeviceInfo(testFailureUnmarshal)
	var expected device.AMTInfo
	assert.Error(t, err)
	assert.Equal(t, expected, res)
}

func testSuccess(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testFailure(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFailure", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testFailureUnmarshal(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFailureUnmarshal", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestFailure(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}

func TestFailureUnmarshal(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", string("not a json"))
	os.Exit(0)
}
