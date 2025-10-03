// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"time"

	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

// CommandExecutor defines an interface for executing commands.
// This allows for mocking in tests.
type CommandExecutor interface {
	ExecuteAMTInfo() ([]byte, error)
	ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error)
	ExecuteAMTDeactivate() ([]byte, error)
}

type RealCommandExecutor struct{}

func ExecuteCommand(command string, args []string) ([]byte, error) {
	// Check if agent is being run with the DM manager mock and skip command if so
	_, testCheck := os.LookupEnv("PMA_BINARY_PATH")
	if !testCheck {
		cmd := exec.Command(command, args...)
		output, err := cmd.Output()
		if err != nil {
			log.Logger.Errorf("Failed to execute command %s with args %v: %v", command, args, err)
			return nil, fmt.Errorf("failed to execute command %s with args %v: %w", command, args, err)
		}
		return output, nil
	} else {
		if slices.Contains(args, "is-active") {
			output := []byte("active")
			return output, nil
		} else {
			output := []byte("success")
			return output, nil
		}
	}
}

// ExecuteAMTInfo executes the AMT info command with retries.
func (r *RealCommandExecutor) ExecuteAMTInfo() ([]byte, error) {
	maxRetries := 3
	retryInterval := 5 * time.Second
	var output []byte

	var err error
	for i := 1; i <= maxRetries; i++ {
		// Check if agent is being run with the DM manager mock and skip command if so
		_, testCheck := os.LookupEnv("PMA_BINARY_PATH")
		if !testCheck {
			cmd := exec.Command("sudo", "/usr/bin/rpc", "amtinfo")
			output, err = cmd.CombinedOutput()
			if err == nil {
				return output, nil
			}
			log.Logger.Warnf("Failed to execute AMT info command (attempt %d/%d): %v", i, maxRetries, err)
			if i < maxRetries {
				time.Sleep(retryInterval)
			}
		} else {
			output = []byte("Version: 0.1.0\nRAS Remote Status: connecting")
			return output, nil
		}
	}
	return output, fmt.Errorf("amtInfo command failed after %d retries: %v", maxRetries, err)
}

// ExecuteAMTActivate executes the AMT activate command.
func (r *RealCommandExecutor) ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error) {
	cmd := exec.Command("sudo", "-E", "/usr/bin/rpc", "activate", "-u", rpsAddress, "-n")
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("AMT_PASSWORD=%s", password))
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PROFILE=%s", profileName))
	return cmd.CombinedOutput()
}

// ExecuteAMTDeactivate executes the AMT deactivate command for stuck connecting states.
func (r *RealCommandExecutor) ExecuteAMTDeactivate() ([]byte, error) {
	// Check if agent is being run with the DM manager mock and skip command if so
	_, testCheck := os.LookupEnv("PMA_BINARY_PATH")
	if !testCheck {
		cmd := exec.Command("sudo", "/usr/bin/rpc", "deactivate", "-local")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Logger.Errorf("Failed to execute AMT deactivate command: %v, Output: %s", err, string(output))
			return output, fmt.Errorf("failed to execute AMT deactivate command: %w", err)
		}
		log.Logger.Infof("AMT deactivate command executed successfully: %s", string(output))
		return output, nil
	} else {
		output := []byte("deactivation success")
		return output, nil
	}
}
