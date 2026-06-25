// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

func DetectOS(reader FileReader, forcedOS string) (string, error) {
	if forcedOS != "" {
		switch forcedOS {
		case "ubuntu":
			return "ubuntu", nil
		case "emt":
			return "emt", nil
		case "debian":
			return "debian", nil
		default:
			return "", fmt.Errorf("unsupported forced OS: %s", forcedOS)
		}
	}

	content, err := reader.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to open /etc/os-release: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var osId string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			osId = strings.Trim(strings.TrimPrefix(line, "ID="), `"'`)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read /etc/os-release: %v", err)
	}

	switch osId {
	case "ubuntu":
		return "ubuntu", nil
	case "Edge Microvisor Toolkit":
		return "emt", nil
	case "debian":
		return "debian", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", osId)
	}
}
