/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockCmd is a simple mock implementation of an executable command
type mockCmd struct {
	name string
	args []string
}

// Test helper functions
func createMockCmd(name string, args ...string) *mockCmd {
	return &mockCmd{name: name, args: args}
}

func executeMockCmd(_ *mockCmd) ([]byte, []byte, error) {
	// For most tests, we don't need to actually call the mock
	// Just return some default values
	return []byte("mock stdout"), []byte("mock stderr"), nil
}

func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with provided functions", func(t *testing.T) {
		executor := NewExecutor(createMockCmd, executeMockCmd)
		assert.NotNil(t, executor)
	})
}

func TestIsAllowedCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "reboot command is allowed",
			command:  RebootCmd,
			expected: true,
		},
		{
			name:     "shutdown command is allowed",
			command:  ShutdownCmd,
			expected: true,
		},
		{
			name:     "truncate command is allowed",
			command:  TruncateCmd,
			expected: true,
		},
		{
			name:     "os update tool command is allowed",
			command:  OsUpdateToolCmd,
			expected: true,
		},
		{
			name:     "gpg command is allowed",
			command:  GPGCmd,
			expected: true,
		},
		{
			name:     "ip command is allowed",
			command:  IPCmd,
			expected: true,
		},
		{
			name:     "snapper command is allowed",
			command:  SnapperCmd,
			expected: true,
		},
		{
			name:     "apt-get command is allowed",
			command:  AptGetCmd,
			expected: true,
		},
		{
			name:     "dpkg command is allowed",
			command:  DpkgCmd,
			expected: true,
		},
		{
			name:     "unknown command is not allowed",
			command:  "/bin/rm",
			expected: false,
		},
		{
			name:     "malicious command is not allowed",
			command:  "/usr/bin/curl",
			expected: false,
		},
		{
			name:     "empty command is not allowed",
			command:  "",
			expected: false,
		},
		{
			name:     "relative path command is not allowed",
			command:  "ls",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedCommand(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutor_Execute(t *testing.T) {
	t.Run("successful execution with allowed command", func(t *testing.T) {
		executor := NewExecutor(createMockCmd, func(cmd *mockCmd) ([]byte, []byte, error) {
			return []byte("stdout output"), []byte("stderr output"), nil
		})

		stdout, stderr, err := executor.Execute([]string{RebootCmd, "arg1", "arg2"})

		assert.NoError(t, err)
		assert.Equal(t, []byte("stdout output"), stdout)
		assert.Equal(t, []byte("stderr output"), stderr)
	})

	t.Run("empty arguments should fail", func(t *testing.T) {
		executor := NewExecutor(createMockCmd, executeMockCmd)

		stdout, stderr, err := executor.Execute([]string{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command '' is not allowed")
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
	})

	t.Run("disallowed command should fail", func(t *testing.T) {
		executor := NewExecutor(createMockCmd, executeMockCmd)

		stdout, stderr, err := executor.Execute([]string{"/bin/rm", "-rf", "/"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command '/bin/rm' is not allowed")
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
	})

	t.Run("command execution error should be propagated", func(t *testing.T) {
		executor := NewExecutor(createMockCmd, func(cmd *mockCmd) ([]byte, []byte, error) {
			return []byte("stdout"), []byte("stderr"), errors.New("execution failed")
		})

		stdout, stderr, err := executor.Execute([]string{DpkgCmd, "--configure", "-a"})

		assert.Error(t, err)
		assert.Equal(t, "execution failed", err.Error())
		assert.Equal(t, []byte("stdout"), stdout)
		assert.Equal(t, []byte("stderr"), stderr)
	})

	t.Run("command with multiple arguments", func(t *testing.T) {
		var capturedName string
		var capturedArgs []string

		createCmdFunc := func(name string, args ...string) *mockCmd {
			capturedName = name
			capturedArgs = args
			return &mockCmd{}
		}

		executor := NewExecutor(createCmdFunc, func(cmd *mockCmd) ([]byte, []byte, error) {
			return []byte("success"), []byte(""), nil
		})

		stdout, stderr, err := executor.Execute([]string{AptGetCmd, "update", "-y", "--force"})

		assert.NoError(t, err)
		assert.Equal(t, AptGetCmd, capturedName)
		assert.Equal(t, []string{"update", "-y", "--force"}, capturedArgs)
		assert.Equal(t, []byte("success"), stdout)
		assert.Equal(t, []byte(""), stderr)
	})

	t.Run("command with no additional arguments", func(t *testing.T) {
		var capturedName string
		var capturedArgs []string

		createCmdFunc := func(name string, args ...string) *mockCmd {
			capturedName = name
			capturedArgs = args
			return &mockCmd{}
		}

		executor := NewExecutor(createCmdFunc, func(cmd *mockCmd) ([]byte, []byte, error) {
			return []byte("reboot output"), []byte(""), nil
		})

		stdout, stderr, err := executor.Execute([]string{RebootCmd})

		assert.NoError(t, err)
		assert.Equal(t, RebootCmd, capturedName)
		assert.Equal(t, []string{}, capturedArgs)
		assert.Equal(t, []byte("reboot output"), stdout)
		assert.Equal(t, []byte(""), stderr)
	})
}

func TestExecuteAndReadOutput(t *testing.T) {
	t.Run("successful command execution", func(t *testing.T) {
		// Create a command that will succeed
		cmd := exec.Command("echo", "hello world")

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.NoError(t, err)
		assert.Equal(t, "hello world\n", string(stdout))
		assert.Equal(t, "", string(stderr))
	})

	t.Run("command with stderr output", func(t *testing.T) {
		// Create a command that writes to stderr
		cmd := exec.Command("sh", "-c", "echo 'error message' >&2")

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.NoError(t, err)
		assert.Equal(t, "", string(stdout))
		assert.Equal(t, "error message\n", string(stderr))
	})

	t.Run("command execution failure", func(t *testing.T) {
		// Create a command that will fail
		cmd := exec.Command("false") // 'false' command always exits with status 1

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
		assert.Contains(t, err.Error(), "false")
		assert.Equal(t, "", string(stdout))
		assert.Equal(t, "", string(stderr))
	})

	t.Run("nonexistent command", func(t *testing.T) {
		// Create a command that doesn't exist
		cmd := exec.Command("nonexistent-command-12345")

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
		assert.Equal(t, "", string(stdout))
		assert.Equal(t, "", string(stderr))
	})

	t.Run("command with both stdout and stderr", func(t *testing.T) {
		// Create a command that writes to both stdout and stderr
		cmd := exec.Command("sh", "-c", "echo 'stdout message'; echo 'stderr message' >&2")

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.NoError(t, err)
		assert.Equal(t, "stdout message\n", string(stdout))
		assert.Equal(t, "stderr message\n", string(stderr))
	})

	t.Run("command with exit code but output", func(t *testing.T) {
		// Create a command that exits with non-zero but produces output
		cmd := exec.Command("sh", "-c", "echo 'some output'; echo 'some error' >&2; exit 1")

		stdout, stderr, err := ExecuteAndReadOutput(cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
		assert.Equal(t, "some output\n", string(stdout))
		assert.Equal(t, "some error\n", string(stderr))
	})
}

// Integration test to verify the entire flow works together
func TestExecutorIntegration(t *testing.T) {
	t.Run("real command execution through executor", func(t *testing.T) {
		// Create executor with real exec.Cmd functions
		createCmdFunc := func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		}

		execCmdFunc := func(cmd *exec.Cmd) ([]byte, []byte, error) {
			return ExecuteAndReadOutput(cmd)
		}

		executor := NewExecutor(createCmdFunc, execCmdFunc)

		// Test with IP command (assuming it's available)
		stdout, _, err := executor.Execute([]string{IPCmd, "--version"})

		// The command should either succeed or fail, but it should be allowed
		if err != nil {
			// If IP command fails, it should be due to execution, not permission
			assert.NotContains(t, err.Error(), "is not allowed")
		} else {
			assert.NotEmpty(t, stdout)
		}
	})

	t.Run("disallowed command through executor", func(t *testing.T) {
		createCmdFunc := func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		}

		execCmdFunc := func(cmd *exec.Cmd) ([]byte, []byte, error) {
			return ExecuteAndReadOutput(cmd)
		}

		executor := NewExecutor(createCmdFunc, execCmdFunc)

		// Test with disallowed command
		stdout, stderr, err := executor.Execute([]string{"curl", "http://example.com"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command 'curl' is not allowed")
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
	})
}

// Benchmark tests for performance validation
func BenchmarkIsAllowedCommand(b *testing.B) {
	b.Run("allowed command", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isAllowedCommand(RebootCmd)
		}
	})

	b.Run("disallowed command", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isAllowedCommand("/bin/rm")
		}
	})
}

func BenchmarkExecutorExecute(b *testing.B) {
	executor := NewExecutor(createMockCmd, func(cmd *mockCmd) ([]byte, []byte, error) {
		return []byte("output"), []byte(""), nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = executor.Execute([]string{RebootCmd, "arg1"})
	}
}
