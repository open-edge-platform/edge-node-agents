/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/afero"
	"log"
	"os/exec"
	"runtime"
	"strings"

	utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
)

// PowerCapabilities represents the power management capabilities of the system
type PowerCapabilities struct {
	Shutdown  bool `json:"shutdown"`
	Reboot    bool `json:"reboot"`
	Suspend   bool `json:"suspend"`
	Hibernate bool `json:"hibernate"`
}

// GetPowerCapabilities retrieves power management capabilities for the system
func GetPowerCapabilities() (*pb.PowerCapabilitiesInfo, error) {
	// Only support Linux - fail fast for other platforms
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("power capabilities only supported on Linux, got: %s", runtime.GOOS)
	}

	capabilities := getLinuxPowerCapabilities()

	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return nil, err
	}

	return &pb.PowerCapabilitiesInfo{
		Shutdown:         capabilities.Shutdown,
		Reboot:           capabilities.Reboot,
		Suspend:          capabilities.Suspend,
		Hibernate:        capabilities.Hibernate,
		CapabilitiesJson: string(capabilitiesJSON),
	}, nil
}

// getLinuxPowerCapabilities checks power capabilities on Linux
func getLinuxPowerCapabilities() PowerCapabilities {
	capabilities := PowerCapabilities{
		Shutdown:  true, // Always supported on Linux
		Reboot:    true, // Always supported on Linux
		Suspend:   checkLinuxSuspendSupport(),
		Hibernate: checkLinuxHibernateSupport(),
	}

	// Fallback checks if primary methods fail
	if !capabilities.Suspend {
		capabilities.Suspend = checkPowerCommandAvailable("suspend") ||
			checkPowerCommandAvailable("pm-suspend")
	}

	if !capabilities.Hibernate {
		capabilities.Hibernate = checkPowerCommandAvailable("hibernate") ||
			checkPowerCommandAvailable("pm-hibernate")
	}

	return capabilities
}

// Alternative method for systems without /sys/power/state
func checkPowerCommandAvailable(command string) bool {
	// Check if command exists in PATH
	cmd := exec.Command("which", command)
	if cmd.Run() == nil {
		return true
	}

	// Check common locations
	commonPaths := []string{
		"/sbin/" + command,
		"/usr/sbin/" + command,
		"/bin/" + command,
		"/usr/bin/" + command,
	}

	for _, path := range commonPaths {
		var fs = afero.NewOsFs()
		if utils.IsFileExist(fs, path) {
			return true
		}
	}

	return false
}

// checkLinuxSuspendSupport checks if suspend is supported on Linux
func checkLinuxSuspendSupport() bool {
	// Check /sys/power/state for available states
	var fs = afero.NewOsFs()
	if data, err := utils.ReadFile(fs, "/sys/power/state"); err == nil {
		states := strings.TrimSpace(string(data))
		if strings.Contains(states, "mem") || strings.Contains(states, "standby") {
			// For systems without systemd, assume suspend is available if kernel supports it
			if !isSystemdAvailable() {
				return true
			}
			// With systemd, check if target is available
			return isSystemdTargetAvailable("suspend.target")
		}
	}
	return false
}

// checkLinuxHibernateSupport checks if hibernation is supported on Linux
func checkLinuxHibernateSupport() bool {
	// Check /sys/power/state for disk state
	var fs = afero.NewOsFs()
	if data, err := utils.ReadFile(fs, "/sys/power/state"); err == nil {
		states := strings.TrimSpace(string(data))
		if strings.Contains(states, "disk") {
			// Check if swap is available for hibernation
			if hasSwapSpace() {
				// For systems without systemd, assume hibernate is available if kernel supports it
				if !isSystemdAvailable() {
					return true
				}
				// With systemd, check if target is available
				return isSystemdTargetAvailable("hibernate.target")
			}
		}
	}
	return false
}

// isSystemdTargetAvailable checks if a systemd target is available
func isSystemdTargetAvailable(target string) bool {
	// Check if systemd is available
	if !isSystemdAvailable() {
		return false
	}

	// Primary check: see if target exists
	cmd := exec.Command("systemctl", "cat", target)
	if cmd.Run() == nil {
		return true
	}

	// Fallback: check if target is in list-unit-files
	cmd = exec.Command("systemctl", "list-unit-files", target)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), target)
}

// isSystemdAvailable checks if systemd is the init system
func isSystemdAvailable() bool {
	cmd := exec.Command("systemctl", "--version")
	return cmd.Run() == nil
}

// hasSwapSpace checks if swap space is available
func hasSwapSpace() bool {
	var fs = afero.NewOsFs()
	data, err := utils.ReadFile(fs, "/proc/swaps")
	if err != nil {
		log.Printf("Warning: Could not read /proc/swaps: %v", err)
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Header line should be present, check if there are actual swap entries
	if len(lines) < 2 {
		return false
	}

	// Check if any non-header line has content
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return true
		}
	}
	return false
}

// GetPowerCapabilitiesJSON returns power capabilities as a JSON string
func GetPowerCapabilitiesJSON() (string, error) {
	capabilities := getLinuxPowerCapabilities()

	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return "", err
	}

	return string(capabilitiesJSON), nil
}

// CheckPowerCommand checks if a specific power command is supported
func CheckPowerCommand(command string) bool {
	capabilities := getLinuxPowerCapabilities()

	switch strings.ToLower(command) {
	case "shutdown":
		return capabilities.Shutdown
	case "reboot":
		return capabilities.Reboot
	case "suspend":
		return capabilities.Suspend
	case "hibernate":
		return capabilities.Hibernate
	default:
		return false
	}
}

// GetSupportedPowerCommands returns a list of supported power commands
func GetSupportedPowerCommands() []string {
	capabilities := getLinuxPowerCapabilities()
	var supported []string

	if capabilities.Shutdown {
		supported = append(supported, "shutdown")
	}
	if capabilities.Reboot {
		supported = append(supported, "reboot")
	}
	if capabilities.Suspend {
		supported = append(supported, "suspend")
	}
	if capabilities.Hibernate {
		supported = append(supported, "hibernate")
	}

	return supported
}
