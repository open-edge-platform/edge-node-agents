// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// DeviceReader handles reading from a single TTY device.
type DeviceReader struct {
	devicePath string
	fd         *os.File
	oldState   *term.State
}

// NewDeviceReader opens a TTY device for reading.
// The devicePath should be the device name without /dev/ prefix (e.g., "ttyS0", "tty0").
func NewDeviceReader(devicePath string) (*DeviceReader, error) {
	fullPath := "/dev/" + devicePath

	// Open the TTY device with read/write access
	fd, err := os.OpenFile(fullPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", fullPath, err)
	}

	return &DeviceReader{
		devicePath: devicePath,
		fd:         fd,
		oldState:   nil,
	}, nil
}

// ReadUsername reads visible username input from the TTY.
// It displays the prompt and waits for the user to enter a line of text.
func (d *DeviceReader) ReadUsername(ctx context.Context, prompt string) (string, error) {
	// Write the prompt to the TTY
	if _, err := fmt.Fprintf(d.fd, "%s", prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt: %w", err)
	}

	// Create a channel to receive the result
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		// Read a line from the TTY
		reader := bufio.NewReader(d.fd)
		line, err := reader.ReadString('\n')
		if err != nil {
			errChan <- fmt.Errorf("failed to read username: %w", err)
			return
		}
		resultChan <- strings.TrimSpace(line)
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errChan:
		return "", err
	case username := <-resultChan:
		return username, nil
	}
}

// ReadPassword reads masked password input from the TTY using golang.org/x/term.
// The password is not echoed to the terminal.
func (d *DeviceReader) ReadPassword(ctx context.Context, prompt string) (string, error) {
	// Write the prompt to the TTY
	if _, err := fmt.Fprintf(d.fd, "%s", prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt: %w", err)
	}

	// Save the current terminal state
	fd := int(d.fd.Fd())
	oldState, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("failed to get terminal state: %w", err)
	}
	d.oldState = oldState

	// Create a channel to receive the result
	resultChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		// Read password (this will block until Enter is pressed)
		password, err := term.ReadPassword(fd)
		if err != nil {
			errChan <- fmt.Errorf("failed to read password: %w", err)
			return
		}
		// Write a newline after password input (since ReadPassword doesn't echo)
		fmt.Fprintf(d.fd, "\n")
		resultChan <- password
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		// Restore terminal state on cancellation
		if d.oldState != nil {
			term.Restore(fd, d.oldState)
		}
		return "", ctx.Err()
	case err := <-errChan:
		// Restore terminal state on error
		if d.oldState != nil {
			term.Restore(fd, d.oldState)
		}
		return "", err
	case password := <-resultChan:
		// Restore terminal state after successful read
		if d.oldState != nil {
			term.Restore(fd, d.oldState)
		}
		return string(password), nil
	}
}

// Prompt writes a message to the TTY without waiting for input.
func (d *DeviceReader) Prompt(message string) error {
	if _, err := fmt.Fprintf(d.fd, "%s", message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}

// Close restores the TTY state (if modified) and closes the file descriptor.
func (d *DeviceReader) Close() error {
	// Restore terminal state if it was saved
	if d.oldState != nil {
		fd := int(d.fd.Fd())
		if err := term.Restore(fd, d.oldState); err != nil {
			// Log the error but continue with closing
			fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal state for %s: %v\n", d.devicePath, err)
		}
	}

	// Close the file descriptor
	if d.fd != nil {
		return d.fd.Close()
	}
	return nil
}
