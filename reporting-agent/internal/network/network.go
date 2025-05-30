// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

type NetworkDevices struct {
	Serial string `json:"serial"`
}

func GetNetworkSerials(executor utils.CmdExecutor) ([]string, error) {
	lshwOutput, err := utils.ReadFromCommand(executor, "sudo", "lshw", "-json", "-class", "network")
	if err != nil {
		return nil, fmt.Errorf("failed to read network devices: %w", err)
	}
	serials := make([]NetworkDevices, 0)
	// Double check on the expected system if the output format is always an array
	// and not a dict or a single object.
	if err = json.Unmarshal(lshwOutput, &serials); err != nil {
		return nil, fmt.Errorf("unable to unmarshal network serials: %w", err)
	}

	serialStrings := make([]string, 0, len(serials))
	for _, d := range serials {
		if d.Serial != "" {
			serialStrings = append(serialStrings, d.Serial)
		}
	}
	if len(serialStrings) == 0 {
		return nil, errors.New("no network serials found")
	}

	return serialStrings, nil
}
