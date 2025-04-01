// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package disk

import (
	"encoding/json"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

type disk struct {
	Serial string `json:"serial"`
	Name   string `json:"kname"`
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
	Size   uint64 `json:"size"` //in bytes
	Type   string `json:"type"`
	Wwn    string `json:"wwn"` //world wide name may also be referred to as WWID
}

type disks struct {
	Disk []disk `json:"blockdevices"`
}

type Disk struct {
	SerialNum string
	Name      string
	Vendor    string
	Model     string
	Size      uint64
	Wwid      string
}

func GetDiskList(executor utils.CmdExecutor) ([]*Disk, error) {
	diskList := []*Disk{}
	var diskStruct disks

	dataBytes, err := utils.ReadFromCommand(executor, "lsblk", "-o", "SERIAL,KNAME,VENDOR,MODEL,SIZE,WWN,TYPE", "-J", "-b")
	if err != nil {
		return []*Disk{}, fmt.Errorf("failed to read data from command; error: %v", err)
	}

	err = json.Unmarshal(dataBytes, &diskStruct)
	if err != nil {
		return []*Disk{}, fmt.Errorf("failed to unmarshal data; error: %v", err)
	}

	for _, disk := range diskStruct.Disk {
		if disk.Type != "disk" || disk.Size == uint64(0) {
			continue
		}
		if disk.Serial == "" {
			disk.Serial = "unknown"
		}
		if disk.Vendor == "" {
			disk.Vendor = "unknown"
		}
		if disk.Model == "" {
			disk.Model = "unknown"
		}
		if disk.Wwn == "" {
			disk.Wwn = "unknown"
		}
		diskData := Disk{
			SerialNum: disk.Serial,
			Name:      disk.Name,
			Vendor:    disk.Vendor,
			Model:     disk.Model,
			Size:      disk.Size,
			Wwid:      disk.Wwn,
		}
		diskList = append(diskList, &diskData)
	}
	return diskList, nil
}
