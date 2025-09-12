/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
)

// RebootMockExecutor is a specific mock implementation for reboot testing.
type RebootMockExecutor struct {
	mock.Mock
	callCount int
	lastArgs  []string
}

func (m *RebootMockExecutor) Execute(args []string) ([]byte, []byte, error) {
	m.callCount++
	m.lastArgs = args
	mockArgs := m.Called(args)
	return []byte(mockArgs.String(0)), []byte(mockArgs.String(1)), mockArgs.Error(2)
}

func TestRebootSystem_Success(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("Reboot command executed", "", nil)

	start := time.Now()
	err := RebootSystem(mockExecutor)
	elapsed := time.Since(start)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Verify the executor was called exactly once
	if mockExecutor.callCount != 1 {
		t.Errorf("Expected executor to be called once, but was called %d times", mockExecutor.callCount)
	}

	// Verify the correct command was passed
	expectedArgs := []string{common.RebootCmd}
	if len(mockExecutor.lastArgs) != len(expectedArgs) || mockExecutor.lastArgs[0] != expectedArgs[0] {
		t.Errorf("Expected args %v, but got %v", expectedArgs, mockExecutor.lastArgs)
	}

	// Verify the 2-second sleep occurred (with some tolerance for timing)
	if elapsed < 2*time.Second || elapsed > 3*time.Second {
		t.Errorf("Expected function to take approximately 2 seconds, but took %v", elapsed)
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

func TestRebootSystem_ExecutorError(t *testing.T) {
	expectedError := errors.New("command execution failed")
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("", "error output", expectedError)

	err := RebootSystem(mockExecutor)

	// Verify error is returned and wrapped correctly
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

	expectedErrorMessage := fmt.Sprintf("reboot failed: %s", expectedError)
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}

	// Verify the executor was still called
	if mockExecutor.callCount != 1 {
		t.Errorf("Expected executor to be called once, but was called %d times", mockExecutor.callCount)
	}

	// Verify the correct command was passed even when it fails
	expectedArgs := []string{common.RebootCmd}
	if len(mockExecutor.lastArgs) != len(expectedArgs) || mockExecutor.lastArgs[0] != expectedArgs[0] {
		t.Errorf("Expected args %v, but got %v", expectedArgs, mockExecutor.lastArgs)
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

func TestRebootSystem_ExecutorReturnsStdoutAndStderr(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("reboot initiated", "warning: system will restart", nil)

	err := RebootSystem(mockExecutor)

	// Should succeed even with stderr output
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Verify the executor was called
	if mockExecutor.callCount != 1 {
		t.Errorf("Expected executor to be called once, but was called %d times", mockExecutor.callCount)
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

func TestRebootSystem_VerifyRebootCommand(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("success", "", nil)

	err := RebootSystem(mockExecutor)
	assert.NoError(t, err, "Expected RebootSystem to succeed")

	// Verify the exact reboot command is used
	assert.Equal(t, 1, len(mockExecutor.lastArgs), "Expected exactly one argument")

	if mockExecutor.lastArgs[0] != "/usr/sbin/reboot" {
		t.Errorf("Expected reboot command to be '/usr/sbin/reboot', but got '%s'", mockExecutor.lastArgs[0])
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

func TestRebootSystem_NilExecutor(t *testing.T) {
	// This test verifies behavior with nil executor
	// Note: This will panic, which might be the intended behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling RebootSystem with nil executor, but no panic occurred")
		}
	}()

	err := RebootSystem(nil)
	assert.Error(t, err, "Expected error when calling RebootSystem with nil executor")
}

// Benchmark to measure performance
func BenchmarkRebootSystem(b *testing.B) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("success", "", nil)

	for b.Loop() {
		err := RebootSystem(mockExecutor)
		assert.NoError(b, err, "Expected RebootSystem to succeed")
	}
}

// Test multiple calls to ensure no state issues
func TestRebootSystem_MultipleCalls(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	// Set up the mock to expect 3 calls
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("success", "", nil).Times(3)

	// Call RebootSystem multiple times
	for i := range 3 {
		err := RebootSystem(mockExecutor)
		if err != nil {
			t.Errorf("Call %d failed with error: %v", i+1, err)
		}
	}

	// Verify all calls were made
	if mockExecutor.callCount != 3 {
		t.Errorf("Expected 3 calls to executor, but got %d", mockExecutor.callCount)
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

// Test edge case with empty command slice (should not happen in practice)
func TestRebootSystem_TimingAccuracy(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("success", "", nil)

	// Test multiple times to ensure timing is consistent
	for i := 0; i < 3; i++ {
		start := time.Now()
		err := RebootSystem(mockExecutor)
		assert.NoError(t, err, "Expected RebootSystem to succeed")
		elapsed := time.Since(start)

		// Allow some tolerance for system timing variations
		if elapsed < 1800*time.Millisecond || elapsed > 2500*time.Millisecond {
			t.Errorf("Run %d: Expected function to take approximately 2 seconds, but took %v", i+1, elapsed)
		}
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

// Test to verify that the function properly handles command formatting
func TestRebootSystem_CommandFormat(t *testing.T) {
	mockExecutor := new(RebootMockExecutor)
	mockExecutor.On("Execute", []string{common.RebootCmd}).Return("success", "", nil)

	err := RebootSystem(mockExecutor)
	assert.NoError(t, err, "Expected RebootSystem to succeed")

	// Verify command is passed as a slice with exactly one element
	assert.Equal(t, 1, len(mockExecutor.lastArgs), "Expected command slice to have exactly 1 element")

	// Verify the command string is exactly the RebootCmd constant
	if mockExecutor.lastArgs[0] != common.RebootCmd {
		t.Errorf("Expected command to be '%s', got '%s'", common.RebootCmd, mockExecutor.lastArgs[0])
	}

	// Verify mock expectations
	mockExecutor.AssertExpectations(t)
}

// TestRebootSystem_ErrorMessage tests error message formatting for various error types
func TestRebootSystem_ErrorMessage(t *testing.T) {
	testCases := []struct {
		name          string
		executorError error
		expectedMsg   string
	}{
		{
			name:          "simple error",
			executorError: errors.New("command not found"),
			expectedMsg:   "reboot failed: command not found",
		},
		{
			name:          "permission error",
			executorError: errors.New("permission denied"),
			expectedMsg:   "reboot failed: permission denied",
		},
		{
			name:          "complex error",
			executorError: errors.New("exit status 1: reboot: unable to restart system"),
			expectedMsg:   "reboot failed: exit status 1: reboot: unable to restart system",
		},
		{
			name:          "systemd error",
			executorError: errors.New("Failed to reboot system via logind: Interactive authentication required"),
			expectedMsg:   "reboot failed: Failed to reboot system via logind: Interactive authentication required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExecutor := &RebootMockExecutor{}
			mockExecutor.On("Execute", []string{common.RebootCmd}).Return("", "", tc.executorError)

			err := RebootSystem(mockExecutor)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if err.Error() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, err.Error())
			}

			mockExecutor.AssertCalled(t, "Execute", []string{common.RebootCmd})
		})
	}
}

// TestShutdownSystem_Success tests successful shutdown execution
func TestShutdownSystem_Success(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("Shutdown command executed", "", nil)

	err := ShutdownSystem(mockExecutor)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}

// TestShutdownSystem_Failure tests shutdown execution failure
func TestShutdownSystem_Failure(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	expectedError := errors.New("shutdown command failed")
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("", "error output", expectedError)

	err := ShutdownSystem(mockExecutor)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	expectedErrorMsg := "shutdown failed: shutdown command failed"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}

	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}

// TestShutdownSystem_TimingAccuracy tests that ShutdownSystem respects the 2-second delay
func TestShutdownSystem_TimingAccuracy(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("success", "", nil)

	start := time.Now()
	err := ShutdownSystem(mockExecutor)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the 2-second sleep occurred (with some tolerance for timing)
	if elapsed < 2*time.Second || elapsed > 3*time.Second {
		t.Errorf("Expected function to take approximately 2 seconds, but took %v", elapsed)
	}

	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}

// TestShutdownSystem_CommandFormat tests the exact command format being executed
func TestShutdownSystem_CommandFormat(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("success", "", nil)

	err := ShutdownSystem(mockExecutor)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the exact command format
	expectedArgs := []string{common.ShutdownCmd, "now"}
	if len(mockExecutor.lastArgs) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d", len(expectedArgs), len(mockExecutor.lastArgs))
	}

	for i, expected := range expectedArgs {
		if i >= len(mockExecutor.lastArgs) || mockExecutor.lastArgs[i] != expected {
			t.Errorf("Expected arg[%d] = '%s', got '%s'", i, expected, mockExecutor.lastArgs[i])
		}
	}

	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}

// TestShutdownSystem_ExecutorReturnsStdoutAndStderr tests handling of command output
func TestShutdownSystem_ExecutorReturnsStdoutAndStderr(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("shutdown initiated", "warning: system will power off", nil)

	err := ShutdownSystem(mockExecutor)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the executor was called
	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}

// TestShutdownSystem_NilExecutor tests behavior with nil executor
func TestShutdownSystem_NilExecutor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when calling ShutdownSystem with nil executor")
		}
	}()

	// This should panic since we're calling a method on nil
	err := ShutdownSystem(nil)

	// If we reach here, the test failed
	t.Errorf("Expected panic, but got error: %v", err)
}

