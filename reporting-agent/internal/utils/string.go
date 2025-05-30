// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import "strings"

func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func TrimSpaceInBytes(b []byte) string {
	return TrimSpace(string(b))
}

func TrimPrefix(s, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, prefix))
}
