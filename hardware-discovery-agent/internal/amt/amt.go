// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package amt

import (
	"encoding/json"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/system"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

type RASInfo struct {
	NetworkStatus string `json:"networkStatus"`
	RemoteStatus  string `json:"remoteStatus"`
	RemoteTrigger string `json:"remoteTrigger"`
	MPSHostname   string `json:"mpsHostname"`
}

type AmtInfo struct {
	Version          string   `json:"amt"`
	DeviceName       string   `json:"hostnameOS"`
	OperationalState string   `json:"operationalState"`
	BuildNumber      string   `json:"buildNumber"`
	Sku              string   `json:"sku"`
	Features         string   `json:"features"`
	Uuid             string   `json:"uuid"`
	ControlMode      string   `json:"controlMode"`
	DNSSuffix        string   `json:"dnsSuffix"`
	RAS              *RASInfo `json:"ras"`
}

func GetAmtInfo(executor utils.CmdExecutor) (*AmtInfo, error) {
	var amtInfo AmtInfo
	dataBytes, err := utils.ReadFromCommand(executor, "sudo", "rpc", "amtinfo", "-json")
	if err != nil {
		return &AmtInfo{}, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	err = json.Unmarshal(dataBytes, &amtInfo)
	if err != nil {
		return &AmtInfo{}, fmt.Errorf("failed to parse data from command; error: %w", err)
	}

	if amtInfo.Uuid == "" {
		systemId, err := system.GetSystemUUID(executor)
		if err != nil {
			return &AmtInfo{}, fmt.Errorf("failed to retrieve system uuid; error: %w", err)
		}
		amtInfo.Uuid = systemId
	}

	return &amtInfo, nil
}
