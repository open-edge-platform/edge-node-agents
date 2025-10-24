// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
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

// ExecuteAMTDeactivate executes the AMT deactivate command and verifies success.
func (r *RealCommandExecutor) ExecuteAMTDeactivate() ([]byte, error) {
	// Check if agent is being run with the DM manager mock and skip command if so
	_, testCheck := os.LookupEnv("PMA_BINARY_PATH")
	if !testCheck {
		// Execute deactivation command
		cmd := exec.Command("sudo", "/usr/bin/rpc", "deactivate", "-local")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		log.Logger.Infof("AMT deactivate command output: %s", outputStr)
		if err != nil {
			log.Logger.Errorf("Failed to execute AMT deactivate command: %v, Output: %s", err, outputStr)
			return output, fmt.Errorf("failed to execute AMT deactivate command: %w", err)
		}
		// Check if deactivation was successful based on output message
		if strings.Contains(outputStr, `msg="Status: Device deactivated"`) {
			log.Logger.Infof("Deactivation command successful - device deactivated")
			// Verify by checking RAS Remote Status is "not connected"
			return r.verifyDeactivationStatus(output)
		}
		// Handle cases where deactivation might fail initially
		if strings.Contains(outputStr, "Deactivation failed") ||
			strings.Contains(outputStr, "pre-provisioning state") ||
			strings.Contains(outputStr, "UnableToDeactivate") {
			log.Logger.Warnf("Deactivation initially failed, checking current RAS status: %s", outputStr)
			// Sometimes deactivation fails but device might already be in correct state
			// Check RAS status directly
			return r.verifyDeactivationStatus(output)
		}
		// Unknown output, but still try to verify status
		log.Logger.Warnf("Deactivation output unknown, verifying RAS status: %s", outputStr)
		return r.verifyDeactivationStatus(output)

	} else {
		// Test mode
		output := []byte("msg=\"Status: Device deactivated\"")
		return output, nil
	}
}

// verifyDeactivationStatus checks if RAS Remote Status is "not connected"
func (r *RealCommandExecutor) verifyDeactivationStatus(originalOutput []byte) ([]byte, error) {
	// Get current AMT status
	infoCmd := exec.Command("sudo", "/usr/bin/rpc", "amtinfo")
	infoOutput, infoErr := infoCmd.CombinedOutput()
	if infoErr != nil {
		log.Logger.Infof("Failed to verify RAS status after deactivation: %v", infoErr)
		// Return original output
		return originalOutput, nil
	}

	// Parse RAS Remote Status from AMT info output
	rasStatus := ""
	lines := strings.Split(string(infoOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "RAS Remote Status") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				rasStatus = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	if rasStatus == "" {
		log.Logger.Infof("Could not parse RAS Remote Status from AMT info")
		return originalOutput, nil
	}

	log.Logger.Infof("Current RAS Remote Status after deactivation: %s", rasStatus)

	if strings.ToLower(strings.TrimSpace(rasStatus)) == "not connected" {
		log.Logger.Infof("Deactivation verified successful - RAS Remote Status is 'not connected'")
		return originalOutput, nil
	}

	// RAS status is not "not connected" - deactivation not successful
	log.Logger.Warnf("Deactivation verification failed - RAS Remote Status is still '%s'", rasStatus)
	return originalOutput, fmt.Errorf("deactivation failed - RAS status is '%s', expected 'not connected'", rasStatus)
}
