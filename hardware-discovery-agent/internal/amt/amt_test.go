// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package amt_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/amt"
)

var testVersion = "16.1.27"
var testDeviceName = "testhost"
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

func getExpectedResult(version string, deviceName string, opState string, buildNum string, sku string,
	features string, uuid string, controlMode string, dnsSuffix string, networkStatus string,
	remoteStatus string, remoteTrigger string) *amt.AmtInfo {
	if networkStatus == "" && remoteStatus == "" && remoteTrigger == "" {
		return &amt.AmtInfo{
			Version:          version,
			DeviceName:       deviceName,
			OperationalState: opState,
			BuildNumber:      buildNum,
			Sku:              sku,
			Features:         features,
			Uuid:             uuid,
			ControlMode:      controlMode,
			DNSSuffix:        dnsSuffix,
		}
	} else {
		return &amt.AmtInfo{
			Version:          version,
			DeviceName:       deviceName,
			OperationalState: opState,
			BuildNumber:      buildNum,
			Sku:              sku,
			Features:         features,
			Uuid:             uuid,
			ControlMode:      controlMode,
			DNSSuffix:        dnsSuffix,
			RAS: &amt.RASInfo{
				NetworkStatus: networkStatus,
				RemoteStatus:  remoteStatus,
				RemoteTrigger: remoteTrigger,
				MPSHostname:   "",
			},
		}
	}
}

func Test_GetAmtInfo(t *testing.T) {
	res, err := amt.GetAmtInfo(testSuccess)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoFailed(t *testing.T) {
	res, err := amt.GetAmtInfo(testFailure)
	assert.Error(t, err)
	assert.Equal(t, &amt.AmtInfo{}, res)
}

func Test_GetAmtInfoSystemUuidFailed(t *testing.T) {
	res, err := amt.GetAmtInfo(testFailureSystemUuid)
	assert.Error(t, err)
	assert.Equal(t, &amt.AmtInfo{}, res)
}

func Test_GetAmtInfoFailedUnmarshal(t *testing.T) {
	res, err := amt.GetAmtInfo(testFailureUnmarshal)
	assert.Error(t, err)
	assert.Equal(t, &amt.AmtInfo{}, res)
}

func Test_GetAmtInfoMissingVersionNumber(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingVersionNumber)
	expected := getExpectedResult("", testDeviceName, testOperationalState, testBuildNumber, testSku, testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingDeviceName(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingDeviceName)
	expected := getExpectedResult(testVersion, "", testOperationalState, testBuildNumber, testSku, testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingOperationalState(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingOperationalState)
	expected := getExpectedResult(testVersion, testDeviceName, "", testBuildNumber, testSku, testFeatures, testUuid,
		testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingBuildNumber(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingBuildNumber)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, "", testSku, testFeatures, testUuid,
		testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingSku(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingSku)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, "", testFeatures,
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingFeatures(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingFeatures)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku, "",
		testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingUuid(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingUuid)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingControlMode(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingControlMode)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, "", testDnsSuffix, testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingDnsSuffix(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingDnsSuffix)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, "", testNetworkStatus, testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingRasInfo(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingRasInfo)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, "", "", "")
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingNetworkStatus(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingNetworkStatus)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, "", testRemoteStatus, testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingRemoteStatus(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingRemoteStatus)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
		testFeatures, testUuid, testControlMode, testDnsSuffix, testNetworkStatus, "", testRemoteTrigger)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func Test_GetAmtInfoMissingRemoteTrigger(t *testing.T) {
	res, err := amt.GetAmtInfo(testMissingRemoteTrigger)
	expected := getExpectedResult(testVersion, testDeviceName, testOperationalState, testBuildNumber, testSku,
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

func testMissingDeviceName(command string, args ...string) *exec.Cmd {
	return testCmd("TestMissingDeviceName", command, args...)
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

func TestMissingDeviceName(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo_missingdevicename.json")
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
