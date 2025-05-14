// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

type CmdExecutor = func(name string, args ...string) *exec.Cmd

// ReadFromCommand executes a command in the operating system and returns the output,
//
//	if return status != 0, then it will return with error message
//
// It returns the output from stdout and error if exist.
func ReadFromCommand(executor CmdExecutor, command string, args ...string) (stdout []byte, err error) {
	var errbuf strings.Builder

	cmd := executor(command, args...)
	cmd.Stderr = &errbuf
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%v; %v", errbuf.String(), err)
	}

	return out, nil
}
