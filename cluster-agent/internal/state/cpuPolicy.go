// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

const bitSize = 32

var reservedCPUsPattern = regexp.MustCompile(`--reserved-cpus[=\s]+["']?([^"'\s]+)["']?`)

type CPU struct {
	Sockets  uint32
	Topology *CPUTopology
}

type CPUTopology struct {
	Sockets []*Socket
}

type Socket struct {
	SocketID   uint32
	CoreGroups []*CoreGroup
}

type CoreGroup struct {
	Type string
	List []uint32
}

type cores struct {
	CPU    string
	Socket string
	MaxMhz string
}

// ResolveReservedCPUPolicy processes the install command and resolves reserved CPU policy keywords.
// If reserved-cpus is a literal value (e.g., "0-1", "16-31"), it passes through unchanged.
// If it's a policy keyword (auto, auto:pcore, auto:ecore), it computes the actual CPU set
// based on local system topology.
func ResolveReservedCPUPolicy(installCmd string) (string, error) {
	matches := reservedCPUsPattern.FindStringSubmatch(installCmd)
	if len(matches) < 2 {
		return installCmd, nil
	}

	reservedValue := matches[1]
	if !isCPUPolicy(reservedValue) {
		log.Debugf("reserved-cpus value %q is literal, passing through unchanged", reservedValue)
		return installCmd, nil
	}

	cpu, err := getCPUList()
	if err != nil {
		return "", fmt.Errorf("failed to get CPU information: %w", err)
	}
	if cpu == nil || cpu.Topology == nil || len(cpu.Topology.Sockets) == 0 {
		return "", fmt.Errorf("invalid or empty CPU topology detected")
	}

	var firstPCoreList, eCoreList []uint32
	for _, socket := range cpu.Topology.Sockets {
		for _, coreGroup := range socket.CoreGroups {
			if coreGroup.Type == "P-Core" && len(firstPCoreList) == 0 {
				firstPCoreList = coreGroup.List
			} else if coreGroup.Type == "E-Core" {
				eCoreList = append(eCoreList, coreGroup.List...)
			}
		}
	}

	var cpuSet string
	switch reservedValue {
	case "auto", "auto:pcore":
		if len(firstPCoreList) == 0 {
			return "", fmt.Errorf("no P-cores detected in CPU topology")
		}
		cpuSet = formatCPUSet(firstPCoreList)
	case "auto:ecore":
		if len(eCoreList) > 0 {
			cpuSet = formatCPUSet(eCoreList)
		} else if len(firstPCoreList) > 0 {
			log.Info("No E-cores detected, falling back to P-core reservation for auto:ecore policy")
			cpuSet = formatCPUSet(firstPCoreList)
		} else {
			return "", fmt.Errorf("no P-cores detected in CPU topology")
		}
	}

	oldFlag := matches[0]
	newFlag := fmt.Sprintf(`--reserved-cpus="%s"`, cpuSet)
	modifiedCmd := strings.Replace(installCmd, oldFlag, newFlag, 1)

	log.Infof("Resolved reserved CPU policy %q to %q", reservedValue, cpuSet)
	return modifiedCmd, nil
}

// isCPUPolicy reports whether the given value is a recognized CPU policy keyword.
func isCPUPolicy(value string) bool {
	return value == "auto" || value == "auto:pcore" || value == "auto:ecore"
}

// formatCPUSet converts a list of CPU IDs to kernel-acceptable range format.
// E.g., [0, 1, 2, 3] -> "0-3", [5] -> "5"
func formatCPUSet(cpuList []uint32) string {
	if len(cpuList) == 0 {
		return ""
	}
	first, last := cpuList[0], cpuList[len(cpuList)-1]
	if first == last {
		return fmt.Sprintf("%d", first)
	}
	return fmt.Sprintf("%d-%d", first, last)
}

