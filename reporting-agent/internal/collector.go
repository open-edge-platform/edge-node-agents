// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"go.uber.org/zap"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/cpu"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/disk"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/identity"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/k8s"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/memory"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/system"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

type Collector struct {
	getTimezone       getTimezoneFunc
	getLocaleData     getLocaleDataFunc
	getKernelData     getKernelDataFunc
	getReleaseData    getReleaseDataFunc
	getUptimeData     getUptimeDataFunc
	getCPUData        getCPUDataFunc
	getMemoryData     getMemoryDataFunc
	getDiskData       getDiskDataFunc
	getKubernetesData getKubernetesDataFunc
	newIdentity       newIdentityFunc
	log               *zap.SugaredLogger
	execCmd           utils.CmdExecutor
}

type (
	getTimezoneFunc       func(utils.CmdExecutor) (string, error)
	getLocaleDataFunc     func(utils.CmdExecutor) (model.Locale, error)
	getKernelDataFunc     func(utils.CmdExecutor) (model.Kernel, error)
	getReleaseDataFunc    func(utils.CmdExecutor) (model.Release, error)
	getUptimeDataFunc     func(utils.CmdExecutor) (float64, error)
	getCPUDataFunc        func(utils.CmdExecutor) (model.CPU, error)
	getMemoryDataFunc     func(utils.CmdExecutor) (model.Memory, error)
	getDiskDataFunc       func(utils.CmdExecutor) ([]model.Disk, error)
	getKubernetesDataFunc func(utils.CmdExecutor, config.K8sConfig) (model.Kubernetes, error)
	newIdentityFunc       func() identity.Provider
)

func NewCollector(logger *zap.SugaredLogger) Collector {
	return Collector{
		getTimezone:       system.GetTimezone,
		getLocaleData:     system.GetLocaleData,
		getKernelData:     system.GetKernelData,
		getReleaseData:    system.GetReleaseData,
		getUptimeData:     system.GetUptimeData,
		getCPUData:        cpu.GetCPUData,
		getMemoryData:     memory.GetMemoryData,
		getDiskData:       disk.GetDiskData,
		getKubernetesData: k8s.GetKubernetesData,
		newIdentity:       identity.NewIdentity,
		log:               logger,
		execCmd:           utils.ExecCmdExecutor,
	}
}

// CollectData collects system and hardware data and returns the model.Root struct.
func (c *Collector) CollectData(cfg config.Config) model.Root {
	var err error
	dataCollected := model.InitializeRoot()

	dataCollected.OperatingSystem.Timezone, err = c.getTimezone(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting timezone: %v", err)
	}

	dataCollected.OperatingSystem.Locale, err = c.getLocaleData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting locale data: %v", err)
	}

	dataCollected.OperatingSystem.Kernel, err = c.getKernelData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting kernel data: %v", err)
	}

	dataCollected.OperatingSystem.Release, err = c.getReleaseData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting release data: %v", err)
	}

	dataCollected.OperatingSystem.UptimeSeconds, err = c.getUptimeData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting uptime data: %v", err)
	}

	dataCollected.ComputerSystem.CPU, err = c.getCPUData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting CPU data: %v", err)
	}

	dataCollected.ComputerSystem.Memory, err = c.getMemoryData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting memory data: %v", err)
	}

	dataCollected.ComputerSystem.Disk, err = c.getDiskData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting disk data: %v", err)
	}

	dataCollected.Kubernetes, err = c.getKubernetesData(c.execCmd, cfg.K8s)
	if err != nil {
		c.log.Errorf("Error occurred while collecting k8s data: %v", err)
	}

	c.collectIdentity(&dataCollected)

	return dataCollected
}

// CollectDataShort collects only Identity, UptimeSeconds and Kubernetes data.
func (c *Collector) CollectDataShort(cfg config.Config) model.Root {
	var err error
	dataCollected := model.InitializeRoot()

	dataCollected.OperatingSystem.UptimeSeconds, err = c.getUptimeData(c.execCmd)
	if err != nil {
		c.log.Errorf("Error occurred while collecting uptime data: %v", err)
	}

	dataCollected.Kubernetes, err = c.getKubernetesData(c.execCmd, cfg.K8s)
	if err != nil {
		c.log.Errorf("Error occurred while collecting k8s data: %v", err)
	}

	c.collectIdentity(&dataCollected)

	return dataCollected
}

func (c *Collector) collectIdentity(data *model.Root) {
	idt := c.newIdentity()
	groupID, err := idt.GetGroupID()
	if err != nil {
		c.log.Errorf("Failed to get group ID: %v", err)
	}
	data.Identity.GroupID = groupID

	machineID, err := idt.CalculateMachineID(c.execCmd)
	if err != nil {
		c.log.Errorf("Failed to calculate machine ID: %v", err)
	}
	data.Identity.MachineID = machineID

	initialMachineID, err := idt.SaveMachineIDs(machineID)
	if err != nil {
		c.log.Errorf("Failed to save machine ID: %v", err)
	}
	data.Identity.InitialMachineID = initialMachineID
}
