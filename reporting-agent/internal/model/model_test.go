// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInitializeRoot checks that InitializeRoot initializes slices and nested structs correctly.
func TestInitializeRoot(t *testing.T) {
	root := InitializeRoot()
	require.NotNil(t, root.ComputerSystem.Memory.Devices, "Memory.Devices should be initialized")
	require.Empty(t, root.ComputerSystem.Memory.Devices, "Memory.Devices should be empty")
	require.NotNil(t, root.ComputerSystem.Disk, "Disk should be initialized")
	require.Empty(t, root.ComputerSystem.Disk, "Disk should be empty")
	require.NotNil(t, root.Kubernetes.Applications, "Kubernetes.Applications should be initialized")
	require.Empty(t, root.Kubernetes.Applications, "Kubernetes.Applications should be empty")
}

// TestKubernetesApplicationGetKey checks that GetKey returns the correct key.
func TestKubernetesApplicationGetKey(t *testing.T) {
	app := KubernetesApplication{
		Name:       "intel",
		Version:    "1.0",
		AppName:    "kube",
		AppVersion: "2.0",
		AppPartOf:  "platform",
		HelmChart:  "chart-1.2.3",
	}
	expected := "intel|1.0|kube|2.0|platform|chart-1.2.3"
	require.Equal(t, expected, app.GetKey(), "GetKey should return correct key")
}

// TestOperatingSystemIsZero checks IsZero for OperatingSystem struct.
func TestOperatingSystemIsZero(t *testing.T) {
	var os OperatingSystem
	require.True(t, os.IsZero(), "IsZero should return true for zero-value OperatingSystem")

	os.Timezone = "Europe/Warsaw"
	require.False(t, os.IsZero(), "IsZero should return false if Timezone is set")
	os.Timezone = ""
	os.Locale = Locale{CountryName: "Poland"}
	require.False(t, os.IsZero(), "IsZero should return false if Locale is not zero")
	os.Locale = Locale{}
	os.Kernel = Kernel{Name: "Linux"}
	require.False(t, os.IsZero(), "IsZero should return false if Kernel is not zero")
	os.Kernel = Kernel{}
	os.Release = Release{ID: "ubuntu"}
	require.False(t, os.IsZero(), "IsZero should return false if Release is not zero")
	os.Release = Release{}
	os.UptimeSeconds = 1.23
	require.False(t, os.IsZero(), "IsZero should return false if UptimeSeconds is not zero")
}

// TestLocaleIsZero checks IsZero for Locale struct.
func TestLocaleIsZero(t *testing.T) {
	var l Locale
	require.True(t, l.IsZero(), "IsZero should return true for zero-value Locale")
	l.CountryName = "Poland"
	require.False(t, l.IsZero(), "IsZero should return false if CountryName is set")
	l = Locale{CountryAbbr: "PL"}
	require.False(t, l.IsZero(), "IsZero should return false if CountryAbbr is set")
	l = Locale{LangName: "Polski"}
	require.False(t, l.IsZero(), "IsZero should return false if LangName is set")
	l = Locale{LangAbbr: "pl"}
	require.False(t, l.IsZero(), "IsZero should return false if LangAbbr is set")
}

// TestKernelIsZero checks IsZero for Kernel struct.
func TestKernelIsZero(t *testing.T) {
	var k Kernel
	require.True(t, k.IsZero(), "IsZero should return true for zero-value Kernel")
	k.Machine = "x86_64"
	require.False(t, k.IsZero(), "IsZero should return false if Machine is set")
	k = Kernel{Name: "Linux"}
	require.False(t, k.IsZero(), "IsZero should return false if Name is set")
	k = Kernel{Release: "1.0"}
	require.False(t, k.IsZero(), "IsZero should return false if Release is set")
	k = Kernel{Version: "v1"}
	require.False(t, k.IsZero(), "IsZero should return false if Version is set")
	k = Kernel{System: "GNU/Linux"}
	require.False(t, k.IsZero(), "IsZero should return false if System is set")
}

