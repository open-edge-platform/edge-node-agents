// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import "net/url"

// NoNullByteString checks if the provided string does not contain any null bytes.
func NoNullByteString(s string) bool {
	for _, b := range []byte(s) {
		if b == 0 {
			return false
		}
	}
	return true
}

// NoNullByteURL checks if the provided URL string does not contain any null bytes.
func NoNullByteURL(u string) bool {
	s, err := url.PathUnescape(u)
	if err != nil {
		return false
	}
	if !NoNullByteString(s) {
		return false
	}
	return true
}
