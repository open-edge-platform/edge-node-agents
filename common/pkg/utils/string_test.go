// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTrimSpace checks that TrimSpace removes leading and trailing whitespace.
func TestTrimSpace(t *testing.T) {
	require.Equal(t, "abc", TrimSpace("  abc  "), "TrimSpace should trim leading and trailing spaces")
	require.Equal(t, "abc", TrimSpace("\tabc\n"), "TrimSpace should trim tabs and newlines")
	require.Empty(t, TrimSpace("   "), "TrimSpace should return empty string for all spaces")
	require.Equal(t, "abc def", TrimSpace("  abc def  "), "TrimSpace should preserve inner spaces")
	require.Empty(t, TrimSpace(""), "TrimSpace should return empty string for empty input")
}

// TestTrimSpaceInBytes checks that TrimSpaceInBytes trims whitespace from byte slices.
func TestTrimSpaceInBytes(t *testing.T) {
	require.Equal(t, "abc", TrimSpaceInBytes([]byte("  abc  ")), "TrimSpaceInBytes should trim spaces in byte slice")
	require.Empty(t, TrimSpaceInBytes([]byte("   ")), "TrimSpaceInBytes should return empty string for all spaces")
	require.Equal(t, "abc", TrimSpaceInBytes([]byte("abc")), "TrimSpaceInBytes should return string unchanged if no spaces")
	require.Empty(t, TrimSpaceInBytes([]byte{}), "TrimSpaceInBytes should return empty string for empty byte slice")
}

// TestTrimPrefix checks that TrimPrefix removes the prefix and trims the result.
func TestTrimPrefix(t *testing.T) {
	require.Equal(t, "bar", TrimPrefix("foo bar", "foo"), "TrimPrefix should remove prefix and trim")
	require.Equal(t, "bar", TrimPrefix("   foo bar", "   foo"), "TrimPrefix should remove prefix and trim leading spaces")
	require.Equal(t, "bar", TrimPrefix("foo bar   ", "foo"), "TrimPrefix should remove prefix and trim trailing spaces")
	require.Equal(t, "foo bar", TrimPrefix("foo bar", "baz"), "TrimPrefix should not remove non-matching prefix")
	require.Empty(t, TrimPrefix("", "foo"), "TrimPrefix should return empty string for empty input")
	require.Empty(t, TrimPrefix("   ", "foo"), "TrimPrefix should return empty string for all spaces")
}