// TestReleaseIsZero checks IsZero for Release struct.
func TestReleaseIsZero(t *testing.T) {
	var r Release
	require.True(t, r.IsZero(), "IsZero should return true for zero-value Release")
	r.ID = "ubuntu"
	require.False(t, r.IsZero(), "IsZero should return false if ID is set")
	r = Release{VersionID: "22.04"}
	require.False(t, r.IsZero(), "IsZero should return false if VersionID is set")
	r = Release{Version: "22.04"}
	require.False(t, r.IsZero(), "IsZero should return false if Version is set")
	r = Release{Codename: "jammy"}
	require.False(t, r.IsZero(), "IsZero should return false if Codename is set")
	r = Release{Family: "debian"}
	require.False(t, r.IsZero(), "IsZero should return false if Family is set")
	r = Release{BuildID: "build"}
	require.False(t, r.IsZero(), "IsZero should return false if BuildID is set")
	r = Release{ImageID: "img"}
	require.False(t, r.IsZero(), "IsZero should return false if ImageID is set")
	r = Release{ImageVersion: "1.0"}
	require.False(t, r.IsZero(), "IsZero should return false if ImageVersion is set")
}

// TestComputerSystemIsZero checks IsZero for ComputerSystem struct.
func TestComputerSystemIsZero(t *testing.T) {
	var cs ComputerSystem
	require.True(t, cs.IsZero(), "IsZero should return true for zero-value ComputerSystem")

	cs.CPU = CPU{Architecture: "x86_64"}
	require.False(t, cs.IsZero(), "IsZero should return false if CPU is not zero")
	cs.CPU = CPU{}
	cs.Memory = Memory{Summary: MemorySummary{TotalSizeMB: 1}}
	require.False(t, cs.IsZero(), "IsZero should return false if Memory is not zero")
	cs.Memory = Memory{}
	cs.Disk = []Disk{{Name: "sda"}}
	require.False(t, cs.IsZero(), "IsZero should return false if Disk is not zero")
	cs.Disk = []Disk{}
}

