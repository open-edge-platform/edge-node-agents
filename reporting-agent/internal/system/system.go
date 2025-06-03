// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

func GetTimezone(executor utils.CmdExecutor) (string, error) {
	timezoneBytes, err := utils.ReadFromCommand(executor, "date", "+%Z")
	if err != nil {
		return "", fmt.Errorf("failed to get timezone: %w", err)
	}
	return utils.TrimSpaceInBytes(timezoneBytes), nil
}

func GetLocaleData(executor utils.CmdExecutor) (model.Locale, error) {
	locale := model.Locale{}

	localeBytes, err := utils.ReadFromCommand(executor, "locale", "-k", "LC_ADDRESS")
	if err != nil {
		return locale, fmt.Errorf("failed to get locale data: %w", err)
	}

	lines := strings.Split(string(localeBytes), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := utils.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)

		switch key {
		case "country_name":
			locale.CountryName = value
		case "country_ab2":
			locale.CountryAbbr = value
		case "lang_name":
			locale.LangName = value
		case "lang_ab":
			locale.LangAbbr = value
		}
	}

	return locale, nil
}

func GetKernelData(executor utils.CmdExecutor) (model.Kernel, error) {
	kernel := model.Kernel{}

	osMachineHardwareName, err := utils.ReadFromCommand(executor, "uname", "-m")
	if err != nil {
		return kernel, fmt.Errorf("failed to get OS information (machine hardware name): %w", err)
	}
	kernel.Machine = utils.TrimSpaceInBytes(osMachineHardwareName)

	osKernelName, err := utils.ReadFromCommand(executor, "uname", "-s")
	if err != nil {
		return kernel, fmt.Errorf("failed to get OS information (kernel name): %w", err)
	}
	kernel.Name = utils.TrimSpaceInBytes(osKernelName)

	osKernelRelease, err := utils.ReadFromCommand(executor, "uname", "-r")
	if err != nil {
		return kernel, fmt.Errorf("failed to get OS information (kernel release): %w", err)
	}
	kernel.Release = utils.TrimSpaceInBytes(osKernelRelease)

	osKernelVersion, err := utils.ReadFromCommand(executor, "uname", "-v")
	if err != nil {
		return kernel, fmt.Errorf("failed to get OS information (kernel version): %w", err)
	}
	kernel.Version = utils.TrimSpaceInBytes(osKernelVersion)

	osOperatingSystem, err := utils.ReadFromCommand(executor, "uname", "-o")
	if err != nil {
		return kernel, fmt.Errorf("failed to get OS information (operating system): %w", err)
	}
	kernel.System = utils.TrimSpaceInBytes(osOperatingSystem)

	return kernel, nil
}

func GetReleaseData(executor utils.CmdExecutor) (model.Release, error) {
	release := model.Release{}

	osReleaseBytes, err := utils.ReadFromCommand(executor, "cat", "/etc/os-release")
	if err != nil {
		return release, fmt.Errorf("failed to read data from /etc/os-release: %w", err)
	}

	lines := strings.Split(string(osReleaseBytes), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := utils.TrimSpace(parts[0])
		value := utils.TrimSpace(strings.Trim(parts[1], `"`))

		switch key {
		case "ID":
			release.ID = value
		case "VERSION_ID":
			release.VersionID = value
		case "VERSION":
			release.Version = value
		case "VERSION_CODENAME":
			release.Codename = value
		case "ID_LIKE":
			release.Family = value
		case "BUILD_ID":
			release.BuildID = value
		case "IMAGE_ID":
			release.ImageID = value
		case "IMAGE_VERSION":
			release.ImageVersion = value
		}
	}

	return release, nil
}

func GetUptimeData(executor utils.CmdExecutor) (float64, error) {
	uptimeBytes, err := utils.ReadFromCommand(executor, "cat", "/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("failed to read data from /proc/uptime: %w", err)
	}

	uptimeStr := utils.TrimSpaceInBytes(uptimeBytes)
	parts := strings.Fields(uptimeStr)
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected format in /proc/uptime: %s", uptimeStr)
	}

	uptime, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime value: %w", err)
	}

	return uptime, nil
}

func GetSerialNumber(executor utils.CmdExecutor) (string, error) {
	serialNumber, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-serial-number")
	if err != nil {
		return "", fmt.Errorf("failed to get serial number: %w", err)
	}
	return utils.TrimSpaceInBytes(serialNumber), nil
}

func GetSystemUUID(executor utils.CmdExecutor) (string, error) {
	uuid, err := utils.ReadFromCommand(executor, "sudo", "dmidecode", "-s", "system-uuid")
	if err != nil {
		return "", fmt.Errorf("failed to get system UUID: %w", err)
	}
	return utils.TrimSpaceInBytes(uuid), nil
}
