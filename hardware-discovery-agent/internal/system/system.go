// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package system

import (
	"fmt"
	"regexp"
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
	ID       string
	Version  string
	Metadata []*OsMetadata
}

type Os struct {
	Kernel  *OsKern
	Release *OsRel
}

var datePatterns = []string{
	"^(0[1-9]|1[012])([/]|[-])(0[1-9]|[12][0-9]|3[01])([/]|[-])(19|20)\\d\\d$",
	"^(19|20)\\d\\d([/]|[-])(0[1-9]|[12][0-9]|3[01])([/]|[-])(0[1-9]|1[012])$",
	"^(0[1-9]|[12][0-9]|3[01])([/]|[-])(0[1-9]|1[012])([/]|[-])(19|20)\\d\\d$",
	"^(19|20)\\d\\d([/]|[-])(0[1-9]|1[012])([/]|[-])(0[1-9]|[12][0-9]|3[01])$",
}

func parseBiosDate(releaseDate []byte) string {
	parsedDate := ""
	for patternCount := 0; patternCount < len(datePatterns); patternCount++ {
		pattern := regexp.MustCompile(datePatterns[patternCount])
		if pattern.Match(releaseDate) {
			releaseDateStr := string(releaseDate)
			var dateData []string
			var reorderedDate []string
			if strings.Contains(releaseDateStr, "/") {
				dateData = strings.Split(releaseDateStr, "/")
			} else if strings.Contains(releaseDateStr, "-") {
				dateData = strings.Split(releaseDateStr, "-")
			}
			switch patternCount {
			case 0:
				for loop := 0; loop < len(dateData); loop++ {
					reorderedDate = append(reorderedDate, dateData[loop])
				}
				parsedDate = strings.Join(reorderedDate, "/")
			case 1:
				for loop := len(dateData) - 1; loop >= 0; loop-- {
					reorderedDate = append(reorderedDate, dateData[loop])
				}
				parsedDate = strings.Join(reorderedDate, "/")
			case 2:
				for loop := len(dateData) - 2; loop >= 0; loop-- {
					reorderedDate = append(reorderedDate, dateData[loop])
				}
				reorderedDate = append(reorderedDate, dateData[len(dateData)-1])
				parsedDate = strings.Join(reorderedDate, "/")
			case 3:
				for loop := 1; loop < len(dateData); loop++ {
					reorderedDate = append(reorderedDate, dateData[loop])
				}
				reorderedDate = append(reorderedDate, dateData[0])
				parsedDate = strings.Join(reorderedDate, "/")
			}
			patternCount = len(datePatterns)
		}
	}
	return parsedDate
}

func GetBiosInfo(executor utils.CmdExecutor) (*Bios, error) {
	version, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-version")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get BIOS version: %w", err)
	}

	releaseDate, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-release-date")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get Bios release date: %w", err)
	}
	parsedReleaseDate := parseBiosDate(releaseDate)

	vendor, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "bios-vendor")
	if err != nil {
		return &Bios{}, fmt.Errorf("failed to get Bios vendor: %w", err)
	}

	biosInfo := Bios{
		Version: strings.TrimSpace(string(version)),
		RelDate: strings.TrimSpace(parsedReleaseDate),
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
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
	}
	osKernel.Version = string(osKernelVersion)

	osKernelConfig := []*OsMetadata{}
	osHwPlatform, err := utils.ReadFromCommand(executor, "uname", "-i")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
	}
	kernelHwConfig := OsMetadata{
		Key:   "Platform",
		Value: string(osHwPlatform),
	}
	osKernelConfig = append(osKernelConfig, &kernelHwConfig)

	osOperatingSystem, err := utils.ReadFromCommand(executor, "uname", "-o")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
	}
	kernelOperatingSystemConfig := OsMetadata{
		Key:   "Operating System",
		Value: string(osOperatingSystem),
	}
	osKernelConfig = append(osKernelConfig, &kernelOperatingSystemConfig)
	osKernel.Config = osKernelConfig

	// Get OS release info
	osID, err := utils.ReadFromCommand(executor, "lsb_release", "-i")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
	}
	osIDInfo := strings.Split(string(osID), ":")
	osRelease.ID = strings.TrimSpace(osIDInfo[1])

	osVersion, err := utils.ReadFromCommand(executor, "lsb_release", "-d")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
	}
	osVersionInfo := strings.Split(string(osVersion), ":")
	osRelease.Version = strings.TrimSpace(osVersionInfo[1])

	osRelMetadataList := []*OsMetadata{}
	osCodeName, err := utils.ReadFromCommand(executor, "lsb_release", "-c")
	if err != nil {
		return &Os{}, fmt.Errorf("failed to get OS information: %w", err)
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
		return "", fmt.Errorf("failed to get product name: %w", err)
	}
	return strings.TrimSpace(string(productName)), nil
}

func GetSerialNumber(executor utils.CmdExecutor) (string, error) {
	serialNumber, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-serial-number")
	if err != nil {
		return "", fmt.Errorf("failed to get serial number: %w", err)
	}
	return strings.TrimSpace(string(serialNumber)), nil
}

func GetSystemUUID(executor utils.CmdExecutor) (string, error) {
	uuid, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-uuid")
	if err != nil {
		return "", fmt.Errorf("failed to get uuid: %w", err)
	}
	return strings.TrimSpace(string(uuid)), nil
}
