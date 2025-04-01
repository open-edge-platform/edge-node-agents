// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main_test package implements integration test for the Platform Telemetry Agent
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

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/stretchr/testify/require"
)

// Test is configured via environment files, paths to PTA & TMM executables,
// PTA configuration file & TMM certificate/key must be provided
var ptAgentBinary string = os.Getenv("PTA_BINARY_PATH")
var serverMockBinary string = os.Getenv("TMM_BINARY_PATH")
var statusServerMockBinary string = os.Getenv("SS_BINARY_PATH")
var workingConfig string = os.Getenv("WORKING_CONFIG_PATH")
var ptaBuffer bytes.Buffer
var mockBuffer bytes.Buffer

// global timeout for entire test
const testTimeout = 90 * time.Second

var tmmListenAddress string = "0.0.0.0:5001"

func TestVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ver, err := utils.ReadFileNoLinks("../../VERSION")
	require.NoError(t, err)

	version := strings.TrimSuffix(string(ver), "\n")

	out, _ := exec.CommandContext(ctx, ptAgentBinary, "version").CombinedOutput()
	require.Contains(t, string(out), version)
}

// Execute Platform Telemetry Agent smoke test
// The test intent is to check if Platform Telemetry Agent is able to retrieve config info.
// server Mock mock is used to stub out real server and return static data to PTA.
// Test is successful if data is sent and received.
func TestPTAgentAndMock(t *testing.T) {

	require.NoError(t, validateInputVars())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	fmt.Println("Starting Server Mock...")
	tmmCmd, err := startServerMock(ctx, serverMockBinary)
	require.NoError(t, err)

	fmt.Println("Testing working configuration...")
	startStopPTA(ctx, workingConfig, t)
	fmt.Println(ptaBuffer.String())
	fmt.Println(mockBuffer.String())
	require.Contains(t, ptaBuffer.String(), "Connecting to telemetrymgr")
	require.Contains(t, mockBuffer.String(), "Listen")
	ptaBuffer.Reset()
	mockBuffer.Reset()

	fmt.Println("Stopping Server Mock...")
	_ = killProcess(ctx, tmmCmd.Process.Pid)
	_ = tmmCmd.Wait()
	//	require.NoError(t, err)

	require.NoError(t, ctx.Err(), "Test took too long!")
}

func startStopPTA(ctx context.Context, config string, t *testing.T) {
	fmt.Println("Starting Status Server Mock...")
	ssmCmd, err := startStatusServerMock(ctx, statusServerMockBinary)
	require.NoError(t, err)

	fmt.Println("Starting Platform Telemetry Agent...")
	ptaCmd, err := startPTAgent(ctx, ptAgentBinary, config)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	fmt.Println("Stopping Status Server Mock...")
	_ = killProcess(ctx, ssmCmd.Process.Pid)

	fmt.Println("Stopping Platform Telemetry Agent...")
	err = killProcess(ctx, ptaCmd.Process.Pid)
	require.NoError(t, err)
	err = ptaCmd.Wait()
	require.NoError(t, err)
}

func startStatusServerMock(ctx context.Context, statusServerMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, statusServerMockBinary)
	return cmd, cmd.Start()
}

func startServerMock(ctx context.Context, serverMockBinary string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, serverMockBinary,
		"-address", tmmListenAddress)
	cmd.Stderr = &mockBuffer
	return cmd, cmd.Start()
}

func startPTAgent(ctx context.Context, ptAgentBinary string, config string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, ptAgentBinary, "-config", config)
	cmd.Stdout = &ptaBuffer
	return cmd, cmd.Start()
}

func killProcess(ctx context.Context, pid int) error {
	stop_cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("%v", pid))
	return stop_cmd.Run()
}

func validateInputVars() error {
	for _, p := range []string{ptAgentBinary, serverMockBinary, workingConfig} {
		if _, err := os.Stat(p); err != nil {
			return err
		}
	}
	return nil
}
