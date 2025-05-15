// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package network_test

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/network"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/tool"
)

var testDataPath = "../../test/data"

func parseFilePathIntoTestData(path string) (ret string) {
	// make /sys/block/sda into _sys_block_sda and trim out the first character
	// then it will return with "sys_block_sda"
	return strings.ReplaceAll(path, "/", "_")[1:]
}

// NOTE: Due to the implementation of mock function of testing, we mocked 2 functions
// os.ReadDir and os.ReadFile, in order to mock them, the fake function needs to
// accept the exact same parameter type as os.ReadDir / os.ReadFile.
// However, os.ReadDir takes fs.DirEntry as input, but fs.DirEntry is an interface
// Therefore, we have to implement a type (the "file" type) which implements fs.DirEntry.
type file struct {
	name string
}

func (f *file) Name() string             { return f.name }
func (*file) IsDir() bool                { return false }
func (*file) Type() fs.FileMode          { return 0 }
func (*file) Info() (fs.FileInfo, error) { return nil, nil }

// Mocked_ReadDir implements simple version of os.ReadDir, it reads directory file list from
// a file named "some_path_in_os_content", the file must ends with "_content".
// It returns a []fs.DirEntry slice which has nil on every field except Name.
func mockedReadDir(path string) (ret []fs.DirEntry, err error) {
	// content is a special keyword, when a testdata with "content" suffix
	// it represents it's a directory list
	mockedPath := "dir-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadDirFailure(path string) (ret []fs.DirEntry, err error) {
	return []fs.DirEntry{}, fmt.Errorf("Failed to read %v, exiting", path)
}

