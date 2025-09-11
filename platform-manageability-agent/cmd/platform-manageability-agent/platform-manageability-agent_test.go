// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test is configured via environment files, paths to PMA and mock executables
// PMA configuration file & mock certificate/key must be provided
var platformManageabilityAgentBinary string = os.Getenv("PMA_BINARY_PATH")
var dmManagerMockBinary string = os.Getenv("DMM_BINARY_PATH")
var statusServerMockBinary string = os.Getenv("SS_BINARY_PATH")
var testConfig string = os.Getenv("TEST_CONFIG_PATH")
var testCertificate string = os.Getenv("TEST_CERT_PATH")
var testPrivateKey string = os.Getenv("TEST_KEY_PATH")
var pmaVersion string = os.Getenv("PMA_VERSION")

// global timeout for entire test
const testTimeout = 60 * time.Second

var comListenAddress string = "localhost:12345"

var pmaBuffer bytes.Buffer
var mockBuffer bytes.Buffer

// Execute Platform Manageability Agent smoke test
// The test intent is to check if Platform Manageability Agent is able to detect AMT status from the node
// and apply the required profile.
// DM Manager mock is used to return static data to the PMA.
// Test is successful if no errors are detected
func TestDeviceConfiguration(t *testing.T) {
	require.NoError(t, validateInputVars())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Starting DM Manager Mock...")
	dmCmd, err := startMock(ctx, dmManagerMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Status Server Mock...")
	ssmCmd, err := startStatusServerMock(ctx, statusServerMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Platform Manageability Agent...")
	pmaCmd, err := startPlatformManageabilityAgent(ctx, platformManageabilityAgentBinary)
	require.NoError(t, err)

	time.Sleep(20 * time.Second)
	require.Contains(t, pmaBuffer.String(), "Platform Manageability Agent started successfully")
	require.Contains(t, pmaBuffer.String(), "Successfully reported AMT status for host")
	require.Contains(t, pmaBuffer.String(), "Successfully retrieved activation details for host")
	require.Contains(t, pmaBuffer.String(), "Status Ready")
	require.Contains(t, mockBuffer.String(), "AMTStatusReport")
	require.Contains(t, mockBuffer.String(), "ActivationRequest")
	require.Contains(t, mockBuffer.String(), "ActivationResponseDetails")
	require.Contains(t, mockBuffer.String(), "ActivationResultRequest")
	pmaBuffer.Reset()
	mockBuffer.Reset()

	fmt.Println("Stopping Platform Manageability Agent...")
	_ = killProcess(ctx, pmaCmd.Process.Pid)
	err = pmaCmd.Wait()
	require.NoError(t, err)

	fmt.Println("Stopping Status Server Mock...")
	_ = killProcess(ctx, ssmCmd.Process.Pid)

	fmt.Println("Stopping DM Manager Mock...")
	_ = killProcess(ctx, dmCmd.Process.Pid)
	// FIXME: DM Manager Mock doesn't exit cleanly on SIGTERM
	_ = dmCmd.Wait()

	require.NoError(t, ctx.Err(), "Test took too long!")
}

// Execute `platform-manageability-agent version` and check if it returns proper version and exits with 0 return code
func TestVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Printf("Checking PMA version; Expected: %v", pmaVersion)
	out, err := exec.CommandContext(ctx, platformManageabilityAgentBinary, "version").Output()
	require.NoError(t, err)

	require.Equal(t, pmaVersion+"\n", string(out))
}

// Execute `platform-manageability-agent` without arguments. It should exit with 1 return code and priont usage message.
func TestInvalidArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, platformManageabilityAgentBinary).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Error: --config flag is required and must not be empty")

	out, err = exec.CommandContext(ctx, platformManageabilityAgentBinary, "-config", "").CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Error: --config flag is required and must not be empty")

	out, err = exec.CommandContext(ctx, platformManageabilityAgentBinary, "-config", "no/such/config").CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "unable to initialize configuration")
}

func startMock(ctx context.Context, dmManagerMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, dmManagerMockBinary,
		"-certPath", testCertificate,
		"-keyPath", testPrivateKey,
		"-address", comListenAddress)
	cmd.Stderr = &mockBuffer
	return cmd, cmd.Start()
}

func startStatusServerMock(ctx context.Context, statusServerMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, statusServerMockBinary)
	return cmd, cmd.Start()
}

func startPlatformManageabilityAgent(ctx context.Context, platformManageabilityAgentBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, platformManageabilityAgentBinary, "-config", testConfig)
	cmd.Stdout = io.MultiWriter(&pmaBuffer, os.Stdout)
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func killProcess(ctx context.Context, pid int) error {
	stop_cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("%v", pid))
	return stop_cmd.Run()
}

func validateInputVars() error {
	for _, p := range []string{platformManageabilityAgentBinary, dmManagerMockBinary, testConfig, testCertificate, testPrivateKey} {
		if _, err := os.Stat(p); err != nil {
			return err
		}
	}
	return nil
}
