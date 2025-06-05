// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import "strings"

// Root represents the root structure of the system information model.
type Root struct {
	Identity        Identity        `json:"Identity"`
	OperatingSystem OperatingSystem `json:"OperatingSystem,omitzero"`
	ComputerSystem  ComputerSystem  `json:"ComputerSystem,omitzero"`
	Kubernetes      Kubernetes      `json:"Kubernetes,omitzero"`
}

// Identity holds the unique identifiers for the machine and its group.
type Identity struct {
	MachineID        string `json:"MachineId"`
	InitialMachineID string `json:"InitialMachineId"`
	GroupID          string `json:"GroupId"`
}

// OperatingSystem contains information about the operating system, including timezone, locale, kernel, release, and uptime.
type OperatingSystem struct {
	Timezone      string  `json:"Timezone,omitzero"`
	Locale        Locale  `json:"Locale,omitzero"`
	Kernel        Kernel  `json:"Kernel,omitzero"`
	Release       Release `json:"Release,omitzero"`
	UptimeSeconds float64 `json:"UptimeSeconds,omitzero"`
}

// Locale holds information about the system's locale settings, including country and language names and abbreviations.
type Locale struct {
	CountryName string `json:"CountryName,omitzero"`
	CountryAbbr string `json:"CountryAbbr,omitzero"`
	LangName    string `json:"LangName,omitzero"`
	LangAbbr    string `json:"LangAbbr,omitzero"`
}

// Kernel contains information about the system's kernel, including machine type, name, release version, and system type.
type Kernel struct {
	Machine string `json:"Machine,omitzero"`
	Name    string `json:"Name,omitzero"`
	Release string `json:"Release,omitzero"`
	Version string `json:"Version,omitzero"`
	System  string `json:"System,omitzero"`
}

// Release holds information about the operating system release, including ID, version, codename, family, build ID, image ID, and image version.
type Release struct {
	ID           string `json:"Id,omitzero"`
	VersionID    string `json:"VersionId,omitzero"`
	Version      string `json:"Version,omitzero"`
	Codename     string `json:"Codename,omitzero"`
	Family       string `json:"Family,omitzero"`
	BuildID      string `json:"BuildID,omitzero"`
	ImageID      string `json:"ImageID,omitzero"`
	ImageVersion string `json:"ImageVersion,omitzero"`
}

// ComputerSystem contains information about the computer system, including CPU, memory, and disk details.
type ComputerSystem struct {
	CPU    CPU    `json:"CPU,omitzero"`
	Memory Memory `json:"Memory,omitzero"`
	Disk   []Disk `json:"Disk,omitzero"`
}

// CPU holds information about the CPU architecture, vendor, family, model, stepping, socket count, core count, thread count,
// virtualization support, and hypervisor.
type CPU struct {
	Architecture   string `json:"Architecture,omitzero"`
	Vendor         string `json:"Vendor,omitzero"`
	Family         string `json:"Family,omitzero"`
	ModelName      string `json:"ModelName,omitzero"`
	Model          string `json:"Model,omitzero"`
	Stepping       string `json:"Stepping,omitzero"`
	SocketCount    uint64 `json:"SocketCount,omitzero"`
	CoreCount      uint64 `json:"CoreCount,omitzero"`
	ThreadCount    uint64 `json:"ThreadCount,omitzero"`
	Virtualization string `json:"Virtualization,omitzero"`
	Hypervisor     string `json:"Hypervisor,omitzero"`
}

// Memory contains information about the system's memory, including a summary and details of individual memory devices.
type Memory struct {
	Summary MemorySummary  `json:"Summary,omitzero"`
	Devices []MemoryDevice `json:"Devices,omitzero"`
}

// MemorySummary provides a summary of the system's memory, including total size, common type, and common form factor.
type MemorySummary struct {
	TotalSizeMB      uint64 `json:"TotalSizeMB,omitzero"`
	CommonType       string `json:"CommonType,omitzero"`
	CommonFormFactor string `json:"CommonFormFactor,omitzero"`
}

// MemoryDevice holds information about individual memory devices, including form factor, size, type, speed, and manufacturer.
type MemoryDevice struct {
	FormFactor   string `json:"FormFactor,omitzero"`
	Size         string `json:"Size,omitzero"`
	Type         string `json:"Type,omitzero"`
	Speed        string `json:"Speed,omitzero"`
	Manufacturer string `json:"Manufacturer,omitzero"`
}

// Disk represents a disk device in the system, including its name, vendor, model, size, and number of children.
type Disk struct {
	Name          string `json:"Name,omitzero"`
	Vendor        string `json:"Vendor,omitzero"`
	Model         string `json:"Model,omitzero"`
	Size          uint64 `json:"Size,omitzero"`
	ChildrenCount int    `json:"ChildrenCount,omitzero"`
}

// Kubernetes holds information about the Kubernetes provider, server version, and applications running in the cluster.
type Kubernetes struct {
	Provider      string                  `json:"Provider,omitzero"`
	ServerVersion string                  `json:"ServerVersion,omitzero"`
	Applications  []KubernetesApplication `json:"Applications,omitzero"`
}