// TestShutdownSystem_MultipleCalls tests multiple consecutive calls
func TestShutdownSystem_MultipleCalls(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("success", "", nil).Times(3)

	// Call shutdown multiple times
	for i := 0; i < 3; i++ {
		err := ShutdownSystem(mockExecutor)
		if err != nil {
			t.Errorf("Call %d: Expected no error, got %v", i+1, err)
		}
	}

	// Verify all calls were made
	if mockExecutor.callCount != 3 {
		t.Errorf("Expected 3 calls, but got %d", mockExecutor.callCount)
	}

	mockExecutor.AssertExpectations(t)
}

// TestShutdownSystem_ErrorMessage tests error message formatting
func TestShutdownSystem_ErrorMessage(t *testing.T) {
	testCases := []struct {
		name          string
		executorError error
		expectedMsg   string
	}{
		{
			name:          "simple error",
			executorError: errors.New("command not found"),
			expectedMsg:   "shutdown failed: command not found",
		},
		{
			name:          "permission error",
			executorError: errors.New("permission denied"),
			expectedMsg:   "shutdown failed: permission denied",
		},
		{
			name:          "complex error",
			executorError: errors.New("exit status 1: shutdown: unable to shutdown system"),
			expectedMsg:   "shutdown failed: exit status 1: shutdown: unable to shutdown system",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExecutor := &RebootMockExecutor{}
			mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("", "", tc.executorError)

			err := ShutdownSystem(mockExecutor)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if err.Error() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, err.Error())
			}

			mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
		})
	}
}

// TestShutdownSystem_VerifyShutdownCommand tests that the correct shutdown command constant is used
func TestShutdownSystem_VerifyShutdownCommand(t *testing.T) {
	mockExecutor := &RebootMockExecutor{}
	mockExecutor.On("Execute", []string{common.ShutdownCmd, "now"}).Return("success", "", nil)

	err := ShutdownSystem(mockExecutor)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the first argument is the correct shutdown command
	if len(mockExecutor.lastArgs) == 0 || mockExecutor.lastArgs[0] != common.ShutdownCmd {
		t.Errorf("Expected first argument to be '%s', got '%s'", common.ShutdownCmd, mockExecutor.lastArgs[0])
	}

	// Verify the second argument is "now"
	if len(mockExecutor.lastArgs) < 2 || mockExecutor.lastArgs[1] != "now" {
		t.Errorf("Expected second argument to be 'now', got '%s'", mockExecutor.lastArgs[1])
	}

	mockExecutor.AssertCalled(t, "Execute", []string{common.ShutdownCmd, "now"})
}
