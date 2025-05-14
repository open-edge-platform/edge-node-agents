// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"encoding/json"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/utils"
)

type memory struct {
	Size  int    `json:"size"`
	State string `json:"state"`
}

type memories struct {
	Memory []memory `json:"memory"`
}

func GetMemory(executor utils.CmdExecutor) (uint64, error) {
	var dataStruct memories
	var total uint64

	dataBytes, err := utils.ReadFromCommand(executor, "lsmem", "-J", "-b")
	if err != nil {
		return 0, fmt.Errorf("failed to read data from command; error: %v", err)
	}

	err = json.Unmarshal(dataBytes, &dataStruct)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal data; error: %v", err)
	}

	for _, mem := range dataStruct.Memory {
		if mem.State == "online" {
			total += uint64(mem.Size)
		}
	}

	return total, nil
}
