/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"fmt"
	"log"
)

const dispatcherStatePath = "/var/intel-manageability/dispatcher_state"

// ClearStateFile clears the dispatcher state file by truncating it to zero size.
func ClearStateFile(cmdExecutor Executor) error {
	log.Println("Clearing dispatcher state file.")
	
	// Clear the dispatcher state file before writing it.
	// We use truncate rather than remove here as some OSes like EMT require files that need to persist
	// between reboots to not be removed.
	dispatcherStateTruncateCommand := []string{
		"truncate", "-s", "0", dispatcherStatePath,
	}

	if _, _, err := cmdExecutor.Execute(dispatcherStateTruncateCommand); err != nil {
		return fmt.Errorf("failed to truncate dispatcher state file with command(%v)- %w", dispatcherStateTruncateCommand, err)
	}
	return nil
}