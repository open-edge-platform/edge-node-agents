// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package system_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/system"
	"github.com/stretchr/testify/assert"
)

var expectedProductName = "Test Product"
var expectedReleaseDate = "21/09/2023"
var expectedSN = "26B06S3"
var expectedUUID = "ec2b1731-304d-853d-cac8-659fe7fcb6ab"
var expectedVendor = "Test Vendor"
var expectedVersion = "1.2.3"
var expectedKernelVersion = "5.15.0-82-generic"
var expectedKernelPlatform = "x86_64"
var expectedKernelOS = "GNU/Linux"
var expectedReleaseId = "Ubuntu"
var expectedReleaseVersion = "Ubuntu 20.04 LTS"

// Test cases
func TestGetBiosInfoSuccess(t *testing.T) {
	biosInfo, err := system.GetBiosInfo(testCmdExecutorSuccessBiosInfo)

	expectedInfo := &system.Bios{
		Version: expectedVersion,
		RelDate: expectedReleaseDate,
		Vendor:  expectedVendor,
	}

	assert.NoError(t, err)
	assert.Equal(t, expectedInfo, biosInfo)
}

func TestGetBiosInfoVersionFailure(t *testing.T) {
	biosInfo, err := system.GetBiosInfo(testCmdExecutorFailure)
	assert.Error(t, err)
	assert.Empty(t, biosInfo)
}

func TestGetBiosInfoReleaseDateFailure(t *testing.T) {
	biosInfo, err := system.GetBiosInfo(testCmdExecutorFailureBiosInfoReleaseDate)
	assert.Error(t, err)
	assert.Empty(t, biosInfo)
}

func TestGetBiosInfoVendorFailure(t *testing.T) {
	biosInfo, err := system.GetBiosInfo(testCmdExecutorFailureBiosInfoVendor)
	assert.Error(t, err)
	assert.Empty(t, biosInfo)
}

func TestGetOsInfoSuccess(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorSuccessOsInfo)

	expectedConfig := []*system.OsMetadata{}
	hwPlatform := &system.OsMetadata{
		Key:   "Platform",
		Value: expectedKernelPlatform,
	}
	expectedConfig = append(expectedConfig, hwPlatform)
	osType := &system.OsMetadata{
		Key:   "Operating System",
		Value: expectedKernelOS,
	}
	expectedConfig = append(expectedConfig, osType)

	expectedMetadata := []*system.OsMetadata{}
	releaseMetadata := &system.OsMetadata{
		Key:   "Codename",
		Value: "jammy",
	}
	expectedMetadata = append(expectedMetadata, releaseMetadata)

	expectedInfo := &system.Os{
		Kernel: &system.OsKern{
			Version: expectedKernelVersion,
			Config:  expectedConfig,
		},
		Release: &system.OsRel{
			Id:       expectedReleaseId,
			Version:  expectedReleaseVersion,
			Metadata: expectedMetadata,
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, expectedInfo, osInfo)
}

func TestGetOsInfoKernelVersionFailure(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorFailure)
	assert.Error(t, err)
	assert.Empty(t, osInfo)
}

func TestGetOsInfoKernelConfigFailure(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorFailureOsInfoKernelPlatform)
	assert.Error(t, err)
	assert.Empty(t, osInfo)

	osInfo, err = system.GetOsInfo(testCmdExecutorFailureOsInfoKernelOs)
	assert.Error(t, err)
	assert.Empty(t, osInfo)
}

func TestGetOsInfoReleaseIdFailure(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorFailureOsInfoReleaseId)
	assert.Error(t, err)
	assert.Empty(t, osInfo)
}

func TestGetOsInfoReleaseVersionFailure(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorFailureOsInfoReleaseVersion)
	assert.Error(t, err)
	assert.Empty(t, osInfo)
}