// getCPUList collects CPU information from `lscpu` and processes the result to generate structured CPU data
// It returns with a CPU struct.
// This reuses the same logic as hardware-discovery-agent/internal/cpu/cpu.go
func getCPUList() (*CPU, error) {
	dataBytes, err := utils.ReadFromCommand(nil, "lscpu")
	if err != nil {
		return &CPU{}, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	lscpu := strings.Split(string(dataBytes), "\n")
	var cpu CPU
	var maxMhz string
	for _, attribute := range lscpu {
		attr := strings.TrimSpace(attribute)
		if strings.HasPrefix(attr, "Socket(s)") {
			socketStr := strings.TrimSpace(strings.TrimPrefix(attr, "Socket(s):"))
			sockets, err := strconv.ParseUint(socketStr, 10, bitSize)
			if err != nil {
				continue
			}
			cpu.Sockets = uint32(sockets)
		}
		if strings.HasPrefix(attr, "CPU max MHz:") {
			maxMhz = strings.TrimSpace(strings.TrimPrefix(attr, "CPU max MHz:"))
		}
	}

	// If the number of sockets has been retrieved using the `lscpu` command
	// above, we can determine how many P-Cores and E-Cores are enabled on the Edge Node.
	if cpu.Sockets != 0 {
		coreInfo := []*cores{}
		coreDetails, err := utils.ReadFromCommand(nil, "lscpu", "--extended=CPU,SOCKET,MAXMHZ")
		if err != nil {
			return &cpu, fmt.Errorf("failed to read data from command; error: %w", err)
		}
		parseCoreDetails := strings.SplitAfter(string(coreDetails), "\n")

		for _, coreData := range parseCoreDetails {
			if strings.Contains(coreData, "CPU") || coreData == "" {
				continue
			}
			var core cores
			coreValues := strings.Fields(coreData)
			core.CPU = coreValues[0]
			core.Socket = coreValues[1]
			coreMaxMhz := strings.Split(coreValues[2], "\n")
			core.MaxMhz = coreMaxMhz[0]
			coreInfo = append(coreInfo, &core)
		}

		cpuTopology := inferEPCores(cpu.Sockets, coreInfo, maxMhz)
		cpu.Topology = cpuTopology
	}

	return &cpu, nil
}

// inferEPCores classifies cores as P-Core or E-Core based on frequency heuristic.
func inferEPCores(sockets uint32, coreInfo []*cores, coreMaxFreq string) *CPUTopology {
	socketInfo := []*Socket{}
	for socketID := uint32(0); socketID < sockets; socketID++ {
		// Determine a target max frequency for E Core detection based on the max frequency from lscpu.
		maxCoreFreq, err := strconv.ParseUint(strings.TrimSuffix(coreMaxFreq, ".0000"), 10, 64)
		if err != nil {
			// If max frequency cannot be retrieved from lscpu, default to 0 so that all cores are considered P Cores.
			maxCoreFreq = 0
		}
		eCoreTargetFreq := (3 * maxCoreFreq) / 4
		socketDetails := getCoreGroupsPerSocket(socketID, coreInfo, eCoreTargetFreq)
		socketInfo = append(socketInfo, socketDetails)
	}
	return &CPUTopology{Sockets: socketInfo}
}

// getCoreGroupsPerSocket groups cores by type (P-Core vs E-Core) for a single socket
func getCoreGroupsPerSocket(socketID uint32, coreInfo []*cores, coreMaxFreq uint64) *Socket {
	pCoreList := make([]uint32, 0)
	eCoreList := make([]uint32, 0)

	for _, core := range coreInfo {
		socket, err := strconv.ParseUint(core.Socket, 10, bitSize)
		if err != nil {
			socket = 0
		}
		if socket == uint64(socketID) {
			cpu, err := strconv.ParseUint(core.CPU, 10, bitSize)
			if err != nil {
				continue
			}
			if core.MaxMhz == "-" {
				// If max frequency is not found, default to P Core for cpu ID and continue
				pCoreList = append(pCoreList, uint32(cpu))
				continue
			}
			coreFreq, err := strconv.ParseUint(strings.TrimSuffix(core.MaxMhz, ".0000"), 10, 64)
			if err != nil {
				continue
			}
			if coreFreq <= coreMaxFreq {
				eCoreList = append(eCoreList, uint32(cpu))
			} else {
				pCoreList = append(pCoreList, uint32(cpu))
			}
		}
	}

	coreGroups := []*CoreGroup{}
	coreGroups = append(coreGroups, &CoreGroup{
		Type: "P-Core",
		List: pCoreList,
	})
	if len(eCoreList) > 0 {
		coreGroups = append(coreGroups, &CoreGroup{
			Type: "E-Core",
			List: eCoreList,
		})
	}

	return &Socket{SocketID: socketID, CoreGroups: coreGroups}
}
