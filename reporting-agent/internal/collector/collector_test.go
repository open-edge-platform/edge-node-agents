// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"errors"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/identity"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

// TestCollectorCollectDataSuccess checks that CollectData fills the model with correct values on success.
func TestCollectorCollectDataSuccess(t *testing.T) {
	c := collectorAllSuccess(t)
	cfg := config.Config{}
	root := c.CollectData(cfg)
	require.Equal(t, "Europe/Warsaw", root.OperatingSystem.Timezone, "Timezone should be set")
	require.Equal(t, "PL", root.OperatingSystem.Locale.CountryName, "Locale.CountryName should be set")
	require.Equal(t, "Linux", root.OperatingSystem.Kernel.Name, "Kernel.Name should be set")
	require.Equal(t, "ubuntu", root.OperatingSystem.Release.ID, "Release.ID should be set")
	require.InDelta(t, 123.0, root.OperatingSystem.UptimeSeconds, 0.0001, "UptimeSeconds should be set")
	require.Equal(t, "x86_64", root.ComputerSystem.CPU.Architecture, "CPU.Architecture should be set")
	require.NotNil(t, root.ComputerSystem.Memory.Devices, "Memory.Devices should be set")
	require.NotNil(t, root.ComputerSystem.Disk, "Disk should be set")
	require.Equal(t, "k3s", root.Kubernetes.Provider, "Kubernetes.Provider should be set")
	require.Equal(t, "gid", root.Identity.GroupID, "GroupID should be set")
	require.Equal(t, "mid", root.Identity.MachineID, "MachineID should be set")
	require.Equal(t, "imid", root.Identity.InitialMachineID, "InitialMachineID should be set")
}

