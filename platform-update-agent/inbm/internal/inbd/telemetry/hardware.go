/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"github.com/spf13/afero"
	"log"
	"os/exec"
	"runtime"
	"strings"

	utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
)

func GetHardwareInfo() (*pb.HardwareInfo, error) {
	hw := &pb.HardwareInfo{}

	// Get manufacturer, product, etc. from DMI
	if runtime.GOOS == "linux" {
		if manufacturer, err := readDMIInfo("/sys/class/dmi/id/sys_vendor"); err == nil {
			hw.SystemManufacturer = strings.TrimSpace(manufacturer)
		} else {
			log.Printf("Failed to read SystemManufacturer: %v", err)
		}

		if product, err := readDMIInfo("/sys/class/dmi/id/product_name"); err == nil {
			hw.SystemProductName = strings.TrimSpace(product)
		} else {
			log.Printf("Failed to read SystemProductName: %v", err)
		}

		// Get CPU ID from /proc/cpuinfo
		if cpuID, err := getCPUInfo(); err == nil {
			hw.CpuId = cpuID
		} else {
			log.Printf("Failed to read CpuId: %v", err)
		}

		// Get memory info
		if memInfo, err := getMemoryInfo(); err == nil {
			hw.TotalPhysicalMemory = memInfo
		} else {
			log.Printf("Failed to read TotalPhysicalMemory: %v", err)
		}

		// Get disk info
		if diskInfo, err := getDiskInfo(); err == nil {
			hw.DiskInformation = diskInfo
		} else {
			log.Printf("Failed to read DiskInformation: %v", err)
		}
	}

	return hw, nil
}

func readDMIInfo(path string) (string, error) {
	var fs = afero.NewOsFs()
	data, err := utils.ReadFile(fs, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getCPUInfo() (string, error) {
	var fs = afero.NewOsFs()
	data, err := utils.ReadFile(fs, "/proc/cpuinfo")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return runtime.GOARCH, nil
}

// getMemoryInfo retrieves the total physical memory from /proc/meminfo
// It returns the total memory in kilobytes as a string.
// If the file cannot be read or the information is not found, it returns an error.
func getMemoryInfo() (string, error) {
	var fs = afero.NewOsFs()
	data, err := utils.ReadFile(fs, "/proc/meminfo")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1] + " kB", nil
			}
		}
	}
	return "", fmt.Errorf("memory info not found")
}

// getDiskInfo retrieves disk information using the lsblk command
func getDiskInfo() (string, error) {
	cmd := exec.Command("lsblk", "-b", "-d", "-o", "name,size,rota", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
