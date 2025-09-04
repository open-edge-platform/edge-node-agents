/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"log"

	utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
)

// CheckNetworkConnection verifies if there is an active network connection.
func CheckNetworkConnection(cmdExecutor utils.Executor) bool {
	cmd := []string{"ip", "route", "show", "default"}

	stdout, _, err := cmdExecutor.Execute(cmd)
	if err != nil {
		log.Println("Error running command:", err)
		return false
	}
	if len(stdout) == 0 {
		log.Println("No default gateway detected in output.")
		return false
	}
	log.Println("Default gateway is present. Network connection is likely active.")
	return true
}
