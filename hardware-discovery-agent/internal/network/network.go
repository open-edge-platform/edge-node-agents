// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/tool"
	hda_utils "github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

var log = logger.Logger

var (
	// The functions will be mocked during the testing.
	ReadFile           = utils.ReadFileNoLinks
	ReadDir            = os.ReadDir
	Readlink           = os.Readlink
	Stat               = os.Stat
	Glob               = filepath.Glob
	CollectEthtoolData = tool.CollectEthtoolData
)

var NETROOT = filepath.Join("/sys", "class", "net")

type IPAddress struct {
	IPAddress   string
	NetPrefBits int32
	ConfigMode  proto.ConfigMode
}

type Network struct {
	Name                string
	PciID               string
	Mac                 string
	LinkState           bool
	CurrentSpeed        uint64
	CurrentDuplex       string
	SupportedLinkMode   []string
	AdvertisingLinkMode []string
	Features            []string
	SriovEnabled        bool
	SriovNumVfs         uint32
	SriovVfsTotal       uint32
	PeerName            string
	PeerDescription     string
	PeerMac             string
	PeerManagementIP    string
	PeerPort            string
	IPAddresses         []*IPAddress
	Mtu                 uint32
	BmcNet              bool
}

func GetNICList(executor hda_utils.CmdExecutor) ([]*Network, proto.BmInfo_BmType, string, error) {
	nicList := []*Network{}

	files, err := ReadDir("/sys/class/net")
	if err != nil {
		log.Errorf("read NIC dir error, return empty NIC list : %v", err)
		return nicList, proto.BmInfo_NONE, "", err
	}

	bmcAddr := ""
	bmType := proto.BmInfo_NONE
	if _, err = Stat("/dev/ipmi0"); !errors.Is(err, os.ErrNotExist) {
		bmcData, err := hda_utils.ReadFromCommand(executor, "sudo", "ipmitool", "lan", "print", "1")
		// In ipmitool version 1.8.19, the above call returns an empty value and an error status code even
		// though the same command run manually is successful from command line. To avoid an issue where the
		// failure of this version causes the collection of network interfaces to fail, log an error message
		// but continue with the default settings for BMC set before this check.
		if err != nil {
			log.Errorf("failed to get BMC information : %v", err)
		} else {
			bmcAddr = parseIpmiInfo(bmcData)
			bmType = proto.BmInfo_IPMI
		}
	}

	for _, file := range files {
		// filename is the interface name
		filename := file.Name()

		// ignore lookback and bonding record file
		if filename == "lo" || filename == "bonding_masters" {
			continue
		}

		nicPath := filepath.Join(NETROOT, filename)
		dest, err := Readlink(nicPath)
		if err != nil {
			log.Errorf("unable to read %s, skipping\n : %v", nicPath, err)
			continue
		}

		// Pass the virtual interface
		if strings.Contains(dest, "devices/virtual/net") {
			continue
		}

		// Initial the NIC struct
		var nic Network

		// Assign the basic parameters
		nic.Name = filename
		nic.PciID = getNICPCIAddress(filename)

		// The test files about getNICPCIAddress has complete, next is the next line L50
		nic.Mac = getNICPhysicalAddress(filename)

		// Collect the IP addresses
		interfaceIPInfo, err := hda_utils.ReadFromCommand(executor, "ip", "addr", "show", filename)
		if err != nil {
			log.Errorf("failed to get interface IP information : %v", err)
			continue
		}
		ipAddresses, bmcNet, mtu, err := parseIPInfo(string(interfaceIPInfo), bmcAddr)
		if err != nil {
			continue
		}
		nic.IPAddresses = ipAddresses
		nic.BmcNet = bmcNet
		nic.Mtu = mtu

		// Collect the SR-IOV parameters
		nic.SriovEnabled = checkIfSriovEnabled(filename)
		if nic.SriovEnabled {
			nic.SriovNumVfs, nic.SriovVfsTotal = getSriovVfCount(filename)
		} else {
			nic.SriovNumVfs = 0
			nic.SriovVfsTotal = 0
		}

		ethValues, err := CollectEthtoolData(filename)

		if err != nil {
			// Sometimes the ioctl syscall returns with "resource busy" error,
			// and we will not collect the information of this NIC as a result.
			log.Errorf("collecting ethtool data for %s failed, skip\n : %v", filename, err)
			continue
		}
		nic.Features = ethValues.Features
		nic.LinkState = ethValues.LinkState
		nic.CurrentSpeed = ethValues.CurrentSpeed
		nic.CurrentDuplex = ethValues.CurrentDuplex
		nic.SupportedLinkMode = ethValues.SupportedLinkMode
		nic.AdvertisingLinkMode = ethValues.AdvertisingLinkMode

		nicList = append(nicList, &nic)
	}

	return nicList, bmType, bmcAddr, nil
}

