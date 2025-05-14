// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package gpu

import (
	"fmt"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/utils"
)

type Gpu struct {
	PciId       string
	Product     string
	Vendor      string
	Name        string
	Description string
	Features    []string
}

var infoNotFound string = "Not Available"

func parseGpuDetails(gpuDevice, infoType string) []string {
	gpuInfo := strings.Split(gpuDevice, infoType)
	if len(gpuInfo) == 1 {
		// Return empty string if infoType is not found
		return []string{infoNotFound}
	}
	return strings.Split(gpuInfo[1], "\n")
}

func GetGpuList(executor utils.CmdExecutor) ([]*Gpu, error) {
	gpuInfo, err := utils.ReadFromCommand(executor, "sudo", "lshw", "-C", "display")
	if err != nil {
		return []*Gpu{}, fmt.Errorf("failed to read data from command; error: %v", err)
	}

	gpuDevices := strings.Split(string(gpuInfo), "*-display")
	gpuList := []*Gpu{}
	for _, gpu := range gpuDevices {
		if strings.TrimSpace(gpu) == "" {
			continue
		}

		pciInfo := parseGpuDetails(gpu, "bus info: ")
		pciAddress := strings.TrimPrefix(pciInfo[0], "pci@0000:")

		productName := parseGpuDetails(gpu, "product: ")
		vendor := parseGpuDetails(gpu, "vendor: ")
		description := parseGpuDetails(gpu, "description: ")

		deviceFeatures := parseGpuDetails(gpu, "capabilities: ")
		features := deviceFeatures
		if features[0] != infoNotFound {
			features = strings.Split(deviceFeatures[0], " ")
		}

		device := "Info Not Available"
		if pciAddress != infoNotFound {
			deviceInfo, err := utils.ReadFromCommand(executor, "lspci", "-v", "-s", pciAddress)
			if err != nil {
				return []*Gpu{}, fmt.Errorf("failed to read data from command; error: %v", err)
			}
			deviceName := strings.Split(string(deviceInfo), " (")
			devName := strings.Split(deviceName[0], ": ")
			if len(devName) > 1 {
				device = devName[1]
			}
		}

		gpuList = append(gpuList, &Gpu{
			PciId:       pciAddress,
			Product:     productName[0],
			Vendor:      vendor[0],
			Name:        device,
			Description: description[0],
			Features:    features,
		})
	}

	return gpuList, nil
}
