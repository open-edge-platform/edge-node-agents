// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
)

var log = logger.Logger()

func NewExecutor[C any](createCmdFn func(name string, args ...string) *C, execCmdFn func(*C) (out []byte, e error)) Executor {
	return &executor[C]{
		createExecutableCommand: createCmdFn,
		commandExecutor:         execCmdFn,
	}
}

type Executor interface {
	Execute(args []string) ([]byte, error)
}

type executor[C any] struct {
	createExecutableCommand func(name string, args ...string) *C
	commandExecutor         func(*C) (stdout []byte, err error)
}

func (i *executor[C]) Execute(args []string) ([]byte, error) {
	executableCommand := i.createExecutableCommand(args[0], args[1:]...)
	return i.commandExecutor(executableCommand)
}

// ExecuteAndReadOutput executes a command in the operating system and returns the output,
//
//	if return status != 0, then it will return with error message
//
// It returns the output from stdout and error if exists.
func ExecuteAndReadOutput(executableCommand *exec.Cmd) (stdout []byte, err error) {
	var errbuf strings.Builder

	executableCommand.Stderr = &errbuf
	out, err := executableCommand.Output()
	log.Debugf("'%v' output - %v", executableCommand.String(), string(out))
	if err != nil {
		return nil, fmt.Errorf("failed to run '%v' command - %v; %v", executableCommand.String(), errbuf.String(), err)
	}

	return out, nil
}

func IsSymlink(filePath string) error {

	fileInfo, err := os.Lstat(filePath)

	if err != nil {
		return fmt.Errorf("lstat command failed: %v", err)
	}

	if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		return fmt.Errorf("loading metadata failed- %v is a symlink", filePath)
	}
	return nil
}