func parseIpmiInfo(ipmiData []byte) string {
	ipmiInterface := strings.SplitAfter(string(ipmiData), "IP Address")
	ipmiAddress := strings.SplitAfter(ipmiInterface[2], ": ")
	addr := strings.Split(ipmiAddress[1], "\n")

	return addr[0]
}

func parseIPInfo(ipInfo string, bmcAddr string) ([]*IPAddress, bool, uint32, error) {
	mtuInfo := strings.Split(ipInfo, "mtu ")
	mtu := strings.Split(mtuInfo[1], " ")
	interfaceMtu, err := strconv.ParseUint(mtu[0], 10, 64)
	if err != nil {
		log.Errorf("parsing mtu error : %v", err)
		return []*IPAddress{}, false, 0, err
	}

	ipAddressList := []*IPAddress{}
	isBmcInterface := false

	networkDetails := strings.Split(ipInfo, "inet ")
	for index, netDetails := range networkDetails {
		if index == 0 {
			continue
		}

		var address IPAddress
		ipAddress := strings.Split(netDetails, "/")
		address.IPAddress = ipAddress[0]

		// Check if this is the BMC connection
		if bmcAddr == ipAddress[0] && !isBmcInterface {
			isBmcInterface = true
		}

		netPrefix := strings.Split(ipAddress[1], " ")
		prefix, err := strconv.ParseInt(netPrefix[0], 10, 64)
		if err != nil {
			log.Errorf("parsing network prefix error : %v", err)
			return []*IPAddress{}, false, 0, err
		}
		address.NetPrefBits = int32(prefix)

		if strings.Contains(netDetails, "dynamic") {
			address.ConfigMode = proto.ConfigMode_CONFIG_MODE_DYNAMIC
		} else {
			address.ConfigMode = proto.ConfigMode_CONFIG_MODE_STATIC
		}

		ipAddressList = append(ipAddressList, &address)
	}

	return ipAddressList, isBmcInterface, uint32(interfaceMtu), nil
}

func checkIfSriovEnabled(nicName string) bool {
	numVfsPath := filepath.Join(NETROOT, nicName, "device", "sriov_numvfs")
	if _, err := Stat(numVfsPath); errors.Is(err, os.ErrNotExist) {
		log.Warnf("network interface %s wasn't enabled with SR-IOV feature", nicName)
		return false
	}
	return true
}

func getSriovVfCount(nicName string) (uint32, uint32) {
	devPath := filepath.Join(NETROOT, nicName, "device")

	// Get Number of VFs currently created on the interface
	numVfsContents, err := ReadFile(filepath.Join(devPath, "sriov_numvfs"))
	if err != nil {
		log.Errorf("reading sriov_numvfs file error with network interface %s : %v", nicName, err)
		return 0, 0
	}

	numVfs, err := strconv.ParseUint(strings.TrimSpace(string(numVfsContents)), 10, 32)
	if err != nil {
		log.Errorf("parsing sriov vf number error with network interface %s : %v", nicName, err)
		return 0, 0
	}

	// Get total number of VFs that can be created on the interface
	totalVfsContents, err := ReadFile(filepath.Join(devPath, "sriov_totalvfs"))
	if err != nil {
		log.Errorf("reading sriov_totalvfs file error with network interface %s : %v", nicName, err)
		return uint32(numVfs), 0
	}

	totalVfs, err := strconv.ParseUint(strings.TrimSpace(string(totalVfsContents)), 10, 32)
	if err != nil {
		log.Errorf("parsing sriov vf number error with network interface %s : %v", nicName, err)
		return uint32(numVfs), 0
	}

	return uint32(numVfs), uint32(totalVfs)
}

func getNICPhysicalAddress(nicName string) string {
	addrPath := filepath.Join(NETROOT, nicName, "address")
	contents, err := ReadFile(addrPath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(contents))
}

// getNICPCIAddress uses the softlink of networkinterfacecard to extract PCI id
// It returns a string with NIC's PCI address.
func getNICPCIAddress(nicName string) string {
	netPath := filepath.Join(NETROOT, nicName)
	dest, err := Readlink(netPath)
	if err != nil {
		return ""
	}

	netDev := filepath.Clean(filepath.Join(NETROOT, dest))
	dest, err = Readlink(filepath.Join(netDev, "device"))
	if err != nil {
		return ""
	}

	devPath := filepath.Clean(filepath.Join(netDev, dest))
	dest, err = Readlink(filepath.Join(devPath, "subsystem"))
	if err != nil {
		return ""
	}

	if !strings.HasSuffix(dest, "/bus/pci") {
		return ""
	}

	return filepath.Base(devPath)
}