func TestGetOsInfoReleaseMetadataFailure(t *testing.T) {
	osInfo, err := system.GetOsInfo(testCmdExecutorFailureOsInfoReleaseMetadata)
	assert.Error(t, err)
	assert.Empty(t, osInfo)
}

func TestGetProductNameSuccess(t *testing.T) {
	pn, err := system.GetProductName(testCmdExecutorSuccessProductName)
	assert.NoError(t, err)
	assert.Equal(t, expectedProductName, pn)
}

func TestGetProductNameFailure(t *testing.T) {
	pn, err := system.GetProductName(testCmdExecutorFailure)
	assert.Error(t, err)
	assert.Empty(t, pn)
}

func TestGetSerialNumberSuccess(t *testing.T) {
	sn, err := system.GetSerialNumber(testCmdExecutorSuccessSN)
	assert.NoError(t, err)
	assert.Equal(t, expectedSN, sn)
}

func TestGetSerialNumberFailure(t *testing.T) {
	sn, err := system.GetSerialNumber(testCmdExecutorFailure)
	assert.Error(t, err)
	assert.Empty(t, sn)
}

func TestGetUuidSuccess(t *testing.T) {
	uuid, err := system.GetSystemUUID(testCmdExecutorSuccessUUID)
	assert.NoError(t, err)
	assert.Equal(t, expectedUUID, uuid)
}

func TestGetUuidFailure(t *testing.T) {
	uuid, err := system.GetSystemUUID(testCmdExecutorFailure)
	assert.Error(t, err)
	assert.Empty(t, uuid)
}

// Test executors are functions that initialize a new exec.Cmd, one which will
// simply call mock function rather than the command it is provided. It will
// also pass through the command and its arguments

