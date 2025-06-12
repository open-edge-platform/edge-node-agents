// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExecCmdOutputSuccess checks that ExecCmd.Output returns the expected output for a real command.
func TestExecCmdOutputSuccess(t *testing.T) {
	cmd := &ExecCmd{cmd: exec.Command("echo", "hello")}
	out, err := cmd.Output()
	require.NoError(t, err, "ExecCmd.Output should not return error for echo")
	require.Equal(t, "hello\n", string(out), "ExecCmd.Output should return correct output")
}

// TestExecCmdOutputFailure checks that ExecCmd.Output returns an error for a non-existent command.
func TestExecCmdOutputFailure(t *testing.T) {
	cmd := &ExecCmd{cmd: exec.Command("nonexistent_command_xyz")}
	_, err := cmd.Output()
	require.Error(t, err, "ExecCmd.Output should return error for invalid command")
}

// TestExecCmdSetStderr sets a strings.Builder as stderr and checks that it is set on the exec.Cmd.
func TestExecCmdSetStderr(t *testing.T) {
	cmd := &ExecCmd{cmd: exec.Command("echo", "foo")}
	var sb strings.Builder
	cmd.SetStderr(&sb)
	require.Equal(t, &sb, cmd.cmd.Stderr, "ExecCmd.SetStderr should set the Stderr field")
}

// TestExecCmdExecutor returns an ExecCmd and checks its type and command.
func TestExecCmdExecutor(t *testing.T) {
	c := ExecCmdExecutor("echo", "foo", "bar")
	execCmd, ok := c.(*ExecCmd)
	require.True(t, ok, "ExecCmdExecutor should return *ExecCmd")
	require.Contains(t, execCmd.cmd.Path, "echo", "ExecCmdExecutor should set correct command path containing 'echo'")
	require.Equal(t, []string{"echo", "foo", "bar"}, execCmd.cmd.Args, "ExecCmdExecutor should set correct args")
}

// TestReadFromCommandSuccess checks that ReadFromCommand returns output and no error on success.
func TestReadFromCommandSuccess(t *testing.T) {
	executor := mockCmdExecutor([]byte("ok"), nil, nil)
	out, err := ReadFromCommand(executor, "foo", "bar")
	require.NoError(t, err, "ReadFromCommand should not return error on success")
	require.Equal(t, []byte("ok"), out, "ReadFromCommand should return correct output")
}

// TestReadFromCommandError checks that ReadFromCommand returns error and writes to stderr on failure,
// and covers the code path where ReadFromCommand itself calls SetStderr and Output.
func TestReadFromCommandError(t *testing.T) {
	// This mock will append to errBuf inside ReadFromCommand, so we can check both error and errBuf content.
	executor := func(string, ...string) Command {
		return &mockCommandForRead{
			output: nil,
			err:    mockError("fail123"),
			// m.stderr will be set by SetStderr inside ReadFromCommand
		}
	}
	out, err := ReadFromCommand(executor, "foo", "bar")
	require.Error(t, err, "ReadFromCommand should return error on command failure with stderr")
	require.Nil(t, out, "ReadFromCommand should return nil output on error")
	require.Contains(t, err.Error(), "fail123", "ReadFromCommand error should contain stderr output")
}

// mockCmdExecutor is a helper for mocking command execution in ReadFromCommand tests.
func mockCmdExecutor(output []byte, err error, stderr *strings.Builder) CmdExecutor {
	return func(string, ...string) Command {
		return &mockCommandForRead{
			output: output,
			err:    err,
			stderr: stderr,
		}
	}
}

// mockError is a helper to create errors with a specific message.
func mockError(msg string) error {
	return &customError{msg: msg}
}

type customError struct {
	msg string
}

func (e *customError) Error() string { return e.msg }

// mockCommandForRead implements Command for ReadFromCommand tests.
type mockCommandForRead struct {
	output []byte
	err    error
	stderr *strings.Builder
}

func (m *mockCommandForRead) Output() ([]byte, error) {
	if m.stderr != nil && m.err != nil {
		m.stderr.WriteString(m.err.Error())
	}
	return m.output, m.err
}
func (m *mockCommandForRead) SetStderr(sb *strings.Builder) {
	m.stderr = sb
}