func mockedReadDirNoSriov(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-no-sriov-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadDirIncorrectVFs(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-incorrect-vfs-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadDirIncorrectNicInfo(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-incorrect-nic-info-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadDirNoSriovTotalVFs(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-no-sriov-total-vfs-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadDirIncorrectTotalVFs(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-incorrect-total-vfs-" + parseFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testDataPath, mockedPath))
	if err != nil {
		return ret, err
	}
	defer source.Close()
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		filename := scanner.Text()
		fileList = append(fileList, &file{
			name: filename,
		})
	}

	return fileList, nil
}

func mockedReadFile(path string) ([]byte, error) {
	mockedPath := "file-" + parseFilePathIntoTestData(path)
	return os.ReadFile(filepath.Join(testDataPath, mockedPath))
}

func mockedReadFileNoPhysAddress(path string) ([]byte, error) {
	return []byte{}, fmt.Errorf("Failed to read %v", path)
}

func mockedReadFilePhysAddressSymlink(path string) ([]byte, error) {
	mockedPath := "file-" + parseFilePathIntoTestData(path)
	if strings.Contains(mockedPath, "ens3_address") {
		symlinkPath := "/tmp/symlink_file.txt"
		err := os.Symlink(filepath.Join(testDataPath, mockedPath), symlinkPath)
		if err != nil {
			return os.ReadFile(filepath.Join(testDataPath, mockedPath))
		}
		defer os.Remove(symlinkPath)
		return utils.ReadFileNoLinks(symlinkPath)
	}
	return os.ReadFile(filepath.Join(testDataPath, mockedPath))
}

func mockedReadFileSriovNumVfsSymlink(path string) ([]byte, error) {
	mockedPath := "file-" + parseFilePathIntoTestData(path)
	if strings.Contains(mockedPath, "device_sriov_numvfs") {
		symlinkPath := "/tmp/symlink_file.txt"
		err := os.Symlink(filepath.Join(testDataPath, mockedPath), symlinkPath)
		if err != nil {
			return os.ReadFile(filepath.Join(testDataPath, mockedPath))
		}
		defer os.Remove(symlinkPath)
		return utils.ReadFileNoLinks(symlinkPath)
	}
	return os.ReadFile(filepath.Join(testDataPath, mockedPath))
}

func mockedReadFileSriovTotalVfsSymlink(path string) ([]byte, error) {
	mockedPath := "file-" + parseFilePathIntoTestData(path)
	if strings.Contains(mockedPath, "device_sriov_totalvfs") {
		symlinkPath := "/tmp/symlink_file.txt"
		err := os.Symlink(filepath.Join(testDataPath, mockedPath), symlinkPath)
		if err != nil {
			return os.ReadFile(filepath.Join(testDataPath, mockedPath))
		}
		defer os.Remove(symlinkPath)
		return utils.ReadFileNoLinks(symlinkPath)
	}
	return os.ReadFile(filepath.Join(testDataPath, mockedPath))
}

func mockedReadlink(path string) (string, error) {
	mockedPath := "link-" + parseFilePathIntoTestData(path)
	out, err := os.ReadFile(filepath.Join(testDataPath, mockedPath))

	return string(out), err
}

func mockedReadlinkNoDevice(path string) (string, error) {
	return "", fmt.Errorf("Failed to read %v", path)
}

// check for first or second call for mockedReadlinkNoNicDir.
var checkCall = false

func setCheckCall() bool {
	if !checkCall {
		checkCall = true
		return true
	}
	return false
}

func mockedReadlinkNoNicDir(path string) (string, error) {
	if setCheckCall() {
		return "", nil
	}
	return "", fmt.Errorf("failed to read %v", path)
}

func mockedReadlinkNoNicDevice(path string) (string, error) {
	mockedPath := "link-no-dev-" + parseFilePathIntoTestData(path)
	out, err := os.ReadFile(filepath.Join(testDataPath, mockedPath))

	return string(out), err
}

func mockedReadlinkNoNicSubsystem(path string) (string, error) {
	mockedPath := "link-no-subsystem-" + parseFilePathIntoTestData(path)
	out, err := os.ReadFile(filepath.Join(testDataPath, mockedPath))

	return string(out), err
}

func mockedReadlinkNonPciDevice(path string) (string, error) {
	mockedPath := "link-non-pci-dev-" + parseFilePathIntoTestData(path)
	out, err := os.ReadFile(filepath.Join(testDataPath, mockedPath))

	return string(out), err
}

// mockedStat will mock os.Stat, which is used to check if file exist
// Since we can make sure the testing data exists in test/data, we can return without err
// to make the function works well.
func mockedStat(path string) (fs.FileInfo, error) {
	mockedPath := "file-" + parseFilePathIntoTestData(path)
	_, err := os.ReadFile(filepath.Join(testDataPath, mockedPath))
	return nil, err
}

func mockedStatSriovFailure(path string) (fs.FileInfo, error) {
	_ = "file-" + parseFilePathIntoTestData(path)
	return nil, nil
}

func mockedStatNoBMC(path string) (fs.FileInfo, error) {
	_ = "file-" + parseFilePathIntoTestData(path)
	return nil, os.ErrNotExist
}

func mockedCollectEthtoolData(_ string) (*tool.EthtoolValues, error) {
	ret := &tool.EthtoolValues{
		LinkState: true,
		SupportedLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"1000baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"100baseT Half",
			"1000baseT Full",
		},
		CurrentSpeed:  1000,
		CurrentDuplex: "Full",
		Features: []string{
			"rx-gro",
			"rx-vlan-filter",
			"rx-vlan-hw-parse",
			"tx-checksum-ip-generic",
			"tx-generic-segmentation",
			"tx-scatter-gather",
			"tx-tcp-segmentation",
			"tx-vlan-hw-insert",
		},
	}
	return ret, nil
}

func mockedCollectEthtoolDataFailure(_ string) (*tool.EthtoolValues, error) {
	return &tool.EthtoolValues{}, errors.New("Failed to collect Ethtool information")
}

func expectedIPAddressIPv4() []*network.IPAddress {
	expectedIPAddresses := []*network.IPAddress{}
	expectedIP := &network.IPAddress{
		IPAddress:   "192.168.1.50",
		NetPrefBits: 24,
		ConfigMode:  proto.ConfigMode_CONFIG_MODE_DYNAMIC,
	}
	expectedIPAddresses = append(expectedIPAddresses, expectedIP)
	return expectedIPAddresses
}

func expectedIPAddressIPv6() []*network.IPAddress {
	expectedIPAddresses := []*network.IPAddress{}
	expectedIP := &network.IPAddress{
		IPAddress:   "192.168.192.168.1.50",
		NetPrefBits: 40,
		ConfigMode:  proto.ConfigMode_CONFIG_MODE_STATIC,
	}
	expectedIPAddresses = append(expectedIPAddresses, expectedIP)
	return expectedIPAddresses
}

func expectedIPAddresses(ipAddress []string, prefix []int32, configMode []proto.ConfigMode) []*network.IPAddress {
	expectedIPAddresses := []*network.IPAddress{}
	for index, ipAddr := range ipAddress {
		expectedIP := &network.IPAddress{
			IPAddress:   ipAddr,
			NetPrefBits: prefix[index],
			ConfigMode:  configMode[index],
		}
		expectedIPAddresses = append(expectedIPAddresses, expectedIP)
	}
	return expectedIPAddresses
}

func expectedResult(name string, pciID string, sriovEnabled bool, sriovVfs uint32, sriovTotalVfs uint32, ipAddresses []*network.IPAddress, isBmc bool) []*network.Network {

	expect := []*network.Network{}
	expectedRes := &network.Network{
		Name:          name,
		PciID:         pciID,
		Mac:           "52:54:00:12:34:56",
		LinkState:     true,
		CurrentSpeed:  1000,
		CurrentDuplex: "Full",
		SupportedLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"1000baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"100baseT Half",
			"1000baseT Full",
		},
		Features: []string{
			"rx-gro",
			"rx-vlan-filter",
			"rx-vlan-hw-parse",
			"tx-checksum-ip-generic",
			"tx-generic-segmentation",
			"tx-scatter-gather",
			"tx-tcp-segmentation",
			"tx-vlan-hw-insert",
		},
		SriovEnabled:  sriovEnabled,
		SriovNumVfs:   sriovVfs,
		SriovVfsTotal: sriovTotalVfs,
		IPAddresses:   ipAddresses,
		Mtu:           1500,
		BmcNet:        isBmc,
	}

	expect = append(expect, expectedRes)
	return expect
}

func checkResultPositiveCase(t *testing.T, err error, bmcType proto.BmInfo_BmType, bmcAddr string, got, res []*network.Network) {
	if err != nil {
		t.Errorf("Received err %v", err)
	}
	if bmcType != proto.BmInfo_IPMI {
		t.Errorf("Received incorrect BMC type")
	}
	if bmcAddr == "" {
		t.Errorf("No BMC address received")
	}
	if !reflect.DeepEqual(got, res) {
		t.Errorf("Got %v, expect %v", got, res)
	}
}

func TestGetNICList(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "0000:00:03.0", true, 2, 64, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListSriovDisabled(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirNoSriov
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens4", "0000:00:04.0", false, 0, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoBMC(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirNoSriov
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStatNoBMC

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens4", "0000:00:04.0", false, 0, 0, ipAddress, false)
	if err != nil {
		t.Errorf("Received err %v", err)
	}
	if bmType != proto.BmInfo_NONE {
		t.Errorf("Received incorrect BMC type")
	}
	if bmcAddr != "" {
		t.Errorf("No BMC address received")
	}
	if !reflect.DeepEqual(got, res) {
		t.Errorf("Got %v, expect %v", got, res)
	}
}

func TestFailedNICDeviceList(t *testing.T) {
	network.ReadDir = mockedReadDirFailure

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	if err == nil {
		t.Errorf("No error message received")
	}
	if bmType != proto.BmInfo_NONE {
		t.Errorf("Incorrect BMC type received")
	}
	if bmcAddr != "" {
		t.Errorf("Non-empty bmcAddr received")
	}
	res := []*network.Network{}
	if !reflect.DeepEqual(got, res) {
		t.Errorf("Got %v, expect %v", got, res)
	}
}

func TestFailedNICDeviceRead(t *testing.T) {
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlinkNoDevice
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	res := []*network.Network{}
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListEthtoolFail(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolDataFailure
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	res := []*network.Network{}
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNumVfsSriovReadFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirNoSriov
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStatSriovFailure

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens4", "0000:00:04.0", true, 0, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNumVfsSriovSymlink(t *testing.T) {
	network.ReadFile = mockedReadFileSriovNumVfsSymlink
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "0000:00:03.0", true, 0, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNumVfsSriovIncorrect(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectVFs
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens5", "0000:00:05.0", true, 0, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListTotalVfsSriovReadFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirNoSriovTotalVFs
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens6", "0000:00:06.0", true, 2, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListTotalVfsSriovSymlink(t *testing.T) {
	network.ReadFile = mockedReadFileSriovTotalVfsSymlink
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "0000:00:03.0", true, 2, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListTotalVfsSriovIncorrect(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectTotalVFs
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens7", "0000:00:07.0", true, 2, 0, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoPhysAddress(t *testing.T) {
	network.ReadFile = mockedReadFileNoPhysAddress
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	res := []*network.Network{}
	expectedRes := &network.Network{
		Name:          "ens3",
		PciID:         "0000:00:03.0",
		Mac:           "",
		LinkState:     true,
		CurrentSpeed:  1000,
		CurrentDuplex: "Full",
		SupportedLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"1000baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"100baseT Half",
			"1000baseT Full",
		},
		Features: []string{
			"rx-gro",
			"rx-vlan-filter",
			"rx-vlan-hw-parse",
			"tx-checksum-ip-generic",
			"tx-generic-segmentation",
			"tx-scatter-gather",
			"tx-tcp-segmentation",
			"tx-vlan-hw-insert",
		},
		SriovEnabled: true,
		IPAddresses:  expectedIPAddressIPv4(),
		Mtu:          1500,
		BmcNet:       true,
	}
	res = append(res, expectedRes)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListPhysAddressSymlink(t *testing.T) {
	network.ReadFile = mockedReadFilePhysAddressSymlink
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	res := []*network.Network{}
	expectedRes := &network.Network{
		Name:          "ens3",
		PciID:         "0000:00:03.0",
		Mac:           "",
		LinkState:     true,
		CurrentSpeed:  1000,
		CurrentDuplex: "Full",
		SupportedLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"1000baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"10baseT Half",
			"100baseT Full",
			"100baseT Half",
			"1000baseT Full",
		},
		Features: []string{
			"rx-gro",
			"rx-vlan-filter",
			"rx-vlan-hw-parse",
			"tx-checksum-ip-generic",
			"tx-generic-segmentation",
			"tx-scatter-gather",
			"tx-tcp-segmentation",
			"tx-vlan-hw-insert",
		},
		SriovEnabled:  true,
		SriovNumVfs:   2,
		SriovVfsTotal: 64,
		IPAddresses:   expectedIPAddressIPv4(),
		Mtu:           1500,
		BmcNet:        true,
	}
	res = append(res, expectedRes)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoNicDirectory(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectNicInfo
	network.Readlink = mockedReadlinkNoNicDir
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "", true, 2, 64, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoNicDevice(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectNicInfo
	network.Readlink = mockedReadlinkNoNicDevice
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "", true, 2, 64, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoNicSubsystem(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectNicInfo
	network.Readlink = mockedReadlinkNoNicSubsystem
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "", true, 2, 64, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNonPciDevice(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirIncorrectNicInfo
	network.Readlink = mockedReadlinkNonPciDevice
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorSuccess)
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens3", "", true, 2, 64, ipAddress, true)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListIpmiFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDirNoSriov
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorIpmiFailed)
	if err != nil {
		t.Errorf("No error message received")
	}
	if bmType != proto.BmInfo_NONE {
		t.Errorf("Incorrect BMC type received")
	}
	if bmcAddr != "" {
		t.Errorf("Non-empty bmcAddr received")
	}
	ipAddress := expectedIPAddressIPv4()
	res := expectedResult("ens4", "0000:00:04.0", false, 0, 0, ipAddress, false)
	if !reflect.DeepEqual(got, res) {
		t.Errorf("Got %v, expect %v", got, res)
	}
}

func TestGetNICListIpAddrFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorIPAddrFailed)
	res := []*network.Network{}
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListIpAddrMtuFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorMtuFailed)
	res := []*network.Network{}
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListIpAddrNetPrefixFailed(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorPrefixFailed)
	res := []*network.Network{}
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListStaticIPv6(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorStaticIPv6Success)
	ipAddress := expectedIPAddressIPv6()
	res := expectedResult("ens3", "0000:00:03.0", true, 2, 64, ipAddress, false)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListMultipleIpAddresses(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorMultipleIPAddressesSuccess)
	addresses := []string{"192.168.1.25", "192.168.5.37"}
	prefixes := []int32{24, 24}
	configModes := []proto.ConfigMode{proto.ConfigMode_CONFIG_MODE_DYNAMIC, proto.ConfigMode_CONFIG_MODE_STATIC}
	ipAddresses := expectedIPAddresses(addresses, prefixes, configModes)
	res := expectedResult("ens3", "0000:00:03.0", true, 2, 64, ipAddresses, false)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func TestGetNICListNoAddress(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat

	got, bmType, bmcAddr, err := network.GetNICList(testCmdExecutorNoIPAddress)
	res := expectedResult("ens3", "0000:00:03.0", true, 2, 64, []*network.IPAddress{}, false)
	checkResultPositiveCase(t, err, bmType, bmcAddr, got, res)
}

func testCmdReceived(args ...string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "lan") {
			return true
		}
	}
	return false
}

func testCmdExecutorSuccess(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestIpAddrExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorIpmiFailed(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestExecutionFailed", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorIPAddrFailed(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestExecutionFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorMtuFailed(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionIncorrectMtu", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorPrefixFailed(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionIncorrectPrefix", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorStaticIPv6Success(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionStaticIPv6", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorMultipleIPAddressesSuccess(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionMultipleAddresses", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorNoIPAddress(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestIpmiExecutionSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
	cs := []string{"-test.run=TestIpAddrExecutionNoAddress", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestIpmiExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ipmitool.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestIpAddrExecutionSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestExecutionFailed(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}

func TestIpAddrExecutionIncorrectMtu(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr_mtu.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestIpAddrExecutionIncorrectPrefix(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr_prefix.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestIpAddrExecutionStaticIPv6(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr_static_ipv6.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestIpAddrExecutionMultipleAddresses(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr_multi.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestIpAddrExecutionNoAddress(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr_no_addr.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
