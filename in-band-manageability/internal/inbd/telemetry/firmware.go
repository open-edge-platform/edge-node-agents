/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DMI paths for firmware information
const (
	DMI_BIOS_VENDOR          = "/sys/class/dmi/id/bios_vendor"
	DMI_BIOS_VERSION         = "/sys/class/dmi/id/bios_version"
	DMI_BIOS_DATE            = "/sys/class/dmi/id/bios_date"
	DEVICE_TREE_FW_PATH      = "/proc/device-tree/firmware/bios/"
	DEVICE_TREE_BIOS_VENDOR  = DEVICE_TREE_FW_PATH + "bios-vendor"
	DEVICE_TREE_BIOS_VERSION = DEVICE_TREE_FW_PATH + "bios-version"
	DEVICE_TREE_BIOS_DATE    = DEVICE_TREE_FW_PATH + "bios-release-date"
)

// GetFirmwareInfo retrieves firmware/BIOS information from the system
func GetFirmwareInfo() (*pb.FirmwareInfo, error) {
	fw := &pb.FirmwareInfo{}

	if runtime.GOOS != "linux" {
		return fw, fmt.Errorf("firmware information only supported on Linux")
	}

	// Try device tree first (for ARM systems)
	if isDeviceTreeAvailable() {
		getFirmwareFromDeviceTree(fw)
		return fw, nil
	}

	// Fall back to DMI (for x86 systems)
	getFirmwareFromDMI(fw)

	return fw, nil
}

// isDeviceTreeAvailable checks if device tree firmware info is available
func isDeviceTreeAvailable() bool {
	_, err := os.Stat(DEVICE_TREE_FW_PATH)
	return err == nil
}

// getFirmwareFromDeviceTree reads firmware info from device tree
func getFirmwareFromDeviceTree(fw *pb.FirmwareInfo) {
	// Get BIOS vendor
	if vendor, err := readFileContent(DEVICE_TREE_BIOS_VENDOR); err == nil {
		fw.BiosVendor = strings.TrimSpace(vendor)
	}

	// Get BIOS version
	if version, err := readFileContent(DEVICE_TREE_BIOS_VERSION); err == nil {
		fw.BiosVersion = strings.TrimSpace(version)
	}

	// Get BIOS release date
	if dateStr, err := readFileContent(DEVICE_TREE_BIOS_DATE); err == nil {
		if parsedDate, err := parseBiosDate(strings.TrimSpace(dateStr)); err == nil {
			fw.BiosReleaseDate = parsedDate
		}
	}
}

// getFirmwareFromDMI reads firmware info from DMI
func getFirmwareFromDMI(fw *pb.FirmwareInfo) {

	// Get BIOS vendor
	if vendor, err := readFileContent(DMI_BIOS_VENDOR); err == nil {
		fw.BiosVendor = strings.TrimSpace(vendor)
	}
	// Get BIOS version
	if version, err := readFileContent(DMI_BIOS_VERSION); err == nil {
		fw.BiosVersion = strings.TrimSpace(version)
	}

	// Get BIOS release date
	if dateStr, err := readFileContent(DMI_BIOS_DATE); err == nil {
		if parsedDate, err := parseBiosDate(strings.TrimSpace(dateStr)); err == nil {
			fw.BiosReleaseDate = parsedDate
		}
	}
}

// readFileContent reads and returns the content of a file
func readFileContent(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Remove null bytes and trim whitespace
	content := strings.ReplaceAll(string(data), "\x00", "")
	return strings.TrimSpace(content), nil
}

// parseBiosDate parses BIOS date string into protobuf timestamp
func parseBiosDate(dateStr string) (*timestamppb.Timestamp, error) {
	if dateStr == "" {
		return nil, fmt.Errorf("empty date string")
	}

	// Common BIOS date formats
	formats := []string{
		"01/02/2006",      // MM/DD/YYYY
		"01/02/06",        // MM/DD/YY
		"2006-01-02",      // YYYY-MM-DD
		"02/01/2006",      // DD/MM/YYYY
		"2006/01/02",      // YYYY/MM/DD
		"Jan 2, 2006",     // Mon DD, YYYY
		"January 2, 2006", // Month DD, YYYY
		"02 Jan 2006",     // DD Mon YYYY
		"2006.01.02",      // YYYY.MM.DD
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, dateStr); err == nil {
			return timestamppb.New(parsedTime), nil
		}
	}

	// If no format matches, try to extract year and create a basic date
	if year := extractYear(dateStr); year != "" {
		if parsedTime, err := time.Parse("2006", year); err == nil {
			return timestamppb.New(parsedTime), nil
		}
	}

	return nil, fmt.Errorf("unable to parse date: %s", dateStr)
}

// extractYear attempts to extract a 4-digit year from a date string
func extractYear(dateStr string) string {
	parts := strings.FieldsFunc(dateStr, func(r rune) bool {
		return r == '/' || r == '-' || r == '.' || r == ' '
	})

	for _, part := range parts {
		if len(part) == 4 && isNumeric(part) {
			// Check if it's a reasonable year (1980-2050)
			if part >= "1980" && part <= "2050" {
				return part
			}
		}
	}
	return ""
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GetBootFirmwareInfo gets boot firmware specific information
func GetBootFirmwareInfo() (*pb.FirmwareInfo, error) {
	// This is typically the same as BIOS info on most systems
	return GetFirmwareInfo()
}
