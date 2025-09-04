/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"github.com/spf13/afero"
	"os"
	"runtime"
	"strings"
	"time"

	utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	osUpdater "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/os_updater"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetOSInfo retrieves operating system information
func GetOSInfo() (*pb.OSInfo, error) {
	osInfo := &pb.OSInfo{}
	osInfo.OsInformation = getOSInformation()
	return osInfo, nil
}

// getOSInformation gets comprehensive OS information
func getOSInformation() string {
	// Get system information
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host" // Fallback to a safe default
	}

	// Get kernel version on Linux
	var kernelVersion string
	if runtime.GOOS == "linux" {
		var fs = afero.NewOsFs()
		if data, err := utils.ReadFile(fs, "/proc/version"); err == nil {
			fields := strings.Fields(string(data))
			if len(fields) >= 3 {
				kernelVersion = fields[2] // Extract kernel version
			}
		}
	}

	// Get OS type using existing DetectOS function
	osType, err := osUpdater.DetectOS()
	if err != nil {
		osType = "Unknown"
	}

	// Get OS version
	osVersion := getOSVersion()

	// Get OS release date (if available)
	var releaseDateStr string
	if releaseDate := getOSReleaseDate(); releaseDate != nil {
		releaseDateStr = releaseDate.AsTime().Format("2006-01-02")
	}

	// Build comprehensive OS information string
	parts := []string{
		runtime.GOOS,   // OS name (linux, windows, etc.)
		hostname,       // System hostname
		kernelVersion,  // Kernel version
		runtime.GOARCH, // Architecture (amd64, arm64, etc.)
		osType,         // Detected OS type (Ubuntu, YoctoX86_64, etc.)
		osVersion,      // OS version (20.04, etc.)
	}

	// Add release date if available
	if releaseDateStr != "" {
		parts = append(parts, "released:"+releaseDateStr)
	}

	// Filter out empty parts
	var filteredParts []string
	for _, part := range parts {
		if part != "" && part != "Unknown" {
			filteredParts = append(filteredParts, part)
		}
	}

	return strings.Join(filteredParts, " ")
}

// getOSVersion gets the OS version
func getOSVersion() string {
	if runtime.GOOS == "linux" {
		// Try to get version from /etc/os-release first
		var fs = afero.NewOsFs()
		if data, err := utils.ReadFile(fs, "/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				// Try VERSION_ID first (more reliable)
				if strings.HasPrefix(line, "VERSION_ID=") {
					version := strings.Trim(line[11:], `"`)
					if version != "" {
						return version
					}
				}
				// Fallback to VERSION
				if strings.HasPrefix(line, "VERSION=") {
					version := strings.Trim(line[8:], `"`)
					if version != "" {
						return version
					}
				}
			}
		}

		// Try /etc/lsb-release
		if data, err := utils.ReadFile(fs, "/etc/lsb-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "DISTRIB_RELEASE=") {
					version := strings.Trim(line[16:], `"`)
					if version != "" {
						return version
					}
				}
			}
		}

		// Try /etc/debian_version for Debian-based systems
		if data, err := utils.ReadFile(fs, "/etc/debian_version"); err == nil {
			version := strings.TrimSpace(string(data))
			if version != "" {
				return version
			}
		}
	}

	return "Unknown"
}

// getOSReleaseDate gets the OS release date
func getOSReleaseDate() *timestamppb.Timestamp {
	if runtime.GOOS == "linux" {
		// Try to get build date from /etc/os-release
		var fs = afero.NewOsFs()
		if data, err := utils.ReadFile(fs, "/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "BUILD_ID=") {
					buildID := strings.Trim(line[9:], `"`)
					// Try various date formats
					dateFormats := []string{
						"2006-01-02",           // YYYY-MM-DD
						"2006.01.02",           // YYYY.MM.DD
						"20060102",             // YYYYMMDD
						"2006-01-02T15:04:05Z", // ISO format
					}

					for _, format := range dateFormats {
						if date, err := time.Parse(format, buildID); err == nil {
							return timestamppb.New(date)
						}
					}
				}
			}
		}
	}

	return nil
}