// TestCPUIsZero checks IsZero for CPU struct.
func TestCPUIsZero(t *testing.T) {
	var cpu CPU
	require.True(t, cpu.IsZero(), "IsZero should return true for zero-value CPU")
	cpu.Architecture = "x86_64"
	require.False(t, cpu.IsZero(), "IsZero should return false if Architecture is set")
	cpu = CPU{Vendor: "Intel"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Vendor is set")
	cpu = CPU{Family: "6"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Family is set")
	cpu = CPU{ModelName: "Xeon"}
	require.False(t, cpu.IsZero(), "IsZero should return false if ModelName is set")
	cpu = CPU{Model: "123"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Model is set")
	cpu = CPU{Stepping: "1"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Stepping is set")
	cpu = CPU{SocketCount: 1}
	require.False(t, cpu.IsZero(), "IsZero should return false if SocketCount is set")
	cpu = CPU{CoreCount: 1}
	require.False(t, cpu.IsZero(), "IsZero should return false if CoreCount is set")
	cpu = CPU{ThreadCount: 1}
	require.False(t, cpu.IsZero(), "IsZero should return false if ThreadCount is set")
	cpu = CPU{Virtualization: "full"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Virtualization is set")
	cpu = CPU{Hypervisor: "kvm"}
	require.False(t, cpu.IsZero(), "IsZero should return false if Hypervisor is set")
}

// TestMemoryIsZero checks IsZero for Memory struct.
func TestMemoryIsZero(t *testing.T) {
	var m Memory
	require.True(t, m.IsZero(), "IsZero should return true for zero-value Memory")
	m.Summary = MemorySummary{TotalSizeMB: 1}
	require.False(t, m.IsZero(), "IsZero should return false if Summary is not zero")
	m = Memory{Devices: []MemoryDevice{{FormFactor: "DIMM"}}}
	require.False(t, m.IsZero(), "IsZero should return false if MemoryDevice is not zero")
	m = Memory{Devices: []MemoryDevice{}}
	require.True(t, m.IsZero(), "IsZero should return true if MemoryDevice is empty and Summary is zero")
}

// TestMemorySummaryIsZero checks IsZero for MemorySummary struct.
func TestMemorySummaryIsZero(t *testing.T) {
	var ms MemorySummary
	require.True(t, ms.IsZero(), "IsZero should return true for zero-value MemorySummary")
	ms.TotalSizeMB = 1
	require.False(t, ms.IsZero(), "IsZero should return false if TotalSizeMB is set")
	ms = MemorySummary{CommonType: "DDR4"}
	require.False(t, ms.IsZero(), "IsZero should return false if CommonType is set")
	ms = MemorySummary{CommonFormFactor: "DIMM"}
	require.False(t, ms.IsZero(), "IsZero should return false if CommonFormFactor is set")
}

// TestMemoryDeviceIsZero checks IsZero for MemoryDevice struct.
func TestMemoryDeviceIsZero(t *testing.T) {
	var md MemoryDevice
	require.True(t, md.IsZero(), "IsZero should return true for zero-value MemoryDevice")
	md.FormFactor = "DIMM"
	require.False(t, md.IsZero(), "IsZero should return false if FormFactor is set")
	md = MemoryDevice{Size: "4GB"}
	require.False(t, md.IsZero(), "IsZero should return false if Size is set")
	md = MemoryDevice{Type: "DDR4"}
	require.False(t, md.IsZero(), "IsZero should return false if Type is set")
	md = MemoryDevice{Speed: "3200"}
	require.False(t, md.IsZero(), "IsZero should return false if Speed is set")
	md = MemoryDevice{Manufacturer: "Kingston"}
	require.False(t, md.IsZero(), "IsZero should return false if Manufacturer is set")
}

// TestDiskIsZero checks IsZero for Disk struct.
func TestDiskIsZero(t *testing.T) {
	var d Disk
	require.True(t, d.IsZero(), "IsZero should return true for zero-value Disk")
	d.Name = "sda"
	require.False(t, d.IsZero(), "IsZero should return false if Name is set")
	d = Disk{Vendor: "Intel"}
	require.False(t, d.IsZero(), "IsZero should return false if Vendor is set")
	d = Disk{Model: "SSD"}
	require.False(t, d.IsZero(), "IsZero should return false if Model is set")
	d = Disk{Size: 1024}
	require.False(t, d.IsZero(), "IsZero should return false if Size is set")
	d = Disk{ChildrenCount: 1}
	require.False(t, d.IsZero(), "IsZero should return false if ChildrenCount is set")
}

// TestKubernetesIsZero checks IsZero for Kubernetes struct.
func TestKubernetesIsZero(t *testing.T) {
	var k8s Kubernetes
	require.True(t, k8s.IsZero(), "IsZero should return true for zero-value Kubernetes")
	k8s.Provider = "k3s"
	require.False(t, k8s.IsZero(), "IsZero should return false if Provider is set")
	k8s = Kubernetes{ServerVersion: "1.0"}
	require.False(t, k8s.IsZero(), "IsZero should return false if ServerVersion is set")
	k8s = Kubernetes{Applications: []KubernetesApplication{{AppName: "app"}}}
	require.False(t, k8s.IsZero(), "IsZero should return false if Applications is not zero")
	k8s = Kubernetes{Applications: []KubernetesApplication{}}
	require.True(t, k8s.IsZero(), "IsZero should return true if Applications is empty and other fields are zero")
}

// TestKubernetesApplicationIsZero checks IsZero for KubernetesApplication struct.
func TestKubernetesApplicationIsZero(t *testing.T) {
	var ka KubernetesApplication
	require.True(t, ka.IsZero(), "IsZero should return true for zero-value KubernetesApplication")
	ka.Name = "intel"
	require.False(t, ka.IsZero(), "IsZero should return false if Name is set")
	ka = KubernetesApplication{Version: "1.0"}
	require.False(t, ka.IsZero(), "IsZero should return false if Version is set")
	ka = KubernetesApplication{AppName: "kube"}
	require.False(t, ka.IsZero(), "IsZero should return false if AppName is set")
	ka = KubernetesApplication{AppVersion: "2.0"}
	require.False(t, ka.IsZero(), "IsZero should return false if AppVersion is set")
	ka = KubernetesApplication{AppPartOf: "platform"}
	require.False(t, ka.IsZero(), "IsZero should return false if AppPartOf is set")
	ka = KubernetesApplication{HelmChart: "chart-1.2.3"}
	require.False(t, ka.IsZero(), "IsZero should return false if HelmChart is set")
}
