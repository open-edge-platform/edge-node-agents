// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main_test package implements integration test for the Node Agent
package main_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test is configured via environment files, paths to NA & HMM executables,
// Node configuration file & HMM certificate/key must be provided
var nodeAgentBinary string = os.Getenv("NA_BINARY_PATH")
var hostMgrMockBinary string = os.Getenv("HMM_BINARY_PATH")
var testConfig string = os.Getenv("TEST_CONFIG_PATH")
var naVersion string = os.Getenv("NA_VERSION")
var testCertificate string = os.Getenv("TEST_CERT_PATH")
var testPrivateKey string = os.Getenv("TEST_KEY_PATH")

// global timeout for entire test
const testTimeout = 90 * time.Second

// Execute `nodeagent` without arguments. It should exit with 1 return code and print usage message.
func TestInvalidArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, nodeAgentBinary).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Usage")
}

// Execute `node-agent version` and check if it returns proper version and exits with 0 return code.
func TestVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Printf("Checking NA version; Expected: %v", naVersion)
	out, err := exec.CommandContext(ctx, nodeAgentBinary, "version").Output()
	require.NoError(t, err)

	require.Equal(t, naVersion+"\n", string(out))
}

// Execute node  Agent smoke test
// The test intent is to check if NodeAgent is able to send node status to Host Manager.
// Host Manager mock is used to return static data to NA.
// Test is successful if data is sent and received.
func TestNAgentAndMock(t *testing.T) {

	require.NoError(t, validateInputVars())

	err := os.MkdirAll("/tmp/creds", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("/tmp/creds")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Starting Host Manager Mock...")
	hmmCmd, err := startMock(ctx, hostMgrMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Node Agent...")
	naCmd, pipe, err := startNAgent(ctx, nodeAgentBinary)
	require.NoError(t, err)

	reader := bufio.NewReader(pipe)
	line, err := reader.ReadString('\n')
	counter := 0
	for err == nil {
		fmt.Print(line)
		// Expect error status to be sent as status client are yet to be mocked
		if strings.Contains(line, "UpdateInstanceStatus sent successfully: INSTANCE_STATUS_ERROR") {
			if counter++; counter == 3 {
				break
			}
		}
		line, err = reader.ReadString('\n')
	}

	fmt.Println("Stopping Node Agent...")
	err = killProcess(ctx, naCmd.Process.Pid)
	require.NoError(t, err)

	fmt.Println("Stopping Host Manager Mock...")
	_ = killProcess(ctx, hmmCmd.Process.Pid)
	_ = hmmCmd.Wait()

	require.NoError(t, ctx.Err(), "Test took too long!")
}

func startMock(ctx context.Context, hostMgrMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, hostMgrMockBinary,
		"-address", "localhost:8080",
		"-certPath", testCertificate,
		"-keyPath", testPrivateKey)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func startNAgent(ctx context.Context, nodeAgentBinary string) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, nodeAgentBinary, "-config", testConfig)
	//cmd.Stdout = os.Stdout
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = os.Stderr
	return cmd, pipe, cmd.Start()
}

func killProcess(ctx context.Context, pid int) error {
	na_stop_cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("%v", pid))
	return na_stop_cmd.Run()
}

func validateInputVars() error {
	for _, p := range []string{nodeAgentBinary, hostMgrMockBinary, testConfig} {
		if _, err := os.Stat(p); err != nil {
			return err
		}
	}
	return nil
}
