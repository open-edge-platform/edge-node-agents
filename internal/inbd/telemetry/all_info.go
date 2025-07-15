/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

// GetAllInfo retrieves all system information
func GetAllInfo() (*pb.AllInfo, error) {
	// Get hardware info
	hardware, err := GetHardwareInfo()
	if err != nil {
		return nil, err
	}

	// Get firmware info
	firmware, err := GetFirmwareInfo()
	if err != nil {
		return nil, err
	}

	// Get OS info
	osInfo, err := GetOSInfo()
	if err != nil {
		return nil, err
	}

	// Get version info
	version, err := GetVersionInfo()
	if err != nil {
		return nil, err
	}

	// Get power capabilities
	powerCapabilities, err := GetPowerCapabilities()
	if err != nil {
		return nil, err
	}

	// Get SWBOM info
	swbom, err := GetSoftwareBOM()
	if err != nil {
		return nil, err
	}

	// Build AllInfo response
	allInfo := &pb.AllInfo{
		Hardware:          hardware,
		Firmware:          firmware,
		OsInfo:            osInfo,
		Version:           version,
		PowerCapabilities: powerCapabilities,
		Swbom:             swbom,
		AdditionalInfo:    []string{}, // Additional info if needed
	}

	return allInfo, nil
}
