// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package device_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/device"
)

var testVersion = "16.1.27"
var testHostname = "testhost"
var testOperationalState = "enabled"
var testBuildNumber = "2176"
var testSku = "16392"
var testFeatures = "AMT Pro Corporate"
var testUuid = "1234abcd-ef56-7890-abcd-123456ef7890"
var testControlMode = "activated in client control mode"
var testDnsSuffix = "test.com"
var testNetworkStatus = "direct"
var testRemoteStatus = "not connected"
var testRemoteTrigger = "user initiated"

func getExpectedResult(version string, hostname string, opState string, buildNum string, sku string,
	features string, uuid string, controlMode string, dnsSuffix string, networkStatus string,
	remoteStatus string, remoteTrigger string) device.AMTInfo {
	return device.AMTInfo{
		Version:          version,
		Hostname:         hostname,
		OperationalState: opState,
		BuildNumber:      buildNum,
		Sku:              sku,
		Features:         features,
		Uuid:             uuid,
		ControlMode:      controlMode,
		DNSSuffix:        dnsSuffix,
		RAS: device.RASInfo{
			NetworkStatus: networkStatus,
			RemoteStatus:  remoteStatus,
			RemoteTrigger: remoteTrigger,
			MPSHostname:   "",
		},
	}
}

func Test_GetDeviceInfo(t *testing.T) {
	res, err := device.GetDeviceInfo(testSuccess)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoFailed(t *testing.T) {
	res, err := device.GetDeviceInfo(testFailure)
	var expected device.AMTInfo
	assert.Error(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoSystemUuidFailed(t *testing.T) {
	res, err := device.GetDeviceInfo(testFailureSystemUuid)
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

func Test_GetDeviceInfoMissingVersionNumber(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingVersionNumber)
	expected := getExpectedResult("", testHostname, testOperationalState, testBuildNumber, testSku, testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingHostname(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingHostname)
	expected := getExpectedResult(testVersion, "", testOperationalState, testBuildNumber, testSku, testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingOperationalState(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingOperationalState)
	expected := getExpectedResult(testVersion, testHostname, "", testBuildNumber, testSku, testFeatures, testUuid,
		testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingBuildNumber(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingBuildNumber)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, "", testSku, testFeatures, testUuid,
		testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingSku(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingSku)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, "", testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingFeatures(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingFeatures)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku, "",
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingUuid(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingUuid)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingControlMode(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingControlMode)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, "", testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingDnsSuffix(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingDnsSuffix)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, "", testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingRasInfo(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingRasInfo)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, "", "", "")
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingNetworkStatus(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingNetworkStatus)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, "", testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingRemoteStatus(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingRemoteStatus)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, "", testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetDeviceInfoMissingRemoteTrigger(t *testing.T) {
	res, err := device.GetDeviceInfo(testMissingRemoteTrigger)
	expected := getExpectedResult(testVersion, testHostname, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, "")
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func testCmd(testFunc string, command string, args ...string) *exec.Cmd {
	cs := []string{fmt.Sprintf("-test.run=%s", testFunc), "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testSuccess(command string, args ...string) *exec.Cmd {
	return testCmd("TestSuccess", command, args...)
}

func testFailure(command string, args ...string) *exec.Cmd {
	return testCmd("TestFailure", command, args...)
}

func testFailureSystemUuid(command string, args ...string) *exec.Cmd {
	if strings.Contains(args[0], "rpc") {
		return testCmd("TestMissingUuid", command, args...)
	} else {
		return testCmd("TestFailure", command, args...)
	}
}

func testFailureUnmarshal(command string, args ...string) *exec.Cmd {
	return testCmd("TestFailureUnmarshal", command, args...)
}

func testMissingVersionNumber(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingVersionNumber", command, args...)
}

func testMissingHostname(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingHostname", command, args...)
}

func testMissingOperationalState(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingOperationalState", command, args...)
}

func testMissingBuildNumber(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingBuildNumber", command, args...)
}

func testMissingSku(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingSku", command, args...)
}

func testMissingFeatures(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingFeatures", command, args...)
}

func testMissingUuid(command string, args ...string) *exec.Cmd {
	if strings.Contains(args[0], "rpc") {
		return testCmd("TestMissingUuid", command, args...)
	} else {
		return testCmd("TestSystemUuid", command, args...)
	}
}

func testMissingControlMode(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingControlMode", command, args...)
}

func testMissingDnsSuffix(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingDnsSuffix", command, args...)
}

func testMissingRasInfo(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingRasInfo", command, args...)
}

func testMissingNetworkStatus(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingNetworkStatus", command, args...)
}

func testMissingRemoteStatus(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingRemoteStatus", command, args...)
}

func testMissingRemoteTrigger(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingRemoteTrigger", command, args...)
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

func TestMissingVersionNumber(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingversion.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingHostname(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missinghostname.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingOperationalState(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingopstate.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingBuildNumber(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingbuildnum.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingSku(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingsku.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingFeatures(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingfeature.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingUuid(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missinguuid.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSystemUuid(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", testUuid)
	os.Exit(0)
}

func TestMissingControlMode(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingcontrolmode.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingDnsSuffix(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingdnssuffix.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingRasInfo(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingrasinfo.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingNetworkStatus(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingnetworkstatus.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingRemoteStatus(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingremotestatus.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMissingRemoteTrigger(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingremotetrigger.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
