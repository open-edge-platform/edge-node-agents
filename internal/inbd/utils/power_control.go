/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"fmt"
	"time"

	"github.com/spf13/afero"
)

// RebootSystem reboots the system using the provided command executor.
func RebootSystem(cmdExecutor Executor) error {
	fmt.Println("Rebooting ")

	// Try to clean up LUKS volume if possible
	cleanupLUKS()

	time.Sleep(2 * time.Second)

	_, _, err := cmdExecutor.Execute([]string{RebootCmd})
	if err != nil {
		return fmt.Errorf("reboot failed: %s", err)
	}

	return nil
}

// cleanupLUKS Before Shutdown/Reboot attempts to load config and cleanup LUKS volume
func cleanupLUKS() {
	config, err := LoadConfig(afero.NewOsFs(), ConfigFilePath)
	if err != nil {
		fmt.Printf("Warning: Could not load config for LUKS cleanup: %v\n", err)
		return
	}

	if err := RemoveLUKSVolume(config); err != nil {
		fmt.Printf("Warning: Failed to remove LUKS volume before shutdown: %v\n", err)
	}
}

// ShutdownSystem shuts down the system using the provided command executor.
func ShutdownSystem(cmdExecutor Executor) error {
	fmt.Print("Shutting down ")

	// Try to clean up LUKS volume if possible
	cleanupLUKS()

	time.Sleep(2 * time.Second)

	_, _, err := cmdExecutor.Execute([]string{ShutdownCmd, "now"}) // Shutdown immediately
	if err != nil {
		return fmt.Errorf("shutdown failed: %s", err)
	}

	return nil
}
