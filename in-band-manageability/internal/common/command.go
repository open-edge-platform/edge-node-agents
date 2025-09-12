/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package common provides utilities used by multiple packages
package common

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// NewExecutor creates a new executor.
func NewExecutor[C any](createCmdFn func(name string, args ...string) *C, execCmdFn func(*C) ([]byte, []byte, error)) Executor {
	return &executor[C]{
		createExecutableCommand: createCmdFn,
		commandExecutor:         execCmdFn,
	}
}

// Executor is an interface that contains the method to execute a command.
type Executor interface {
	Execute([]string) (stdout []byte, stderr []byte, err error)
}

type executor[C any] struct {
	createExecutableCommand func(name string, args ...string) *C
	commandExecutor         func(*C) ([]byte, []byte, error)
}

var allowedCommands = []string{
	RebootCmd,
	ShutdownCmd,
	TruncateCmd,
	OsUpdateToolCmd,
	GPGCmd,
	IPCmd,
	SnapperCmd,
	AptGetCmd,
	DpkgCmd,
}

func isAllowedCommand(cmd string) bool {
	return slices.Contains(allowedCommands, cmd)
}

func (i *executor[C]) Execute(args []string) ([]byte, []byte, error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("command '' is not allowed")
	}
	if !isAllowedCommand(args[0]) {
		return nil, nil, fmt.Errorf("command '%s' is not allowed", args[0])
	}
	executableCommand := i.createExecutableCommand(args[0], args[1:]...)
	return i.commandExecutor(executableCommand)
}

// ExecuteAndReadOutput executes a command in the operating system and returns the output.
//
// It takes an *exec.Cmd as input, executes the command, and captures both stdout and stderr.
// If the command execution fails (non-zero exit status), it returns an error containing
// the stderr output and the error message.
//
// Parameters:
//   - executableCommand: A pointer to an exec.Cmd object representing the command to execute.
//
// Returns:
//   - stdout: A byte slice containing the standard error output of the command.
//   - stderr: A byte slice containing the standard output of the command.
//   - err: An error object if the command fails, or nil if it succeeds.
func ExecuteAndReadOutput(executableCommand *exec.Cmd) ([]byte, []byte, error) {
	var stdout, stderr strings.Builder

	executableCommand.Stdout = &stdout
	executableCommand.Stderr = &stderr

	err := executableCommand.Run()

	fmt.Printf("'%v' stderr: %v, stdout: %v\n", executableCommand.String(), stderr.String(), stdout.String())
	if err != nil {
		return []byte(stdout.String()), []byte(stderr.String()), fmt.Errorf("failed to run '%v' command - %v", executableCommand.String(), err)
	}

	return []byte(stdout.String()), []byte(stderr.String()), nil
}
