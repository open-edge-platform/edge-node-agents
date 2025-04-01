// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFromCommandSuccess(t *testing.T) {
	output, err := ExecuteAndReadOutput(exec.Command("true"))
	assert.NoError(t, err)
	assert.Equal(t, "", string(output))
}

func TestReadFromCommandSuccessWithOutput(t *testing.T) {
	expected := "This is expected output"
	output, err := ExecuteAndReadOutput(exec.Command("echo", "-n", expected))
	assert.NoError(t, err)
	assert.Equal(t, expected, string(output))
}

func TestReadFromCommandFailure(t *testing.T) {
	output, err := ExecuteAndReadOutput(exec.Command("false"))
	assert.Error(t, err)
	assert.NotEmpty(t, err.Error())
	assert.Empty(t, string(output))
}

func Test_executor_Execute(t *testing.T) {

	type testCommand struct {
		args []string
	}
	commandFactory := func(name string, args ...string) *testCommand {
		tc := testCommand{
			append([]string{name}, args...),
		}
		return &tc
	}

	var interceptedCommand *testCommand

	commandExecutor := func(command *testCommand) (out []byte, e error) {
		interceptedCommand = command
		return []byte("out"), nil
	}

	executor := NewExecutor[testCommand](commandFactory, commandExecutor)

	sut := executor.Execute

	out, err := sut([]string{"foo", "bar"})
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "out", string(out))
	assert.NotNil(t, interceptedCommand)
	assert.NotNil(t, interceptedCommand.args)
	assert.Equal(t, []string{"foo", "bar"}, interceptedCommand.args)
}

func TestIsSymlink(t *testing.T) {

	symlinkTarget, err := os.CreateTemp("/tmp", "symLinkTarget")
	assert.NoError(t, err)
	defer symlinkTarget.Close()

	symlink := fmt.Sprintf("%s-symlink", symlinkTarget.Name())
	err = os.Symlink(symlinkTarget.Name(), symlink)
	assert.NoError(t, err)

	assert.NoError(t, IsSymlink(symlinkTarget.Name()))
	assert.Error(t, IsSymlink(symlink))

}
