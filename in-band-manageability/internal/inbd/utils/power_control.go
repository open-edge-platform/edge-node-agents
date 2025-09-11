/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"fmt"
	"time"

	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
)

// RebootSystem reboots the system using the provided command executor.
func RebootSystem(cmdExecutor common.Executor) error {
	fmt.Println("Rebooting ")

	time.Sleep(2 * time.Second)

	_, _, err := cmdExecutor.Execute([]string{common.RebootCmd})
	if err != nil {
		return fmt.Errorf("reboot failed: %s", err)
	}

	return nil
}

// ShutdownSystem shuts down the system using the provided command executor.
func ShutdownSystem(cmdExecutor common.Executor) error {
	fmt.Print("Shutting down ")

	time.Sleep(2 * time.Second)

	_, _, err := cmdExecutor.Execute([]string{common.ShutdownCmd, "now"}) // Shutdown immediately
	if err != nil {
		return fmt.Errorf("shutdown failed: %s", err)
	}

	return nil
}
