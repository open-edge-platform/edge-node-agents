// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main_test package implements integration test for the Cluster Agent
package main_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test is configured via environment files, paths to CA & mock executables,
// CA configuration file & mock certificate/key must be provided
var clusterAgentBinary string = os.Getenv("CA_BINARY_PATH")
var clusterOrchMockBinary string = os.Getenv("COM_BINARY_PATH")
var statusServerMockBinary string = os.Getenv("SS_BINARY_PATH")
var testConfig string = os.Getenv("TEST_CONFIG_PATH")
var testCertificate string = os.Getenv("TEST_CERT_PATH")
var testPrivateKey string = os.Getenv("TEST_KEY_PATH")
var caVersion string = os.Getenv("CA_VERSION")

// global timeout for entire test
const testTimeout = 60 * time.Second

var comListenAddress string = "localhost:12345"

const installFilePath = "/tmp/install"
const uninstallFilePath = "/tmp/uninstall"

var installCmd string = "touch " + installFilePath
var uninstallCmd string = "touch " + uninstallFilePath

// Execute Cluster Agent smoke test
// The test intent is to check if Cluster Agent is able to execute cluster installation & deinstallation command.
// Cluster Orch mock is used to return static data to CA. Dummy "touch /tmp/filepath" commands are used
// instead of real commands so they can be executed quickly and reliably.
// Test is successful if both files exist before test timeout expires.
func TestInstallUninstallCluster(t *testing.T) {
	defer removeInstallFiles()

	require.NoError(t, validateInputVars())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Starting Cluster Orch Mock...")
	comCmd, err := startMock(ctx, clusterOrchMockBinary, installCmd, uninstallCmd)
	require.NoError(t, err)

	fmt.Println("Starting Status Server Mock...")
	ssmCmd, err := startStatusServerMock(ctx, statusServerMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Cluster Agent...")
	caCmd, err := startClusterAgent(ctx, clusterAgentBinary)
	require.NoError(t, err)

	fmt.Println("Waiting for installation command execution...")
	require.NoError(t, waitForFile(ctx, installFilePath))
	fmt.Println("Installation command executed!")

	fmt.Println("Waiting for uninstallation command execution...")
	require.NoError(t, waitForFile(ctx, uninstallFilePath))
	fmt.Println("Uninstallation command executed!")

	fmt.Println("Stopping Cluster Agent...")
	_ = killProcess(ctx, caCmd.Process.Pid)
	err = caCmd.Wait()
	require.NoError(t, err)

	fmt.Println("Stopping Status Server Mock...")
	_ = killProcess(ctx, ssmCmd.Process.Pid)

	fmt.Println("Stopping Cluster Orch Mock...")
	_ = killProcess(ctx, comCmd.Process.Pid)
	// FIXME: Cluster Orch Mock doesn't exit cleanly on SIGTERM
	_ = comCmd.Wait()
	//	require.NoError(t, err)

	require.NoError(t, ctx.Err(), "Test took to long!")
}

// Execute `cluster-agent version` and check if it returns proper version and exits with 0 return code.
func TestVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Printf("Checking CA version; Expected: %v", caVersion)
	out, err := exec.CommandContext(ctx, clusterAgentBinary, "version").Output()
	require.NoError(t, err)

	require.Equal(t, caVersion+"\n", string(out))
}

// Execute `cluster-agent` without arguments. It should exit with 1 return code and print usage message.
func TestInvalidArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, clusterAgentBinary).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Usage")
}

func startMock(ctx context.Context, clusterOrchMockBinary, installCmd, uninstallCmd string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, clusterOrchMockBinary,
		"-certPath", testCertificate,
		"-keyPath", testPrivateKey,
		"-address", comListenAddress,
		"-installCmd", installCmd,
		"-uninstallCmd", uninstallCmd,
		"-integrationTestMode")
	return cmd, cmd.Start()
}

func startStatusServerMock(ctx context.Context, statusServerMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, statusServerMockBinary)
	return cmd, cmd.Start()
}

func startClusterAgent(ctx context.Context, clusterAgentBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, clusterAgentBinary, "-config", testConfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func killProcess(ctx context.Context, pid int) error {
	stop_cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("%v", pid))
	return stop_cmd.Run()
}

func waitForFile(ctx context.Context, filepath string) error {
	cyclicalTicker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("canceled")

		case <-cyclicalTicker.C:
			if _, err := os.Stat(filepath); err == nil {
				return nil
			}
		}
	}
}

func removeInstallFiles() {
	for _, p := range []string{installFilePath, uninstallFilePath} {
		if _, err := os.Stat(p); err == nil {
			os.Remove(p)
		}
	}
}

func validateInputVars() error {
	for _, p := range []string{clusterAgentBinary, clusterOrchMockBinary, testConfig, testCertificate, testPrivateKey} {
		if _, err := os.Stat(p); err != nil {
			return err
		}
	}
	return nil
}
