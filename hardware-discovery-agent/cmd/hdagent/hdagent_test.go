// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main_test package implements integration test for the Hardware Discovery Agent
package main_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test is configured via environment files, paths to HDA & HMM executables,
// HDA configuration file & HMM certificate/key must be provided
var (
	hdAgentBinary           string = os.Getenv("HDA_BINARY_PATH")
	hostMgrMockBinary       string = os.Getenv("HMM_BINARY_PATH")
	statusServerMockBinary  string = os.Getenv("SS_BINARY_PATH")
	workingConfig           string = os.Getenv("WORKING_CONFIG_PATH")
	errorWrongAddressConfig string = os.Getenv("ERROR_WRONG_ADDRESS_CONFIG_PATH")
	debugGoodAddressConfig  string = os.Getenv("DEBUG_GOOD_ADDRESS_CONFIG_PATH")
	infoWrongAddressConfig  string = os.Getenv("INFO_WRONG_ADDRESS_CONFIG_PATH")
	testCertificate         string = os.Getenv("TEST_CERT_PATH")
	testPrivateKey          string = os.Getenv("TEST_KEY_PATH")
	noConfig                string = "no/such/config"
	hdaBuffer               bytes.Buffer
	mockBuffer              bytes.Buffer
)

// global timeout for entire test
const testTimeout = 90 * time.Second

var hmmListenAddress string = "localhost:12345"

// Execute `hdagent` without arguments. It should exit with 1 return code and print usage message.
func TestMissingArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, hdAgentBinary, "-config", "").CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "-config must be provided")

	out, err = exec.CommandContext(ctx, hdAgentBinary, "-config", noConfig).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "loading configuration failed")
}

func TestVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ver, err := os.ReadFile("../../VERSION")
	require.NoError(t, err)

	version := strings.TrimSuffix(string(ver), "\n")

	out, err := exec.CommandContext(ctx, hdAgentBinary, "version").CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), "v"+version)
}

