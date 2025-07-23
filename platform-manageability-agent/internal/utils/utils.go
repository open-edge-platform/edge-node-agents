// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

var allowedCommands = map[string][]string{
	"sudo": {"./rpc", "rpc"},
}

// CommandExecutor defines an interface for executing commands.
// This allows for mocking in tests.
type CommandExecutor interface {
	ExecuteWithRetries(command string, args []string) ([]byte, error)
	ExecuteCommand(name string, args ...string) ([]byte, error)
}

type RealCommandExecutor struct{}

func (r *RealCommandExecutor) ExecuteWithRetries(command string, args []string) ([]byte, error) {
	maxRetries := 3
	retryInterval := 5 * time.Second

	if !isCommandAllowed(command, args) {
		return nil, fmt.Errorf("command not allowed: %s %v", command, args)
	}

	var err error
	for i := 1; i <= maxRetries; i++ {
		cmd := exec.Command(command, args...)
		output, err := cmd.Output()
		if err == nil {
			return output, nil
		}
		log.Logger.Warnf("Failed to execute `%s` command (attempt %d/%d): %v", command, i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(retryInterval)
		}
	}
	return nil, fmt.Errorf("command `%s` failed after %d retries: %v", command, maxRetries, err)
}

func (r *RealCommandExecutor) ExecuteCommand(name string, args ...string) ([]byte, error) {
	if !isCommandAllowed(name, args) {
		return nil, fmt.Errorf("command not allowed: %s %v", name, args)
	}
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func GetSystemUUID() (string, error) {
	cmd := exec.Command("sudo", "dmidecode", "-s", "system-uuid")
	uuid, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve UUID: %v", err)
	}
	return strings.TrimSpace(string(uuid)), nil
}

func isCommandAllowed(command string, args []string) bool {
	allowedArgs, exists := allowedCommands[command]
	if !exists {
		return false
	}

	if len(args) == 0 {
		return false
	}
	return slices.Contains(allowedArgs, args[0])
}
