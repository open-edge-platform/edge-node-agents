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

type Cpu struct {
	Arch     string
	Vendor   string
	Model    string
	Sockets  uint32
	Cores    uint32
	Threads  uint32
	Features []string
	Topology *CpuTopology
}

type CpuTopology struct {
	Sockets []*Socket
}

type Socket struct {
	SocketId   uint32
	CoreGroups []*CoreGroup
}

type CoreGroup struct {
	Type string
	List []uint32
}

type cores struct {
	Cpu    string
	Socket string
	MaxMhz string
}

// GetCpuList collects CPU information from `lscpu` and processes the result to generate structured CPU data
// It returns with a Cpu struct
func GetCpuList(executor utils.CmdExecutor) (*Cpu, error) {
	dataBytes, err := utils.ReadFromCommand(executor, "lscpu")
	if err != nil {
		return &Cpu{}, fmt.Errorf("failed to read data from command; error: %w", err)
	}

	lscpu := strings.Split(string(dataBytes), "\n")
	var cpu Cpu
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
			sockets, err := strconv.ParseUint(socketStr, 10, 64)
			if err != nil {
				continue
			}
			cpu.Sockets = uint32(sockets)
		}
		if strings.HasPrefix(attr, "CPU(s)") {
			cpuStr := strings.TrimSpace(strings.TrimPrefix(attr, "CPU(s):"))
			cpus, err := strconv.ParseUint(cpuStr, 10, 64)
			if err != nil {
				continue
			}
			cpu.Threads = uint32(cpus)
		}
		if strings.HasPrefix(attr, "Core(s) per socket") {
			coresStr := strings.TrimSpace(strings.TrimPrefix(attr, "Core(s) per socket:"))
			cores, err := strconv.ParseUint(coresStr, 10, 64)
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
			core.Cpu = coreValues[0]
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
func inferEPCores(sockets uint32, coreInfo []*cores, coreMaxFreq string) *CpuTopology {
	socketInfo := []*Socket{}
	for socketId := uint32(0); socketId < sockets; socketId++ {
		// Determine a target max frequency for E Core detection based on the max frequency from lscpu.
		maxCoreFreq, err := strconv.ParseUint(strings.TrimSuffix(coreMaxFreq, ".0000"), 10, 64)
		if err != nil {
			// If max frequency cannot be retrieved from lscpu, default to 0 so that all cores are considered P Cores.
			maxCoreFreq = 0
		}
		coreFreq := uint32(maxCoreFreq)
		eCoreTargetFreq := (3 * coreFreq) / 4
		socketDetails := getCoreGroupsPerSocket(socketId, coreInfo, eCoreTargetFreq)
		socketInfo = append(socketInfo, socketDetails)
	}
	return &CpuTopology{Sockets: socketInfo}
}

func getCoreGroupsPerSocket(socketId uint32, coreInfo []*cores, coreMaxFreq uint32) *Socket {
	pCoreList := make([]uint32, 0)
	eCoreList := make([]uint32, 0)

	for _, core := range coreInfo {
		socket, err := strconv.ParseUint(core.Socket, 10, 64)
		if err != nil {
			socket = 0
		}
		if uint32(socket) == socketId {
			cpu, err := strconv.ParseUint(core.Cpu, 10, 64)
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
			if uint32(coreFreq) <= coreMaxFreq {
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

	return &Socket{SocketId: socketId, CoreGroups: coreGroups}
}
