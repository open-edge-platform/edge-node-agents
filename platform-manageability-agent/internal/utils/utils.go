// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
)

func ExecuteWithRetries(command string, args []string) ([]byte, error) {
	maxRetries := 3
	retryInterval := 5 * time.Second

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

func GetSystemUUID() (string, error) {
	cmd := exec.Command("sudo", "dmidecode", "-s", "system-uuid")
	uuid, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve UUID: %v", err)
	}
	return strings.TrimSpace(string(uuid)), nil
}
