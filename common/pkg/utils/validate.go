// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import "net/url"

func NoNullByteString(s string) bool {
	for _, b := range []byte(s) {
		if b == 0 {
			return false
		}
	}
	return true
}

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
