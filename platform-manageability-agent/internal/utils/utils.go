// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os/exec"
	"time"

	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

// CommandExecutor defines an interface for executing commands.
// This allows for mocking in tests.
type CommandExecutor interface {
	ExecuteAMTInfo() ([]byte, error)
	ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error)
}

type RealCommandExecutor struct{}

func ExecuteCommand(command string, args []string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		log.Logger.Errorf("Failed to execute command %s with args %v: %v", command, args, err)
		return nil, fmt.Errorf("failed to execute command %s with args %v: %w", command, args, err)
	}
	return output, nil
}

// ExecuteAMTInfo executes the AMT info command with retries.
func (r *RealCommandExecutor) ExecuteAMTInfo() ([]byte, error) {
	maxRetries := 3
	retryInterval := 5 * time.Second

	var err error
	for i := 1; i <= maxRetries; i++ {
		cmd := exec.Command("sudo", "/usr/bin/rpc", "amtinfo")
		output, err := cmd.Output()
		if err == nil {
			return output, nil
		}
		log.Logger.Warnf("Failed to execute AMT info command (attempt %d/%d): %v", i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(retryInterval)
		}
	}
	return nil, fmt.Errorf("amtInfo command failed after %d retries: %v", maxRetries, err)
}

// ExecuteAMTActivate executes the AMT activate command.
func (r *RealCommandExecutor) ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error) {
	cmd := exec.Command("sudo", "/usr/bin/rpc", "activate", "-u", rpsAddress, "-profile", profileName, "-password", password, "-n")
	return cmd.CombinedOutput()
}
