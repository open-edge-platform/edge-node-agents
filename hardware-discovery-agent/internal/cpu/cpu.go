// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cpu

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

const bitSize int = 32

type CPU struct {
	Arch     string
	Vendor   string
	Model    string
	Sockets  uint32
	Cores    uint32
	Threads  uint32
	Features []string
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

// GetCPUList collects CPU information from `lscpu` and processes the result to generate structured CPU data
// It returns with a CPU struct.
func GetCPUList(executor utils.CmdExecutor) (*CPU, error) {
	dataBytes, err := utils.ReadFromCommand(executor, "lscpu")
	if err != nil {
		return &CPU{}, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	lscpu := strings.Split(string(dataBytes), "\n")
	var cpu CPU
	var maxMhz string
	for _, attribute := range lscpu {
		attr := strings.TrimSpace(attribute)
		if strings.HasPrefix(attr, "Architecture") {
			cpu.Arch = strings.TrimSpace(strings.TrimPrefix(attr, "Architecture:"))
		}

		if strings.HasPrefix(attr, "Vendor ID") {
			cpu.Vendor = strings.TrimSpace(strings.TrimPrefix(attr, "Vendor ID:"))
		}

		if strings.HasPrefix(attr, "Model name") {
			cpu.Model = strings.TrimSpace(strings.TrimPrefix(attr, "Model name:"))
		}
		if strings.HasPrefix(attr, "Socket(s)") {
			socketStr := strings.TrimSpace(strings.TrimPrefix(attr, "Socket(s):"))
			sockets, err := strconv.ParseUint(socketStr, 10, bitSize)
			if err != nil {
				continue
			}
			cpu.Sockets = uint32(sockets)
		}
		if strings.HasPrefix(attr, "CPU(s)") {
			cpuStr := strings.TrimSpace(strings.TrimPrefix(attr, "CPU(s):"))
			cpus, err := strconv.ParseUint(cpuStr, 10, bitSize)
			if err != nil {
				continue
			}
			cpu.Threads = uint32(cpus)
		}
		if strings.HasPrefix(attr, "Core(s) per socket") {
			coresStr := strings.TrimSpace(strings.TrimPrefix(attr, "Core(s) per socket:"))
			cores, err := strconv.ParseUint(coresStr, 10, bitSize)
			if err != nil {
				continue
			}
			cpu.Cores = uint32(cores)
		}
		if strings.HasPrefix(attr, "Flags") {
			features := strings.TrimSpace(strings.TrimPrefix(attr, "Flags:"))
			cpu.Features = strings.Split(features, " ")
		}
		if strings.HasPrefix(attr, "CPU max MHz:") {
			maxMhz = strings.TrimSpace(strings.TrimPrefix(attr, "CPU max MHz:"))
		}
	}
	cpu.Cores = cpu.Cores * cpu.Sockets

	// If the number of sockets has been retrieved using the `lscpu` command
	// above, we can determine how many P-Cores and E-Cores are enabled on the Edge Node.
	if cpu.Sockets != 0 {
		coreInfo := []*cores{}
		coreDetails, err := utils.ReadFromCommand(executor, "lscpu", "--extended=CPU,SOCKET,MAXMHZ")
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

// Determine the number P-Cores and E-Cores enabled on the Edge Node. To do this
// we run the command `lscpu --extended=CPU,SOCKET,MAXMHZ` command and compare the total
// number of cores listed in the output to the total physical cores and total
// threads supported on the Edge Node.
// Source: https://stackoverflow.com/questions/71122837/how-to-detect-e-cores-and-p-cores-in-linux-alder-lake-system
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
