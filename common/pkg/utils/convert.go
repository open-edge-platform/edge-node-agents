// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	sizeRegexp = regexp.MustCompile(`^(\d+)\s*(MB|GB|TB)$`)
)

// ParseSizeToMB converts a size string (e.g., "16 GB", "1024 MB", "1 TB") to MB.
func ParseSizeToMB(sizeStr string) (uint64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	matches := sizeRegexp.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	size, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	switch unit {
	case "MB":
		return size, nil
	case "GB":
		return size * 1024, nil
	case "TB":
		return size * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}
}
