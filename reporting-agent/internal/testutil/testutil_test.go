// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSetMockOutputAndTestCmdExecutorSuccess checks that TestCmdExecutor returns the correct output and error for a registered command.
func TestSetMockOutputAndTestCmdExecutorSuccess(t *testing.T) {
	SetMockOutput("echo", []string{"foo", "bar"}, []byte("baz"), nil)
	cmd := TestCmdExecutor("echo", "foo", "bar")
	out, err := cmd.Output()
	require.NoError(t, err, "TestCmdExecutor should not return error for registered command")
	require.Equal(t, []byte("baz"), out, "TestCmdExecutor should return correct output for registered command")
}

// TestSetMockOutputAndTestCmdExecutorError checks that TestCmdExecutor returns the correct error for a registered command with an error.
func TestSetMockOutputAndTestCmdExecutorError(t *testing.T) {
	SetMockOutput("failcmd", []string{"arg"}, nil, errTest)
	cmd := TestCmdExecutor("failcmd", "arg")
	out, err := cmd.Output()
	require.ErrorIs(t, err, errTest, "TestCmdExecutor should return the error set in SetMockOutput")
	require.Nil(t, out, "TestCmdExecutor should return nil output on error")
}

// TestTestCmdExecutorNoMockOutput checks that TestCmdExecutor returns an error if no mock output is set.
func TestTestCmdExecutorNoMockOutput(t *testing.T) {
	cmd := TestCmdExecutor("notset", "arg1")
	out, err := cmd.Output()
	require.Error(t, err, "TestCmdExecutor should return error for unset command")
	require.Contains(t, err.Error(), "no mock output for notset arg1", "Error message should mention missing mock output")
	require.Nil(t, out, "Output should be nil if no mock output is set")
}

// TestMockCommandSetStderr checks that SetStderr sets the stderr writer and Output writes error to it.
func TestMockCommandSetStderr(t *testing.T) {
	mc := &mockCommand{output: []byte("ok"), err: errTest}
	var sb strings.Builder
	mc.SetStderr(&sb)
	_, err := mc.Output()
	require.ErrorIs(t, err, errTest, "Output should return errTest when set")
	require.Contains(t, sb.String(), errTest.Error(), "SetStderr should cause Output to write error to stderr")
}

// TestClearMockOutputs checks that ClearMockOutputs removes all registered mocks.
func TestClearMockOutputs(t *testing.T) {
	SetMockOutput("echo", []string{"foo"}, []byte("bar"), nil)
	ClearMockOutputs()
	cmd := TestCmdExecutor("echo", "foo")
	_, err := cmd.Output()
	require.Error(t, err, "TestCmdExecutor should return error after ClearMockOutputs")
	require.Contains(t, err.Error(), "no mock output for echo foo", "Error message should mention missing mock output after ClearMockOutputs")
}

// errTest is a helper error for testing.
var errTest = &testError{"test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