// KubernetesApplication represents an application running in a Kubernetes cluster, including its name, version, app name, app version, part of, and Helm chart.
type KubernetesApplication struct {
	Name       string `json:"com.intel.edgeplatform.application.name,omitzero"`
	Version    string `json:"com.intel.edgeplatform.application.version,omitzero"`
	AppName    string `json:"app.kubernetes.io/name,omitzero"`
	AppVersion string `json:"app.kubernetes.io/version,omitzero"`
	AppPartOf  string `json:"app.kubernetes.io/part-of,omitzero"`
	HelmChart  string `json:"helm.sh/chart,omitzero"`
}

// InitializeRoot creates a new Root instance with default values for its fields (except slices, which are initialized to empty slices to avoid nulls in JSON).
func InitializeRoot() Root {
	return Root{
		ComputerSystem: ComputerSystem{
			Memory: Memory{
				Devices: []MemoryDevice{},
			},
			Disk: []Disk{},
		},
		Kubernetes: Kubernetes{
			Applications: []KubernetesApplication{},
		},
	}
}

// GetKey generates a unique key for the KubernetesApplication based on its attributes.
func (ka KubernetesApplication) GetKey() string {
	return strings.Join([]string{ka.Name, ka.Version, ka.AppName, ka.AppVersion, ka.AppPartOf, ka.HelmChart}, "|")
}

// IsZero checks if the Root instance has no meaningful data.
func (os OperatingSystem) IsZero() bool {
	return os.Timezone == "" &&
		os.Locale.IsZero() &&
		os.Kernel.IsZero() &&
		os.Release.IsZero() &&
		os.UptimeSeconds == 0
}

// IsZero checks if the Locale instance has no meaningful data.
func (l Locale) IsZero() bool {
	return l.CountryName == "" &&
		l.CountryAbbr == "" &&
		l.LangName == "" &&
		l.LangAbbr == ""
}

// IsZero checks if the Kernel instance has no meaningful data.
func (k Kernel) IsZero() bool {
	return k.Machine == "" &&
		k.Name == "" &&
		k.Release == "" &&
		k.Version == "" &&
		k.System == ""
}

// IsZero checks if the Release instance has no meaningful data.
func (r Release) IsZero() bool {
	return r.ID == "" &&
		r.VersionID == "" &&
		r.Version == "" &&
		r.Codename == "" &&
		r.Family == "" &&
		r.BuildID == "" &&
		r.ImageID == "" &&
		r.ImageVersion == ""
}

// IsZero checks if the ComputerSystem instance has no meaningful data.
func (cs ComputerSystem) IsZero() bool {
	allDisksZero := true
	for _, d := range cs.Disk {
		if !d.IsZero() {
			allDisksZero = false
			break
		}
	}
	return cs.CPU.IsZero() &&
		cs.Memory.IsZero() &&
		(len(cs.Disk) == 0 || allDisksZero)
}

// IsZero checks if the CPU instance has no meaningful data.
func (cpu CPU) IsZero() bool {
	return cpu.Architecture == "" &&
		cpu.Vendor == "" &&
		cpu.Family == "" &&
		cpu.ModelName == "" &&
		cpu.Model == "" &&
		cpu.Stepping == "" &&
		cpu.SocketCount == 0 &&
		cpu.CoreCount == 0 &&
		cpu.ThreadCount == 0 &&
		cpu.Virtualization == "" &&
		cpu.Hypervisor == ""
}

// IsZero checks if the Memory instance has no meaningful data.
func (m Memory) IsZero() bool {
	allDevicesZero := true
	for _, d := range m.Devices {
		if !d.IsZero() {
			allDevicesZero = false
			break
		}
	}
	return m.Summary.IsZero() &&
		(len(m.Devices) == 0 || allDevicesZero)
}

// IsZero checks if the MemorySummary instance has no meaningful data.
func (ms MemorySummary) IsZero() bool {
	return ms.TotalSizeMB == 0 &&
		ms.CommonType == "" &&
		ms.CommonFormFactor == ""
}

// IsZero checks if the MemoryDevice instance has no meaningful data.
func (md MemoryDevice) IsZero() bool {
	return md.FormFactor == "" &&
		md.Size == "" &&
		md.Type == "" &&
		md.Speed == "" &&
		md.Manufacturer == ""
}

// IsZero checks if the Disk instance has no meaningful data.
func (d Disk) IsZero() bool {
	return d.Name == "" &&
		d.Vendor == "" &&
		d.Model == "" &&
		d.Size == 0 &&
		d.ChildrenCount == 0
}

// IsZero checks if the Kubernetes instance has no meaningful data.
func (k8s Kubernetes) IsZero() bool {
	allAppsZero := true
	for _, app := range k8s.Applications {
		if !app.IsZero() {
			allAppsZero = false
			break
		}
	}
	return k8s.Provider == "" &&
		k8s.ServerVersion == "" &&
		(len(k8s.Applications) == 0 || allAppsZero)
}

// IsZero checks if the KubernetesApplication instance has no meaningful data.
func (ka KubernetesApplication) IsZero() bool {
	return ka.Name == "" &&
		ka.Version == "" &&
		ka.AppName == "" &&
		ka.AppVersion == "" &&
		ka.AppPartOf == "" &&
		ka.HelmChart == ""
}
