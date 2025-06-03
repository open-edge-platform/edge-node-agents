// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"fmt"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

func GetMemoryData(executor utils.CmdExecutor) (model.Memory, error) {
	memory := model.Memory{}
	memory.Devices = []model.MemoryDevice{}

	outputBytes, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "--type", "memory")
	if err != nil {
		return memory, fmt.Errorf("failed to read data from dmidecode command: %w", err)
	}

	totalSizeMB := uint64(0)
	var commonType, commonFormFactor string
	firstDevice := true

	entries := strings.Split(string(outputBytes), "\n\n") // Split by blank lines to separate memory devices
	for _, entry := range entries {
		if !strings.Contains(entry, "Memory Device") {
			continue
		}

		device := model.MemoryDevice{}
		lines := strings.Split(entry, "\n")
		for _, line := range lines {
			line = utils.TrimSpace(line)
			if strings.HasPrefix(line, "Form Factor:") {
				device.FormFactor = utils.TrimPrefix(line, "Form Factor:")
			} else if strings.HasPrefix(line, "Size:") {
				device.Size = utils.TrimPrefix(line, "Size:")
			} else if strings.HasPrefix(line, "Type:") {
				device.Type = utils.TrimPrefix(line, "Type:")
			} else if strings.HasPrefix(line, "Speed:") {
				device.Speed = utils.TrimPrefix(line, "Speed:")
			} else if strings.HasPrefix(line, "Manufacturer:") {
				device.Manufacturer = utils.TrimPrefix(line, "Manufacturer:")
			}
		}

		if device.Size != "" && device.Size != "No Module Installed" && device.Size != "Unknown" {
			sizeMB, err := utils.ParseSizeToMB(device.Size)
			if err == nil {
				totalSizeMB += sizeMB
			}
			memory.Devices = append(memory.Devices, device)

			// Determine CommonType and CommonFormFactor
			if firstDevice {
				commonType = device.Type
				commonFormFactor = device.FormFactor
				firstDevice = false
				continue
			}

			if commonType != device.Type {
				commonType = ""
			}
			if commonFormFactor != device.FormFactor {
				commonFormFactor = ""
			}
		}
	}

	memory.Summary.TotalSizeMB = totalSizeMB
	memory.Summary.CommonType = commonType
	memory.Summary.CommonFormFactor = commonFormFactor

	return memory, nil
}