func TestCollectorCollectDataShort(t *testing.T) {
	c := collectorAllSuccess(t)
	cfg := config.Config{}
	root := c.CollectDataShort(cfg)
	require.InDelta(t, 123.0, root.OperatingSystem.UptimeSeconds, 0.0001, "UptimeSeconds should be set")
	require.Equal(t, "k3s", root.Kubernetes.Provider, "Kubernetes.Provider should be set")
	require.Equal(t, "gid", root.Identity.GroupID, "GroupID should be set")
	require.Equal(t, "mid", root.Identity.MachineID, "MachineID should be set")
	require.Equal(t, "imid", root.Identity.InitialMachineID, "InitialMachineID should be set")
	// All other fields should be zero values
	require.Empty(t, root.OperatingSystem.Timezone, "Timezone should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Locale.CountryName, "Locale.CountryName should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Kernel.Name, "Kernel.Name should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Release.ID, "Release.ID should be empty in short mode")
	require.True(t, root.ComputerSystem.CPU.IsZero(), "CPU should be zero in short mode")
	require.True(t, root.ComputerSystem.Memory.IsZero(), "Memory should be zero in short mode")
	require.Empty(t, root.ComputerSystem.Disk, "Disk should be empty in short mode")
}

// TestCollectorCollectDataAllFailures checks that CollectData returns zero values on all errors.
func TestCollectorCollectDataAllFailures(t *testing.T) {
	c := collectorAllFailures(t)
	cfg := config.Config{}
	root := c.CollectData(cfg)
	require.Empty(t, root.OperatingSystem.Timezone, "Timezone should be empty on error")
	require.Empty(t, root.OperatingSystem.Locale.CountryName, "Locale.CountryName should be empty on error")
	require.Empty(t, root.OperatingSystem.Kernel.Name, "Kernel.Name should be empty on error")
	require.Empty(t, root.OperatingSystem.Release.ID, "Release.ID should be empty on error")
	require.InDelta(t, 0.0, root.OperatingSystem.UptimeSeconds, 0.0001, "UptimeSeconds should be zero on error")
	require.Empty(t, root.ComputerSystem.CPU.Architecture, "CPU.Architecture should be empty on error")
	require.NotNil(t, root.ComputerSystem.Memory.Devices, "Memory.Devices should be set (empty slice)")
	require.NotNil(t, root.ComputerSystem.Disk, "Disk should be set (empty slice)")
	require.Empty(t, root.Kubernetes.Provider, "Kubernetes.Provider should be empty on error")
	require.Empty(t, root.Identity.GroupID, "GroupID should be empty on error")
	require.Empty(t, root.Identity.MachineID, "MachineID should be empty on error")
	require.Empty(t, root.Identity.InitialMachineID, "InitialMachineID should be empty on error")
}

func TestCollectorCollectDataShortAllFailures(t *testing.T) {
	c := collectorAllFailures(t)
	cfg := config.Config{}
	root := c.CollectDataShort(cfg)
	require.InDelta(t, 0.0, root.OperatingSystem.UptimeSeconds, 0.0001, "UptimeSeconds should be zero on error")
	require.Empty(t, root.Kubernetes.Provider, "Kubernetes.Provider should be empty on error")
	require.Empty(t, root.Identity.GroupID, "GroupID should be empty on error")
	require.Empty(t, root.Identity.MachineID, "MachineID should be empty on error")
	require.Empty(t, root.Identity.InitialMachineID, "InitialMachineID should be empty on error")
	// All other fields should be zero values
	require.Empty(t, root.OperatingSystem.Timezone, "Timezone should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Locale.CountryName, "Locale.CountryName should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Kernel.Name, "Kernel.Name should be empty in short mode")
	require.Empty(t, root.OperatingSystem.Release.ID, "Release.ID should be empty in short mode")
	require.True(t, root.ComputerSystem.CPU.IsZero(), "CPU should be zero in short mode")
	require.True(t, root.ComputerSystem.Memory.IsZero(), "Memory should be zero in short mode")
	require.Empty(t, root.ComputerSystem.Disk, "Disk should be empty in short mode")
}

// TestCollectorCollectIdentitySuccess checks that collectIdentity fills the model.Identity fields on success.
func TestCollectorCollectIdentitySuccess(t *testing.T) {
	c := collectorAllSuccess(t)
	root := model.InitializeRoot()
	c.collectIdentity(&root)
	require.Equal(t, "gid", root.Identity.GroupID, "GroupID should be set")
	require.Equal(t, "mid", root.Identity.MachineID, "MachineID should be set")
	require.Equal(t, "imid", root.Identity.InitialMachineID, "InitialMachineID should be set")
}

// TestCollectorCollectIdentityFailures checks that collectIdentity sets empty fields on errors.
func TestCollectorCollectIdentityFailures(t *testing.T) {
	c := collectorAllFailures(t)
	root := model.InitializeRoot()
	c.collectIdentity(&root)
	require.Empty(t, root.Identity.GroupID, "GroupID should be empty on error")
	require.Empty(t, root.Identity.MachineID, "MachineID should be empty on error")
	require.Empty(t, root.Identity.InitialMachineID, "InitialMachineID should be empty on error")
}

// identityMock is a mock for the Identity interface used in tests.
type identityMock struct {
	groupID             string
	groupIDErr          error
	machineID           string
	machineIDErr        error
	initialMachineID    string
	initialMachineIDErr error
}

// Implement the identity.Provider interface.
func (i *identityMock) GetGroupID() (string, error) {
	return i.groupID, i.groupIDErr
}
func (i *identityMock) CalculateMachineID(utils.CmdExecutor) (string, error) {
	return i.machineID, i.machineIDErr
}
func (i *identityMock) SaveMachineIDs(string) (string, error) {
	return i.initialMachineID, i.initialMachineIDErr
}

// mockIdentityProviderFactory returns a function compatible with newIdentityFunc that returns the given identityMock as identity.Provider.
func mockIdentityProviderFactory(m identityMock) func() identity.Provider {
	return func() identity.Provider {
		return &m
	}
}

// collectorAllSuccess returns a Collector with all dependencies returning success.
func collectorAllSuccess(t *testing.T) *Collector {
	return &Collector{
		getTimezone: func(utils.CmdExecutor) (string, error) { return "Europe/Warsaw", nil },
		getLocaleData: func(utils.CmdExecutor) (model.Locale, error) {
			return model.Locale{CountryName: "PL"}, nil
		},
		getKernelData: func(utils.CmdExecutor) (model.Kernel, error) {
			return model.Kernel{Name: "Linux"}, nil
		},
		getReleaseData: func(utils.CmdExecutor) (model.Release, error) {
			return model.Release{ID: "ubuntu"}, nil
		},
		getUptimeData: func(utils.CmdExecutor) (float64, error) { return 123.0, nil },
		getCPUData: func(utils.CmdExecutor) (model.CPU, error) {
			return model.CPU{Architecture: "x86_64"}, nil
		},
		getMemoryData: func(utils.CmdExecutor) (model.Memory, error) {
			return model.Memory{Devices: []model.MemoryDevice{}}, nil
		},
		getDiskData: func(utils.CmdExecutor) ([]model.Disk, error) { return []model.Disk{}, nil },
		getKubernetesData: func(utils.CmdExecutor, config.K8sConfig) (model.Kubernetes, error) {
			return model.Kubernetes{Provider: "k3s"}, nil
		},
		newIdentity: mockIdentityProviderFactory(identityMock{
			groupID:          "gid",
			machineID:        "mid",
			initialMachineID: "imid",
		}),
		log:     zaptest.NewLogger(t).Sugar(),
		execCmd: utils.ExecCmdExecutor,
	}
}

// collectorAllFailures returns a Collector with all dependencies returning errors.
func collectorAllFailures(t *testing.T) *Collector {
	return &Collector{
		getTimezone:    func(utils.CmdExecutor) (string, error) { return "", errors.New("fail") },
		getLocaleData:  func(utils.CmdExecutor) (model.Locale, error) { return model.Locale{}, errors.New("fail") },
		getKernelData:  func(utils.CmdExecutor) (model.Kernel, error) { return model.Kernel{}, errors.New("fail") },
		getReleaseData: func(utils.CmdExecutor) (model.Release, error) { return model.Release{}, errors.New("fail") },
		getUptimeData:  func(utils.CmdExecutor) (float64, error) { return 0, errors.New("fail") },
		getCPUData:     func(utils.CmdExecutor) (model.CPU, error) { return model.CPU{}, errors.New("fail") },
		getMemoryData: func(utils.CmdExecutor) (model.Memory, error) {
			return model.Memory{Devices: []model.MemoryDevice{}}, errors.New("fail")
		},
		getDiskData: func(utils.CmdExecutor) ([]model.Disk, error) { return []model.Disk{}, errors.New("fail") },
		getKubernetesData: func(utils.CmdExecutor, config.K8sConfig) (model.Kubernetes, error) {
			return model.Kubernetes{}, errors.New("fail")
		},
		newIdentity: mockIdentityProviderFactory(identityMock{
			groupIDErr:          errors.New("fail"),
			machineIDErr:        errors.New("fail"),
			initialMachineIDErr: errors.New("fail"),
		}),
		log:     zaptest.NewLogger(t).Sugar(),
		execCmd: utils.ExecCmdExecutor,
	}
}
