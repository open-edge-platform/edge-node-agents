// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import "strings"

type Root struct {
	Identity        Identity        `json:"Identity"`
	OperatingSystem OperatingSystem `json:"OperatingSystem,omitzero"`
	ComputerSystem  ComputerSystem  `json:"ComputerSystem,omitzero"`
	Kubernetes      Kubernetes      `json:"Kubernetes,omitzero"`
}

type Identity struct {
	MachineID        string `json:"MachineId"`
	InitialMachineID string `json:"InitialMachineId"`
	PartnerID        string `json:"PartnerId"`
}

type OperatingSystem struct {
	Timezone      string  `json:"Timezone,omitzero"`
	Locale        Locale  `json:"Locale,omitzero"`
	Kernel        Kernel  `json:"Kernel,omitzero"`
	Release       Release `json:"Release,omitzero"`
	UptimeSeconds float64 `json:"UptimeSeconds,omitzero"`
}

type Locale struct {
	CountryName string `json:"CountryName,omitzero"`
	CountryAbbr string `json:"CountryAbbr,omitzero"`
	LangName    string `json:"LangName,omitzero"`
	LangAbbr    string `json:"LangAbbr,omitzero"`
}

type Kernel struct {
	Machine string `json:"Machine,omitzero"`
	Name    string `json:"Name,omitzero"`
	Release string `json:"Release,omitzero"`
	Version string `json:"Version,omitzero"`
	System  string `json:"System,omitzero"`
}

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

type ComputerSystem struct {
	CPU    CPU    `json:"CPU,omitzero"`
	Memory Memory `json:"Memory,omitzero"`
	Disk   []Disk `json:"Disk,omitzero"`
}

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

type Memory struct {
	Summary MemorySummary  `json:"Summary,omitzero"`
	Devices []MemoryDevice `json:"Devices,omitzero"`
}

type MemorySummary struct {
	TotalSizeMB      uint64 `json:"TotalSizeMB,omitzero"`
	CommonType       string `json:"CommonType,omitzero"`
	CommonFormFactor string `json:"CommonFormFactor,omitzero"`
}

type MemoryDevice struct {
	FormFactor   string `json:"FormFactor,omitzero"`
	Size         string `json:"Size,omitzero"`
	Type         string `json:"Type,omitzero"`
	Speed        string `json:"Speed,omitzero"`
	Manufacturer string `json:"Manufacturer,omitzero"`
}

type Disk struct {
	Name          string `json:"Name,omitzero"`
	Vendor        string `json:"Vendor,omitzero"`
	Model         string `json:"Model,omitzero"`
	Size          uint64 `json:"Size,omitzero"`
	ChildrenCount int    `json:"ChildrenCount,omitzero"`
}

type Kubernetes struct {
	Provider      string                  `json:"Provider,omitzero"`
	ServerVersion string                  `json:"ServerVersion,omitzero"`
	Applications  []KubernetesApplication `json:"Applications,omitzero"`
}

type KubernetesApplication struct {
	Name       string `json:"com.intel.edgeplatform.application.name,omitzero"`
	Version    string `json:"com.intel.edgeplatform.application.version,omitzero"`
	AppName    string `json:"app.kubernetes.io/name,omitzero"`
	AppVersion string `json:"app.kubernetes.io/version,omitzero"`
	AppPartOf  string `json:"app.kubernetes.io/part-of,omitzero"`
	HelmChart  string `json:"helm.sh/chart,omitzero"`
}

func InitializeRoot() Root {
	// Initialize only slice fields to avoid nulls in JSON, other fields can have default values
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

func (ka KubernetesApplication) GetKey() string {
	return strings.Join([]string{ka.Name, ka.Version, ka.AppName, ka.AppVersion, ka.AppPartOf, ka.HelmChart}, "|")
}

func (os OperatingSystem) IsZero() bool {
	return os.Timezone == "" &&
		os.Locale.IsZero() &&
		os.Kernel.IsZero() &&
		os.Release.IsZero() &&
		os.UptimeSeconds == 0
}

func (l Locale) IsZero() bool {
	return l.CountryName == "" &&
		l.CountryAbbr == "" &&
		l.LangName == "" &&
		l.LangAbbr == ""
}

func (k Kernel) IsZero() bool {
	return k.Machine == "" &&
		k.Name == "" &&
		k.Release == "" &&
		k.Version == "" &&
		k.System == ""
}

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

func (ms MemorySummary) IsZero() bool {
	return ms.TotalSizeMB == 0 &&
		ms.CommonType == "" &&
		ms.CommonFormFactor == ""
}

func (md MemoryDevice) IsZero() bool {
	return md.FormFactor == "" &&
		md.Size == "" &&
		md.Type == "" &&
		md.Speed == "" &&
		md.Manufacturer == ""
}

func (d Disk) IsZero() bool {
	return d.Name == "" &&
		d.Vendor == "" &&
		d.Model == "" &&
		d.Size == 0 &&
		d.ChildrenCount == 0
}

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

func (ka KubernetesApplication) IsZero() bool {
	return ka.Name == "" &&
		ka.Version == "" &&
		ka.AppName == "" &&
		ka.AppVersion == "" &&
		ka.AppPartOf == "" &&
		ka.HelmChart == ""
}
