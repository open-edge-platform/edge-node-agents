// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseSizeToMBSuccess tests ParseSizeToMB for valid MB, GB, and TB inputs.
func TestParseSizeToMBSuccess(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"1024 MB", 1024},
		{"16 GB", 16 * 1024},
		{"2 TB", 2 * 1024 * 1024},
		{"  8 GB  ", 8 * 1024},
		{"1 MB", 1},
		{"1 TB", 1024 * 1024},
	}
	for _, tc := range tests {
		got, err := ParseSizeToMB(tc.input)
		require.NoError(t, err, "ParseSizeToMB should not return error for valid input: %q", tc.input)
		require.Equal(t, tc.expected, got, "ParseSizeToMB should return correct value for input: %q", tc.input)
	}
}

// TestParseSizeToMBInvalidFormat tests ParseSizeToMB for various invalid formats.
func TestParseSizeToMBInvalidFormat(t *testing.T) {
	invalidInputs := []string{
		"", "GB", "123", "12 XB", "12MBGB", "12 MB GB", "MB 12", "12", "12  ", "MB", "12.5 GB",
	}
	for _, input := range invalidInputs {
		_, err := ParseSizeToMB(input)
		require.Error(t, err, "ParseSizeToMB should return error for invalid format: %q", input)
		require.Contains(t, err.Error(), "invalid size format", "Error message should mention invalid size format for input: %q", input)
	}
}

// TestParseSizeToMBParseUintError tests ParseSizeToMB for parse errors (number too large for uint64).
func TestParseSizeToMBParseUintError(t *testing.T) {
	// Use a number greater than uint64 max
	_, err := ParseSizeToMB("18446744073709551616 GB")
	require.Error(t, err, "ParseSizeToMB should return error for number too large for uint64")
	var numErr *strconv.NumError
	require.ErrorAs(t, err, &numErr, "Error should be of type *strconv.NumError")
}
