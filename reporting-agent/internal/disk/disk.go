// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package disk

import (
	"encoding/json"
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

type disks struct {
	Disk []disk `json:"blockdevices"`
}

type disk struct {
	Name     string `json:"kname"`
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Size     uint64 `json:"size"` // in bytes
	Type     string `json:"type"`
	Children []disk `json:"children"`
}

// GetDiskData retrieves disk information from the system using the lsblk command.
func GetDiskData(executor utils.CmdExecutor) ([]model.Disk, error) {
	diskList := []model.Disk{} //nolint:prealloc // Number of disks on node unknown before run time so cannot preallocate to specific size
	var diskStruct disks

	outputBytes, err := utils.ReadFromCommand(executor, "lsblk", "-o", "KNAME,VENDOR,MODEL,SIZE,TYPE", "-J", "-b", "--tree")
	if err != nil {
		return diskList, fmt.Errorf("failed to read data from lsblk command: %w", err)
	}

	err = json.Unmarshal(outputBytes, &diskStruct)
	if err != nil {
		return diskList, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	for _, disk := range diskStruct.Disk {
		if disk.Type != "disk" {
			continue
		}
		diskData := model.Disk{
			Name:          disk.Name,
			Vendor:        disk.Vendor,
			Model:         disk.Model,
			Size:          disk.Size,
			ChildrenCount: len(disk.Children),
		}

		diskList = append(diskList, diskData)
	}
	return diskList, nil
}
