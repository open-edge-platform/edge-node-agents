// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cpu

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

// GetCPUData collects CPU information from `lscpu` and processes the result to generate structured CPU data.
func GetCPUData(executor utils.CmdExecutor) (model.CPU, error) {
	cpu := model.CPU{}

	outputBytes, err := utils.ReadFromCommand(executor, "lscpu")
	if err != nil {
		return cpu, fmt.Errorf("failed to read data from lscpu command: %w", err)
	}

	lines := strings.Split(string(outputBytes), "\n")
	for _, line := range lines {
		attr := utils.TrimSpace(line)
		if strings.HasPrefix(attr, "Architecture:") {
			cpu.Architecture = utils.TrimPrefix(attr, "Architecture:")
		}
		if strings.HasPrefix(attr, "Vendor ID:") {
			cpu.Vendor = utils.TrimPrefix(attr, "Vendor ID:")
		}
		if strings.HasPrefix(attr, "CPU family:") {
			cpu.Family = utils.TrimPrefix(attr, "CPU family:")
		}
		if strings.HasPrefix(attr, "Model name:") {
			cpu.ModelName = utils.TrimPrefix(attr, "Model name:")
		}
		if strings.HasPrefix(attr, "Model:") {
			cpu.Model = utils.TrimPrefix(attr, "Model:")
		}
		if strings.HasPrefix(attr, "Stepping:") {
			cpu.Stepping = utils.TrimPrefix(attr, "Stepping:")
		}
		if strings.HasPrefix(attr, "Socket(s):") {
			socketStr := utils.TrimPrefix(attr, "Socket(s):")
			sockets, err := strconv.ParseUint(socketStr, 10, 64)
			if err != nil {
				continue
			}
			cpu.SocketCount = sockets
		}
		if strings.HasPrefix(attr, "CPU(s):") {
			threadsStr := utils.TrimPrefix(attr, "CPU(s):")
			threads, err := strconv.ParseUint(threadsStr, 10, 64)
			if err != nil {
				continue
			}
			cpu.ThreadCount = threads
		}
		if strings.HasPrefix(attr, "Core(s) per socket:") {
			coresStr := utils.TrimPrefix(attr, "Core(s) per socket:")
			cores, err := strconv.ParseUint(coresStr, 10, 64)
			if err != nil {
				continue
			}
			// will be multiplied with socket count to get total core count later
			cpu.CoreCount = cores
		}
		if strings.HasPrefix(attr, "Virtualization type:") {
			cpu.Virtualization = utils.TrimPrefix(attr, "Virtualization type:")
		}
		if strings.HasPrefix(attr, "Hypervisor vendor:") {
			cpu.Hypervisor = utils.TrimPrefix(attr, "Hypervisor vendor:")
		}
	}

	cpu.CoreCount = cpu.SocketCount * cpu.CoreCount

	return cpu, nil
}
