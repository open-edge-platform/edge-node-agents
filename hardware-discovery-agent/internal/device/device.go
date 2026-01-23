// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"encoding/json"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

type RASInfo struct {
	NetworkStatus string `json:"networkStatus"`
	RemoteStatus  string `json:"remoteStatus"`
	RemoteTrigger string `json:"remoteTrigger"`
	MPSHostname   string `json:"mpsHostname"`
}

type AMTInfo struct {
	Version     string  `json:"version"`
	BuildNumber string  `json:"buildNumber"`
	Sku         string  `json:"sku"`
	Features    string  `json:"features"`
	Uuid        string  `json:"uuid"`
	ControlMode string  `json:"controlMode"`
	DNSSuffix   string  `json:"dnsSuffix"`
	RAS         RASInfo `json:"ras"`
}

func GetDeviceInfo(executor utils.CmdExecutor) (AMTInfo, error) {
	var amtInfo AMTInfo
	dataBytes, err := utils.ReadFromCommand(executor, "sudo", "rpc", "amtinfo", "-json")
	if err != nil {
		return amtInfo, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	err = json.Unmarshal(dataBytes, &amtInfo)
	if err != nil {
		return amtInfo, fmt.Errorf("failed to parse data from command; error: %w", err)
	}

	return amtInfo, nil
}
