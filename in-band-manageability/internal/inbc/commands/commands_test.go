/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"fmt"
	"testing"
)

func TestMust_NoError(t *testing.T) {
	// Test must function with no error
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("must panicked unexpectedly: %v", r)
		}
	}()
	must(nil) // Should not panic
}

func TestMust_WithError(t *testing.T) {
	// Test must function with an error
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("must did not panic when it should have")
		}
	}()
	must(fmt.Errorf("test error")) // Should panic
}
