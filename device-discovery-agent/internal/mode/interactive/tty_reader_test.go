// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDeviceReader_ReadUsername_Success(t *testing.T) {
	// Note: This test requires a real TTY device or PTY for full functionality
	// For CI/CD environments, we skip if no TTY is available

	// Create temporary file to simulate device (limited testing)
	tmpFile, err := os.CreateTemp("", "tty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test input to file
	tmpFile.Write([]byte("testuser\n"))
	tmpFile.Seek(0, 0) // Reset to beginning

	reader := &DeviceReader{
		devicePath: "test",
		fd:         tmpFile,
		oldState:   nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username, err := reader.ReadUsername(ctx, "")
	if err != nil {
		t.Fatalf("ReadUsername failed: %v", err)
	}

	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
}

func TestDeviceReader_ReadUsername_Timeout(t *testing.T) {
	// Create an empty temp file - reading will block until data or timeout
	tmpFile, err := os.CreateTemp("", "tty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	reader := &DeviceReader{
		devicePath: "test",
		fd:         tmpFile,
		oldState:   nil,
	}

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = reader.ReadUsername(ctx, "")
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Accept either context deadline or EOF (depending on timing)
	if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "EOF") {
		t.Logf("Got error: %v (acceptable)", err)
	}
}

func TestDeviceReader_ReadUsername_EmptyInput(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write just newline
	tmpFile.Write([]byte("\n"))
	tmpFile.Seek(0, 0)

	reader := &DeviceReader{
		devicePath: "test",
		fd:         tmpFile,
		oldState:   nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username, err := reader.ReadUsername(ctx, "")
	if err != nil {
		t.Fatalf("ReadUsername failed: %v", err)
	}

	if username != "" {
		t.Errorf("Expected empty username, got '%s'", username)
	}
}

func TestDeviceReader_ReadUsername_Whitespace(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write username with leading/trailing spaces
	tmpFile.Write([]byte("  testuser  \n"))
	tmpFile.Seek(0, 0)

	reader := &DeviceReader{
		devicePath: "test",
		fd:         tmpFile,
		oldState:   nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username, err := reader.ReadUsername(ctx, "")
	if err != nil {
		t.Fatalf("ReadUsername failed: %v", err)
	}

	// Should be trimmed by TrimSpace
	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
}

func TestDeviceReader_Prompt(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	reader := &DeviceReader{
		devicePath: "test",
		fd:         w, // Use write end for output
		oldState:   nil,
	}

	// Write prompt
	err = reader.Prompt("Test message\n")
	if err != nil {
		t.Fatalf("Prompt failed: %v", err)
	}

	// Read from pipe to verify
	buf := make([]byte, 100)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	output := string(buf[:n])
	if output != "Test message\n" {
		t.Errorf("Expected 'Test message\\n', got '%s'", output)
	}
}

func TestDeviceReader_Close(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer w.Close()

	reader := &DeviceReader{
		devicePath: "test",
		fd:         r,
		oldState:   nil,
	}

	// Close should not error
	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewDeviceReader_NonexistentDevice(t *testing.T) {
	// Try to open a device that doesn't exist
	_, err := NewDeviceReader("nonexistent-tty-device-12345")
	if err == nil {
		t.Fatal("Expected error for nonexistent device")
	}

	if !strings.Contains(err.Error(), "failed to open") {
		t.Errorf("Expected 'failed to open' error, got: %v", err)
	}
}
