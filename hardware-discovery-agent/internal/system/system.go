// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package system

import (
	"fmt"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

type Bios struct {
	Version string
	RelDate string
	Vendor  string
}

type OsMetadata struct {
	Key   string
	Value string
}

type OsKern struct {
	Version string
	Config  []*OsMetadata
}

type OsRel struct {
	Id       string
	Version  string
	Metadata []*OsMetadata
}

type Os struct {
	Kernel  *OsKern
	Release *OsRel
}

func GetBiosInfo(executor utils.CmdExecutor) (*Bios, error) {
	version, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-version")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get BIOS version: %v", err)
	}

	releaseDate, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-release-date")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get Bios release date: %v", err)
	}

	vendor, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-vendor")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get Bios vendor: %v", err)
	}

	biosInfo := Bios{
		Version: strings.TrimSpace(string(version)),
		RelDate: strings.TrimSpace(string(releaseDate)),
		Vendor:  strings.TrimSpace(string(vendor)),
	}

	return &biosInfo, nil
}

func GetOsInfo(executor utils.CmdExecutor) (*Os, error) {
	var osKernel OsKern
	var osRelease OsRel

	// Get OS kernel info
	osKernelVersion, err := utils.ReadFromCommand(executor, "uname", "-r")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	osKernel.Version = string(osKernelVersion)

	osKernelConfig := []*OsMetadata{}
	osHwPlatform, err := utils.ReadFromCommand(executor, "uname", "-i")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	kernelHwConfig := OsMetadata{
		Key:   "Platform",
		Value: string(osHwPlatform),
	}
	osKernelConfig = append(osKernelConfig, &kernelHwConfig)

	osOperatingSystem, err := utils.ReadFromCommand(executor, "uname", "-o")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	kernelOperatingSystemConfig := OsMetadata{
		Key:   "Operating System",
		Value: string(osOperatingSystem),
	}
	osKernelConfig = append(osKernelConfig, &kernelOperatingSystemConfig)
	osKernel.Config = osKernelConfig

	// Get OS release info
	osId, err := utils.ReadFromCommand(executor, "lsb_release", "-i")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	osIdInfo := strings.Split(string(osId), ":")
	osRelease.Id = strings.TrimSpace(osIdInfo[1])

	osVersion, err := utils.ReadFromCommand(executor, "lsb_release", "-d")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	osVersionInfo := strings.Split(string(osVersion), ":")
	osRelease.Version = strings.TrimSpace(osVersionInfo[1])

	osRelMetadataList := []*OsMetadata{}
	osCodeName, err := utils.ReadFromCommand(executor, "lsb_release", "-c")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %v", err)
	}
	osCodeNameInfo := strings.Split(string(osCodeName), ":")
	osRelMetadata := OsMetadata{
		Key:   strings.TrimSpace(osCodeNameInfo[0]),
		Value: strings.TrimSpace(osCodeNameInfo[1]),
	}
	osRelMetadataList = append(osRelMetadataList, &osRelMetadata)
	osRelease.Metadata = osRelMetadataList

	osInfo := Os{
		Kernel:  &osKernel,
		Release: &osRelease,
	}

	return &osInfo, nil
}

func GetProductName(executor utils.CmdExecutor) (string, error) {
	productName, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-product-name")
	if err != nil {
		return "", fmt.Errorf("failed to get product name: %v", err)
	}
	return strings.TrimSpace(string(productName)), nil
}

func GetSerialNumber(executor utils.CmdExecutor) (string, error) {
	serialNumber, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-serial-number")
	if err != nil {
		return "", fmt.Errorf("failed to get serial number: %v", err)
	}
	return strings.TrimSpace(string(serialNumber)), nil
}

func GetSystemUUID(executor utils.CmdExecutor) (string, error) {
	uuid, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-uuid")
	if err != nil {
		return "", fmt.Errorf("failed to get uuid: %v", err)
	}
	return strings.TrimSpace(string(uuid)), nil
}
