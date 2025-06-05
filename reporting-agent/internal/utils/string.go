// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import "strings"

// TrimSpace removes leading and trailing whitespace from the string.
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// TrimSpaceInBytes converts a byte slice to a string and trims leading and trailing whitespace.
func TrimSpaceInBytes(b []byte) string {
	return TrimSpace(string(b))
}

// TrimPrefix removes the specified prefix from the string and trims any leading or trailing whitespace.
func TrimPrefix(s, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, prefix))
}
