/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater provides the implementation for updating the firmware.
package fwupdater

const firmwareToolInfoFilePath = "/etc/firmware_tool_info.conf"

const firmwareToolInfoSchemaFilePath = "/usr/share/firmware_tool_config_schema.json"

// FirmwareToolInfo is the matching firmware tool information for the platform.
type FirmwareToolInfo struct {
	Name                  string `json:"name"`
	ToolOptions           bool   `json:"tool_options"`
	GUID                  bool   `json:"guid"`
	BiosVendor            string `json:"bios_vendor"`
	FirmwareTool          string `json:"firmware_tool"`
	FirmwareToolArgs      string `json:"firmware_tool_args"`
	FirmwareToolCheckArgs string `json:"firmware_tool_check_args"`
	FirmwareFileType      string `json:"firmware_file_type"`
}
