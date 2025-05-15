// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package usb_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/usb"
)

func getExpectedOutput(serial string) []*usb.Usb {
	testOutput := []*usb.Usb{}
	testInterfaces := []*usb.Interface{}
	interfaces := &usb.Interface{Class: "Hub"}
	testInterfaces = append(testInterfaces, interfaces)
	expected := &usb.Usb{
		Class:       "Hub",
		VendorID:    "1d6b",
		ProductID:   "0003",
		Bus:         2,
		Address:     1,
		Description: "Linux Foundation 3.0 root hub",
		Serial:      serial,
		Interfaces:  testInterfaces,
	}
	testOutput = append(testOutput, expected)
	return testOutput
}

func TestGetUsbList(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorSuccessLSUSB)
	testOutput := getExpectedOutput("0000:00:14.0")
	assert.NotNil(t, out)
	assert.Equal(t, testOutput, out)
	require.NoError(t, err)
}

func TestGetUsbListNoSerial(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorSuccessNoSerial)
	testOutput := getExpectedOutput("Not available")
	assert.NotNil(t, out)
	assert.Equal(t, testOutput, out)
	require.NoError(t, err)
}

func TestGetUsbListFirstLSUSBFailed(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorFirstCommandFailed)
	assert.Equal(t, []*usb.Usb{}, out)
	require.Error(t, err)
}

func TestGetUsbListSecondLSUSBFailed(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorSecondCommandFailed)
	assert.Equal(t, []*usb.Usb{}, out)
	require.Error(t, err)
}

func TestGetUsbListGetBusFailed(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorFailedGetBus)
	assert.Equal(t, []*usb.Usb{}, out)
	require.Error(t, err)
}

func TestGetUsbListGetAddressFailed(t *testing.T) {
	out, err := usb.GetUsbList(testCmdExecutorFailedGetAddress)
	assert.Equal(t, []*usb.Usb{}, out)
	require.Error(t, err)
}

func testCmdReceived(args ...string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "-v") {
			return true
		}
	}
	return false
}

func testCmdExecutorSuccessLSUSB(command string, args ...string) *exec.Cmd {
	if !testCmdReceived(args...) {
		cs := []string{"-test.run=TestUsbListBasicExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestUsbListVerboseExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorSuccessNoSerial(command string, args ...string) *exec.Cmd {
	if !testCmdReceived(args...) {
		cs := []string{"-test.run=TestUsbListBasicExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestUsbListVerboseExecutionNoSerial", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFirstCommandFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestUsbListExecutionCommandFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorSecondCommandFailed(command string, args ...string) *exec.Cmd {
	if !testCmdReceived(args...) {
		cs := []string{"-test.run=TestUsbListBasicExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestUsbListExecutionCommandFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailedGetBus(command string, args ...string) *exec.Cmd {
	if !testCmdReceived(args...) {
		cs := []string{"-test.run=TestUsbListExecutionIncorrectBus", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestUsbListVerboseExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailedGetAddress(command string, args ...string) *exec.Cmd {
	if !testCmdReceived(args...) {
		cs := []string{"-test.run=TestUsbListExecutionIncorrectAddress", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestUsbListVerboseExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestUsbListBasicExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestUsbListVerboseExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb_verbose.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestUsbListVerboseExecutionNoSerial(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb_verbose_no_serial_data.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestUsbListExecutionCommandFailed(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}

func TestUsbListExecutionIncorrectBus(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb_incorrect_usb_bus.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestUsbListExecutionIncorrectAddress(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb_incorrect_usb_address.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