func testCmdExecutorSuccessUUID(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSerialNumberExecutionSuccessUUID", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorSuccessSN(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSerialNumberExecutionSuccessSN", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorSuccessProductName(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSystemExecutionSuccessProductName", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdOsInfoKernelTest(command string, args ...string) []string {
	if strings.Contains(args[0], "-r") {
		return []string{"--test.run=TestOsInfoKernelVersionSuccess", "--", command}
	} else if strings.Contains(args[0], "-i") {
		return []string{"--test.run=TestOsInfoKernelPlatformSuccess", "--", command}
	} else {
		return []string{"--test.run=TestOsInfoKernelOperatingSystemSuccess", "--", command}
	}
}

func testCmdOsInfoReleaseTest(command string, isErrorCase bool, args ...string) []string {
	if strings.Contains(args[0], "-i") {
		return []string{"--test.run=TestOsInfoReleaseIdSuccess", "--", command}
	} else if strings.Contains(args[0], "-d") {
		return []string{"--test.run=TestOsInfoReleaseVersionSuccess", "--", command}
	} else {
		if !isErrorCase {
			return []string{"--test.run=TestOsInfoReleaseMetadataSuccess", "--", command}
		} else {
			return []string{"--test.run=TestGenericExecutionFailure", "--", command}
		}
	}
}

func testCmdExecutorSuccessOsInfo(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "uname") {
		cs := testCmdOsInfoKernelTest(command, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	} else {
		cs := testCmdOsInfoReleaseTest(command, false, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func testCmdExecutorFailureOsInfoKernelPlatform(command string, args ...string) *exec.Cmd {
	var cs []string
	if strings.Contains(args[0], "-r") {
		cs = []string{"--test.run=TestOsInfoKernelVersionSuccess", "--", command}
	} else {
		cs = []string{"--test.run=TestGenericExecutionFailure", "--", command}
	}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailureOsInfoKernelOs(command string, args ...string) *exec.Cmd {
	var cs []string
	if strings.Contains(args[0], "-r") {
		cs = []string{"--test.run=TestOsInfoKernelVersionSuccess", "--", command}
	} else if strings.Contains(args[0], "-i") {
		cs = []string{"--test.run=TestOsInfoKernelPlatformSuccess", "--", command}
	} else {
		cs = []string{"--test.run=TestGenericExecutionFailure", "--", command}
	}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailureOsInfoReleaseId(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "uname") {
		cs := testCmdOsInfoKernelTest(command, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	} else {
		cs := []string{"--test.run=TestGenericExecutionFailure", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func testCmdExecutorFailureOsInfoReleaseVersion(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "uname") {
		cs := testCmdOsInfoKernelTest(command, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	} else {
		var cs []string
		if strings.Contains(args[0], "-i") {
			cs = []string{"--test.run=TestOsInfoReleaseIdSuccess", "--", command}
		} else {
			cs = []string{"--test.run=TestGenericExecutionFailure", "--", command}
		}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func testCmdExecutorFailureOsInfoReleaseMetadata(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "uname") {
		cs := testCmdOsInfoKernelTest(command, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	} else {
		cs := testCmdOsInfoReleaseTest(command, true, args...)
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func testCmdExecutorFailure(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestGenericExecutionFailure", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorSuccessBiosInfo(command string, args ...string) *exec.Cmd {
	var cs []string
	if strings.Contains(args[2], "bios-version") {
		cs = []string{"--test.run=TestBiosInfoVersionSuccess", "--", command}
	} else if strings.Contains(args[2], "bios-release-date") {
		cs = []string{"--test.run=TestBiosInfoReleaseDateSuccess", "--", command}
	} else {
		cs = []string{"--test.run=TestBiosInfoVendorSuccess", "--", command}
	}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailureBiosInfoReleaseDate(command string, args ...string) *exec.Cmd {
	var cs []string
	if strings.Contains(args[2], "bios-version") {
		cs = []string{"--test.run=TestBiosInfoVersionSuccess", "--", command}
	} else {
		cs = []string{"--test.run=TestGenericExecutionFailure", "--", command}
	}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailureBiosInfoVendor(command string, args ...string) *exec.Cmd {
	var cs []string
	if strings.Contains(args[2], "bios-version") {
		cs = []string{"--test.run=TestBiosInfoVersionSuccess", "--", command}
	} else if strings.Contains(args[2], "bios-release-date") {
		cs = []string{"--test.run=TestBiosInfoReleaseDateSuccess", "--", command}
	} else {
		cs = []string{"--test.run=TestGenericExecutionFailure", "--", command}
	}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

// Test executor mock functions are called as a substitute for a shell command,
// the GO_TEST_PROCESS flag ensures that if it is called as part of the test suite, it is
// skipped.

func TestOsInfoKernelVersionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedKernelVersion)
	os.Exit(0)
}

func TestOsInfoKernelPlatformSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedKernelPlatform)
	os.Exit(0)
}

func TestOsInfoKernelOperatingSystemSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedKernelOS)
	os.Exit(0)
}

func TestOsInfoReleaseIdSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", "Distributor ID: "+expectedReleaseId)
	os.Exit(0)
}

func TestOsInfoReleaseVersionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", "Description:    "+expectedReleaseVersion)
	os.Exit(0)
}

func TestOsInfoReleaseMetadataSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", "Codename:       jammy")
	os.Exit(0)
}

func TestSystemExecutionSuccessProductName(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedProductName)
	os.Exit(0)
}

func TestBiosInfoVersionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedVersion)
	os.Exit(0)
}

func TestBiosInfoReleaseDateSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedReleaseDate)
	os.Exit(0)
}

func TestBiosInfoVendorSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedVendor)
	os.Exit(0)
}

func TestSerialNumberExecutionSuccessUUID(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedUUID)
	os.Exit(0)
}

func TestSerialNumberExecutionSuccessSN(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stdout, "%v", expectedSN)
	os.Exit(0)
}

func TestGenericExecutionFailure(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	// Print out the test value to stdout
	fmt.Fprintf(os.Stderr, "failed to execute")
	os.Exit(1)
}
