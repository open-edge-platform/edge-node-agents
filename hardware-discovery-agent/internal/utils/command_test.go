// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	"os/exec"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestReadFromCommandSuccess(t *testing.T) {
	output, err := utils.ReadFromCommand(exec.Command, "true")
	assert.NoError(t, err)
	assert.Equal(t, "", string(output))
}

func TestReadFromCommandSuccessWithOutput(t *testing.T) {
	expected := "This is expected output"
	output, err := utils.ReadFromCommand(exec.Command, "echo", "-n", expected)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(output))
}

func TestReadFromCommandFailure(t *testing.T) {
	output, err := utils.ReadFromCommand(exec.Command, "false")
	assert.Error(t, err)
	assert.NotEmpty(t, err.Error())
	assert.Empty(t, string(output))
}
