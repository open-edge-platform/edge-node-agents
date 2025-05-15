// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package memory_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/memory"
)

var expectedTotal uint64 = 17179869184

func Test_GetMemory(t *testing.T) {
	out, err := memory.GetMemory(testCmdExecutorSuccessLSMEM)
	assert.Equal(t, expectedTotal, out)
	require.NoError(t, err)
}

func Test_GetMemoryUnmarshalFailed(t *testing.T) {
	out, err := memory.GetMemory(testCmdExecutorFailedUnmarshal)
	assert.Equal(t, uint64(0), out)
	require.Error(t, err)
}

func Test_GetMemoryCommandFailed(t *testing.T) {
	out, err := memory.GetMemory(testCmdExecutorCommandFailed)
	assert.Equal(t, uint64(0), out)
	require.Error(t, err)
}

func testCmdExecutorSuccessLSMEM(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestMemoryListExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorFailedUnmarshal(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestMemoryListExecutionUnmarshalFail", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorCommandFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestMemoryListExecutionCommandFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestMemoryListExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_memory.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestMemoryListExecutionUnmarshalFail(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", string("not a json"))
	os.Exit(0)
}

func TestMemoryListExecutionCommandFailed(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}
