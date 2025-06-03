// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"fmt"
	"strings"
	"sync"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

var (
	mockOutputMap   = map[outputMapKey]outputMapValue{}
	mockOutputMapMu sync.RWMutex
)

type outputMapKey struct {
	command string
	args    string
}

type outputMapValue struct {
	output []byte
	err    error
}

type mockCommand struct {
	output []byte
	err    error
	stderr *strings.Builder
}

// Output returns the mocked output and error for the command.
func (m *mockCommand) Output() ([]byte, error) {
	if m.stderr != nil && m.err != nil {
		m.stderr.WriteString(m.err.Error())
	}
	return m.output, m.err
}

// SetStderr sets the standard error output for the mock command to a provided strings.Builder.
func (m *mockCommand) SetStderr(stderr *strings.Builder) {
	m.stderr = stderr
}

// SetMockOutput sets the mocked output and error for a command with its arguments.
func SetMockOutput(command string, args []string, output []byte, err error) {
	mockOutputMapMu.Lock()
	defer mockOutputMapMu.Unlock()

	key := outputMapKey{command: command, args: strings.Join(args, " ")}
	mockOutputMap[key] = outputMapValue{
		output: output,
		err:    err,
	}
}

// ClearMockOutputs clears all the mocked outputs stored in the map.
func ClearMockOutputs() {
	mockOutputMapMu.Lock()
	defer mockOutputMapMu.Unlock()

	mockOutputMap = map[outputMapKey]outputMapValue{}
}

// TestCmdExecutor returns a mock command executor that simulates command execution.
func TestCmdExecutor(command string, args ...string) utils.Command {
	mockOutputMapMu.RLock()
	defer mockOutputMapMu.RUnlock()

	key := outputMapKey{command, strings.Join(args, " ")}
	val, ok := mockOutputMap[key]
	if !ok {
		return &mockCommand{output: nil, err: fmt.Errorf("no mock output for %s %s", command, strings.Join(args, " "))}
	}
	return &mockCommand{output: val.output, err: val.err}
}
