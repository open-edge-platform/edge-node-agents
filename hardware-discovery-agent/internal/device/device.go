// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package device

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

type DeviceInfo struct {
	Version          string  `json:"amt"`
	Hostname         string  `json:"hostnameOS"`
	OperationalState string  `json:"operationalState"`
	BuildNumber      string  `json:"buildNumber"`
	Sku              string  `json:"sku"`
	Features         string  `json:"features"`
	Uuid             string  `json:"uuid"`
	ControlMode      string  `json:"controlMode"`
	DNSSuffix        string  `json:"dnsSuffix"`
	RAS              RASInfo `json:"ras"`
}

func GetDeviceInfo(executor utils.CmdExecutor) (DeviceInfo, error) {
	var deviceInfo DeviceInfo
	dataBytes, err := utils.ReadFromCommand(executor, "sudo", "rpc", "amtinfo", "-json")
	if err != nil {
		return deviceInfo, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	err = json.Unmarshal(dataBytes, &deviceInfo)
	if err != nil {
		return deviceInfo, fmt.Errorf("failed to parse data from command; error: %w", err)
	}

	if deviceInfo.Uuid == "" {
		systemId, err := system.GetSystemUUID(executor)
		if err != nil {
			return DeviceInfo{}, fmt.Errorf("failed to retrieve system uuid; error: %w", err)
		}
		deviceInfo.Uuid = systemId
	}

	return deviceInfo, nil
}
