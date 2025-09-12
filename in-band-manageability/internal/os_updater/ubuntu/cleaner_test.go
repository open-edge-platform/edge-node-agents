// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ubuntu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleaner_Clean(t *testing.T) {
	// Create a Cleaner instance
	cleaner := Cleaner{}

	// Call the Clean method
	err := cleaner.Clean()

	// Assertions
	assert.NoError(t, err, "Clean should not return an error")
}
