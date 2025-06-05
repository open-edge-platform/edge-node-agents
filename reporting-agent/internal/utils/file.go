// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
)

// ReadFileTrimmed reads a file and returns its content as a string with whitespace trimmed.
func ReadFileTrimmed(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return TrimSpaceInBytes(data), nil
}
