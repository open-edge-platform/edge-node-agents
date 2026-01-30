// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

//go:embed client-auth.sh
var authScript []byte

// CreateTempScript creates a temporary script file with the given content.
func CreateTempScript(scriptContent []byte) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "client-auth.sh")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	if _, err := tmpfile.Write(scriptContent); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("error writing to temporary file: %w", err)
	}

	if err := tmpfile.Chmod(0700); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("error setting permissions on temporary file: %w", err)
	}

	return tmpfile, nil
}

// ExecuteAuthScript executes the embedded client-auth.sh script for TTY-based authentication.
// The script prompts the user for Keycloak credentials via TTY devices.
func ExecuteAuthScript(ctx context.Context) error {
	tmpfile, err := CreateTempScript(authScript)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()

	cmd := exec.CommandContext(ctx, "/bin/sh", tmpfile.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				fmt.Printf("STDERR:\n%s\n", string(exitErr.Stderr))
			}
			return fmt.Errorf("error executing command: %w", err)
		}
		fmt.Println("client-auth.sh executed successfully")
		return nil
	case <-ctx.Done():
		fmt.Println("client-auth.sh timed out, killing process group...")
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			fmt.Printf("Failed to kill process group: %v\n", err)
		}
		return fmt.Errorf("client-auth.sh timed out: %w", ctx.Err())
	}
}