// Execute Hardware Discovery Agent smoke test
// The test intent is to check if Hardware Discovery Agent is able to discover and send hardware info.
// Host manager mock is used to return static data to HDA.
// Test is successful if data is sent and received.
func TestHDAgentAndMock(t *testing.T) {
	require.NoError(t, validateInputVars())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Starting host manager mock...")
	HMMCmd, err := startHMMock(ctx, hostMgrMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Status Server Mock...")
	ssmCmd, err := startStatusServerMock(ctx, statusServerMockBinary)
	require.NoError(t, err)

	fmt.Println("Testing working configuration...")
	// Use high idle time to ensure status relay to node agent happens at
	// least once after HM call, the cycle time for status relay is 10s
	startStopHDA(ctx, workingConfig, 20*time.Second, t)
	require.Contains(t, hdaBuffer.String(), "HW Discovery Agent comms: UpdateHostSystemInfoByGUIDRequest sent successfully")
	require.Contains(t, hdaBuffer.String(), "Status Ready")
	require.Contains(t, mockBuffer.String(), "UpdateSystemInfoByGUIDResponse")
	hdaBuffer.Reset()
	mockBuffer.Reset()

	fmt.Println("Testing wrong host manager address ERROR level configuration...")
	startStopHDA(ctx, errorWrongAddressConfig, 5*time.Second, t)
	require.Contains(t, hdaBuffer.String(), ":8080: connect: connection refused")
	hdaBuffer.Reset()

	fmt.Println("Testing working DEBUG level configuration...")
	startStopHDA(ctx, debugGoodAddressConfig, 5*time.Second, t)
	require.Contains(t, hdaBuffer.String(), "Sending System info")
	require.Contains(t, mockBuffer.String(), "system_info")
	hdaBuffer.Reset()
	mockBuffer.Reset()

	fmt.Println("Testing wrong host manager address INFO level configuration...")
	startStopHDA(ctx, infoWrongAddressConfig, 5*time.Second, t)
	require.Contains(t, hdaBuffer.String(), "UpdateHostSystemInfoByGUID failed!")
	hdaBuffer.Reset()

	fmt.Println("Stopping host manager mock...")
	_ = killProcess(ctx, HMMCmd.Process.Pid)
	// FIXME: HMM doesn't exit cleanly now on SIGTERM
	_ = HMMCmd.Wait()
	//	require.NoError(t, err)

	fmt.Println("Stopping Status Server Mock...")
	_ = killProcess(ctx, ssmCmd.Process.Pid)

	require.NoError(t, ctx.Err(), "Test took too long!")
}

func TestHMDisconnectAndRecovery(t *testing.T) {
	require.NoError(t, validateInputVars())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Testing server disconnect and recovery...")
	fmt.Println("Starting host manager mock...")
	HMMCmd, err := startHMMock(ctx, hostMgrMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Status Server Mock...")
	ssmCmd, err := startStatusServerMock(ctx, statusServerMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Hardware Discovery Agent...")
	hdaCmd, err := startHDAgent(ctx, hdAgentBinary, workingConfig)
	require.NoError(t, err)
	time.Sleep(15 * time.Second)

	require.Contains(t, hdaBuffer.String(), "Status Ready")
	hdaBuffer.Reset()

	// Simulate server disconnect
	_ = killProcess(ctx, HMMCmd.Process.Pid)
	_ = HMMCmd.Wait()
	time.Sleep(15 * time.Second)

	require.Contains(t, hdaBuffer.String(), "Status Not Ready")
	hdaBuffer.Reset()

	// Restart Status Server Mock
	fmt.Println("Restarting host manager mock...")
	HMMCmd, err = startHMMock(ctx, hostMgrMockBinary)
	require.NoError(t, err)
	time.Sleep(15 * time.Second)

	require.Contains(t, hdaBuffer.String(), "Status Ready")
	hdaBuffer.Reset()

	fmt.Println("Stopping hd agent...")
	_ = killProcess(ctx, hdaCmd.Process.Pid)
	fmt.Println("Stopping host manager mock...")
	_ = killProcess(ctx, HMMCmd.Process.Pid)
	_ = HMMCmd.Wait()

	fmt.Println("Stopping Status Server Mock...")
	_ = killProcess(ctx, ssmCmd.Process.Pid)

	require.NoError(t, ctx.Err(), "Test took too long!")
}

func startStopHDA(ctx context.Context, config string, sleepDuration time.Duration, t *testing.T) {
	fmt.Println("Starting Hardware Discovery Agent...")
	hdaCmd, err := startHDAgent(ctx, hdAgentBinary, config)
	require.NoError(t, err)

	time.Sleep(sleepDuration)

	fmt.Println("Stopping Hardware Discovery Agent...")
	err = killProcess(ctx, hdaCmd.Process.Pid)
	require.NoError(t, err)
	err = hdaCmd.Wait()
	require.NoError(t, err)
}

func startHMMock(ctx context.Context, hostMgrMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, hostMgrMockBinary,
		"-certPath", testCertificate,
		"-keyPath", testPrivateKey,
		"-address", hmmListenAddress)
	cmd.Stderr = &mockBuffer
	return cmd, cmd.Start()
}

func startStatusServerMock(ctx context.Context, statusServerMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, statusServerMockBinary)
	return cmd, cmd.Start()
}

func startHDAgent(ctx context.Context, hdAgentBinary string, config string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, hdAgentBinary, "-config", config)
	cmd.Stdout = &hdaBuffer
	return cmd, cmd.Start()
}

func killProcess(ctx context.Context, pid int) error {
	hda_stop_cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("%v", pid))
	return hda_stop_cmd.Run()
}

func validateInputVars() error {
	for _, p := range []string{hdAgentBinary, hostMgrMockBinary, workingConfig, testCertificate, testPrivateKey} {
		if _, err := os.Stat(p); err != nil {
			return err
		}
	}
	return nil
}
