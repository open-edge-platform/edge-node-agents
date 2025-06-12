// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// Command interface defines the methods required for executing a command and capturing its output.
type Command interface {
	// Output executes the command and returns its standard output as a byte slice.
	Output() ([]byte, error)
	// SetStderr sets the standard error output for the command to a provided strings.Builder.
	SetStderr(stderr *strings.Builder)
}

// CmdExecutor is a function type that takes a command name and its arguments, and returns a Command interface instance that can execute the command.
type CmdExecutor = func(name string, args ...string) Command

// ExecCmd is a struct that implements the Command interface using the exec package to run commands.
type ExecCmd struct {
	cmd *exec.Cmd
}

// Output executes the command and returns its standard output as a byte slice.
func (e *ExecCmd) Output() ([]byte, error) {
	return e.cmd.Output()
}

// SetStderr sets the standard error output for the command to a provided strings.Builder.
func (e *ExecCmd) SetStderr(stderr *strings.Builder) {
	e.cmd.Stderr = stderr
}

// ExecCmdExecutor is a function that creates an ExecCmd instance for executing commands.
func ExecCmdExecutor(name string, args ...string) Command {
	return &ExecCmd{cmd: exec.Command(name, args...)}
}

// ReadFromCommand executes a command in the operating system and returns the output,
// It returns the output from stdout and error if exists.
func ReadFromCommand(executor CmdExecutor, command string, args ...string) (stdout []byte, err error) {
	var errBuf strings.Builder

	cmd := executor(command, args...)
	cmd.SetStderr(&errBuf)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%v: %w", errBuf.String(), err)
	}

	return out, nil
}
