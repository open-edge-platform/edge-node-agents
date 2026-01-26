// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package comms_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/testutils"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/network"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/tool"
)

type mockServer struct {
	proto.HostmgrServer
}

type failedServer struct {
	proto.HostmgrServer
}

func (*mockServer) UpdateHostSystemInfoByGUID(_ context.Context, _ *proto.UpdateHostSystemInfoByGUIDRequest) (*proto.UpdateHostSystemInfoByGUIDResponse, error) { //nolint:unused
	updateHostSystemInfoByGUIDResponse := proto.UpdateHostSystemInfoByGUIDResponse{}
	return &updateHostSystemInfoByGUIDResponse, nil
}

func (*failedServer) UpdateHostSystemInfoByGUID(_ context.Context, _ *proto.UpdateHostSystemInfoByGUIDRequest) (*proto.UpdateHostSystemInfoByGUIDResponse, error) { //nolint:unused
	return nil, errors.New("failed to update system info")
}

func runMockServer(certFile string, keyFile string) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	creds, _ := credentials.NewServerTLSFromFile(certFile, keyFile)
	s := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterHostmgrServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()
	return lis
}

func runFailedServer(certFile string, keyFile string) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	creds, _ := credentials.NewServerTLSFromFile(certFile, keyFile)
	s := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterHostmgrServer(s, &failedServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()
	return lis
}

// Helper function for dailing to a server using the bufconn package.
func WithBufconnDialer(_ context.Context, lis *bufconn.Listener) func(*comms.Client) {
	return func(s *comms.Client) {
		s.Dialer = grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		})
	}
}

// Mock file read functions for TestGenerateUpdateDeviceRequestSuccess

var testdataPath = "../../test/data"

type file struct {
	name string
}

func (f *file) Name() string             { return f.name }
func (*file) IsDir() bool                { return false }
func (*file) Type() fs.FileMode          { return 0 }
func (*file) Info() (fs.FileInfo, error) { return nil, nil }

func parsedFilePathIntoTestData(path string) (ret string) {
	return strings.ReplaceAll(path, "/", "_")[1:]
}

func mockedReadFile(path string) ([]byte, error) {
	mockedPath := "file-" + parsedFilePathIntoTestData(path)
	return os.ReadFile(filepath.Join(testdataPath, mockedPath))
}

func mockedReadDir(path string) (ret []fs.DirEntry, err error) {
	mockedPath := "dir-" + parsedFilePathIntoTestData(path)

	fileList := make([]fs.DirEntry, 0)
	source, err := os.Open(filepath.Join(testdataPath, mockedPath))
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

func mockedReadlink(path string) (string, error) {
	mockedPath := "link-" + parsedFilePathIntoTestData(path)
	out, err := os.ReadFile(filepath.Join(testdataPath, mockedPath))

	return string(out), err
}

func mockedCollectEthtoolData(_ string) (*tool.EthtoolValues, error) {
	ret := &tool.EthtoolValues{
		LinkState: true,
		SupportedLinkMode: []string{
			"10baseT Full",
			"100baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"100baseT Full",
		},
		CurrentSpeed:  100,
		CurrentDuplex: "Full",
		Features: []string{
			"tx-generic-segmentation",
			"tx-vlan-hw-insert",
			"tx-checksum-ip-generic",
			"rx-vlan-hw-parse",
			"rx-gro",
			"rx-vlan-filter",
			"tx-tcp-segmentation",
			"tx-scatter-gather",
		},
	}
	return ret, nil
}

func mockedStat(path string) (fs.FileInfo, error) {
	mockedPath := "file-" + parsedFilePathIntoTestData(path)
	_, err := os.ReadFile(filepath.Join(testdataPath, mockedPath))
	return nil, err
}

func TestSystemInfoUpdate(t *testing.T) {
	certFile, keyFile, err := testutils.CreateTestCertificates()
	require.NoError(t, err)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)
	ctx := t.Context()
	lis := runMockServer(certFile, keyFile)
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	hostMgr := comms.NewClient("", tlsConfig, WithBufconnDialer(ctx, lis))
	require.NoError(t, hostMgr.Connect())

	resp, err := hostMgr.UpdateHostSystemInfoByGUID(ctx, "dummy_guid", &proto.SystemInfo{})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestFailedSystemInfoUpdate(t *testing.T) {
	certFile, keyFile, err := testutils.CreateTestCertificates()
	require.NoError(t, err)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)
	lis := runFailedServer(certFile, keyFile)
	ctx := t.Context()
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err = testClient.Connect()
	require.NoError(t, err)
	resp, err := testClient.UpdateHostSystemInfoByGUID(ctx, "dummy-token", &proto.SystemInfo{})
	require.Error(t, err)
	assert.Nil(t, resp)
}

func expectedSystemInfoResult(sn string, productName string, bmcAddr string, osInfo *proto.OsInfo, biosInfo *proto.BiosInfo, cpu *proto.SystemCPU, storage []*proto.SystemDisk,
	gpu []*proto.SystemGPU, mem uint64, networks []*proto.SystemNetwork, bmType proto.BmInfo_BmType, usbInfo []*proto.SystemUSB) *proto.SystemInfo {

	return &proto.SystemInfo{
		HwInfo: &proto.HWInfo{
			SerialNum:   sn,
			ProductName: productName,
			Cpu:         cpu,
			Memory:      &proto.SystemMemory{Size: mem},
			Storage:     &proto.Storage{Disk: storage},
			Gpu:         gpu,
			Network:     networks,
			Usb:         usbInfo,
		},
		OsInfo: osInfo,
		BmCtlInfo: &proto.BmInfo{
			BmType: bmType,
			BmcInfo: &proto.BmcInfo{
				BmIp: bmcAddr,
			},
		},
		BiosInfo: biosInfo,
	}
}

func getOsInfo() *proto.OsInfo {
	expectedConfig := []*proto.Config{}
	hwPlatform := &proto.Config{
		Key:   "Platform",
		Value: "x86_64",
	}
	expectedConfig = append(expectedConfig, hwPlatform)
	osType := &proto.Config{
		Key:   "Operating System",
		Value: "GNU/Linux",
	}
	expectedConfig = append(expectedConfig, osType)

	expectedMetadata := []*proto.Metadata{}
	releaseMetadata := &proto.Metadata{
		Key:   "Codename",
		Value: "jammy",
	}
	expectedMetadata = append(expectedMetadata, releaseMetadata)

	return &proto.OsInfo{
		Kernel: &proto.OsKernel{
			Version: "5.15.0-82-generic",
			Config:  expectedConfig,
		},
		Release: &proto.OsRelease{
			Id:       "Ubuntu",
			Version:  "Ubuntu 20.04 LTS",
			Metadata: expectedMetadata,
		},
	}
}

func getBiosInfo() *proto.BiosInfo {
	return &proto.BiosInfo{
		Version:     "1.2.3",
		ReleaseDate: "01/01/2000",
		Vendor:      "Test Vendor",
	}
}

func getCpuInfo() *proto.SystemCPU {
	socket := []*proto.Socket{}
	coreGroupsSocket0 := []*proto.CoreGroup{}
	coreGroupsSocket0 = append(coreGroupsSocket0,
		&proto.CoreGroup{
			CoreType: "P-Core",
			CoreList: []uint32{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46},
		},
		&proto.CoreGroup{
			CoreType: "E-Core",
			CoreList: []uint32{48, 50, 52, 54},
		})
	socket = append(socket, &proto.Socket{
		SocketId:   0,
		CoreGroups: coreGroupsSocket0,
	})

	coreGroupsSocket1 := []*proto.CoreGroup{}
	coreGroupsSocket1 = append(coreGroupsSocket1,
		&proto.CoreGroup{
			CoreType: "P-Core",
			CoreList: []uint32{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39, 41, 43, 45, 47},
		},
		&proto.CoreGroup{
			CoreType: "E-Core",
			CoreList: []uint32{49, 51, 53, 55},
		})
	socket = append(socket, &proto.Socket{
		SocketId:   1,
		CoreGroups: coreGroupsSocket1,
	})

	return &proto.SystemCPU{
		Arch:        "x86_64",
		Sockets:     2,
		Vendor:      "GenuineIntel",
		Model:       "Intel(R) Xeon(R) CPU E5-2687W v4 @ 3.00GHz",
		Cores:       32,
		Threads:     56,
		Features:    []string{"fpu", "vme", "de", "pse", "tsc", "msr", "pae", "mce", "cx8", "apic", "sep", "mtrr", "pge", "mca", "cmov", "pat", "pse36", "clflush", "dts", "acpi", "mmx", "fxsr", "sse", "sse2", "ss", "ht", "tm", "pbe", "syscall", "nx", "pdpe1gb", "rdtscp", "lm", "constant_tsc", "arch_perfmon", "pebs", "bts", "rep_good", "nopl", "xtopology", "nonstop_tsc", "cpuid", "aperfmperf", "pni", "pclmulqdq", "dtes64", "monitor", "ds_cpl", "vmx", "smx", "est", "tm2", "ssse3", "sdbg", "fma", "cx16", "xtpr", "pdcm", "pcid", "dca", "sse4_1", "sse4_2", "x2apic", "movbe", "popcnt", "tsc_deadline_timer", "aes", "xsave", "avx", "f16c", "rdrand", "lahf_lm", "abm", "3dnowprefetch", "cpuid_fault", "epb", "cat_l3", "cdp_l3", "invpcid_single", "pti", "ssbd", "ibrs", "ibpb", "stibp", "tpr_shadow", "vnmi", "flexpriority", "ept", "vpid", "ept_ad", "fsgsbase", "tsc_adjust", "bmi1", "hle", "avx2", "smep", "bmi2", "erms", "invpcid", "rtm", "cqm", "rdt_a", "rdseed", "adx", "smap", "intel_pt", "xsaveopt", "cqm_llc", "cqm_occup_llc", "cqm_mbm_total", "cqm_mbm_local", "dtherm", "ida", "arat", "pln", "pts", "md_clear", "flush_l1d"},
		CpuTopology: &proto.CPUTopology{Sockets: socket},
	}
}

func getStorageInfo() []*proto.SystemDisk {
	result := []*proto.SystemDisk{}
	disk1 := &proto.SystemDisk{
		SerialNumber: "unknown",
		Name:         "nvme0n1p1",
		Vendor:       "unknown",
		Model:        "unknown",
		Size:         1127219200,
		Wwid:         "unknown",
	}
	disk2 := &proto.SystemDisk{
		SerialNumber: "002bb496324e7da81d0018d730708741",
		Name:         "sda",
		Vendor:       "DELL    ",
		Model:        "PERC H730P Mini",
		Size:         399431958528,
		Wwid:         "0x5000c5008e0b3b1d",
	}
	disk3 := &proto.SystemDisk{
		SerialNumber: "CVFT521000J6800CGN",
		Name:         "nvme0n1",
		Vendor:       "unknown",
		Model:        "INTEL SSDPEDMD800G4",
		Size:         800166076416,
		Wwid:         "eui.01000000010000005cd2e43cf16e5451",
	}
	result = append(result, disk1, disk2, disk3)
	return result
}

func getGpuInfo() []*proto.SystemGPU {
	result := []*proto.SystemGPU{}
	gpuDevice := &proto.SystemGPU{
		PciId:       "03:00.0",
		Product:     "Graphics Controller",
		Vendor:      "Graphics",
		Name:        "Graphics Controller",
		Description: "VGA compatible controller",
		Features:    []string{"pm", "vga_controller", "bus_master", "cap_list", "rom", "fb"},
	}
	result = append(result, gpuDevice)
	return result
}

func getNetworkInfo() []*proto.SystemNetwork {
	ipAddresses := []*proto.IPAddress{}
	ipAddr := &proto.IPAddress{
		IpAddress:         "192.168.1.50",
		NetworkPrefixBits: 24,
		ConfigMode:        proto.ConfigMode_CONFIG_MODE_DYNAMIC,
	}
	ipAddresses = append(ipAddresses, ipAddr)

	networkInfo := []*proto.SystemNetwork{}
	network := &proto.SystemNetwork{
		Name:          "ens3",
		PciId:         "0000:00:03.0",
		Mac:           "52:54:00:12:34:56",
		LinkState:     true,
		CurrentSpeed:  100,
		CurrentDuplex: "Full",
		SupportedLinkMode: []string{
			"10baseT Full",
			"100baseT Full",
		},
		AdvertisingLinkMode: []string{
			"10baseT Full",
			"100baseT Full",
		},
		Features: []string{
			"tx-generic-segmentation",
			"tx-vlan-hw-insert",
			"tx-checksum-ip-generic",
			"rx-vlan-hw-parse",
			"rx-gro",
			"rx-vlan-filter",
			"tx-tcp-segmentation",
			"tx-scatter-gather",
		},
		Sriovenabled:  true,
		Sriovnumvfs:   2,
		SriovVfsTotal: 64,
		IpAddresses:   ipAddresses,
		Mtu:           1500,
		BmcNet:        true,
	}
	networkInfo = append(networkInfo, network)
	return networkInfo
}

func getUsbInfo() []*proto.SystemUSB {
	usbInfo := []*proto.SystemUSB{}
	usbInterfaces := []*proto.Interfaces{}
	interfaces := &proto.Interfaces{Class: "Hub"}
	usbInterfaces = append(usbInterfaces, interfaces)
	usbDevice := &proto.SystemUSB{
		Class:       "Hub",
		Idvendor:    "1d6b",
		Idproduct:   "0003",
		Bus:         2,
		Addr:        1,
		Description: "Linux Foundation 3.0 root hub",
		Serial:      "0000:00:14.0",
		Interfaces:  usbInterfaces,
	}
	usbInfo = append(usbInfo, usbDevice)
	return usbInfo
}

func TestGenerateUpdateDeviceRequestErr(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandFailed)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, storage, gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessAllInfo(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassed)
	expected := expectedSystemInfoResult("12A34B5", "Test Product", "192.168.1.50", getOsInfo(), getBiosInfo(), getCpuInfo(), getStorageInfo(), getGpuInfo(), 17179869184, getNetworkInfo(), proto.BmInfo_IPMI, getUsbInfo())
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessStorageOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedStorageOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, getStorageInfo(), gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessOsOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedOsOnly)
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", getOsInfo(), &proto.BiosInfo{}, cpu, storage, gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessBiosOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedBiosOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, getBiosInfo(), cpu, storage, gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessCpuOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedCpuOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, getCpuInfo(), storage, gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessGpuOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedGpuOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, storage, getGpuInfo(), uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessMemoryOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedMemoryOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, storage, gpu, 17179869184, networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessNetworkOnly(t *testing.T) {
	network.ReadFile = mockedReadFile
	network.ReadDir = mockedReadDir
	network.Readlink = mockedReadlink
	network.CollectEthtoolData = mockedCollectEthtoolData
	network.Stat = mockedStat
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedNetworkOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "192.168.1.50", osInfo, &proto.BiosInfo{}, cpu, storage, gpu, uint64(0), getNetworkInfo(), proto.BmInfo_IPMI, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessUsbInfoOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedUsbOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, storage, gpu, uint64(0), networks, proto.BmInfo_NONE, getUsbInfo())
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func TestGenerateUpdateDeviceRequestSuccessDeviceInfoOnly(t *testing.T) {
	json := comms.GenerateSystemInfoRequest(testCmdExecutorCommandPassedDeviceOnly)
	osKern := proto.OsKernel{}
	osRelease := proto.OsRelease{}
	osInfo := &proto.OsInfo{
		Kernel:  &osKern,
		Release: &osRelease,
	}
	cpu := &proto.SystemCPU{}
	storage := []*proto.SystemDisk{}
	gpu := []*proto.SystemGPU{}
	networks := []*proto.SystemNetwork{}
	usbInfo := []*proto.SystemUSB{}
	expected := expectedSystemInfoResult("", "", "", osInfo, &proto.BiosInfo{}, cpu, storage, gpu, uint64(0), networks, proto.BmInfo_NONE, usbInfo)
	require.NotNil(t, json)
	assert.Equal(t, expected, json)
}

func testCmd(testFunc string, command string, args ...string) *exec.Cmd {
	cs := []string{fmt.Sprintf("-test.run=%s", testFunc), "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorCommandPassed(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "lscpu") {
		if len(args) == 0 {
			return testCmd("TestGenerateUpdateDeviceRequestCommandCpuDetails", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandCoreDetails", command, args...)
		}
	} else if strings.Contains(command, "lsblk") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandDiskDetails", command, args...)
	} else if strings.Contains(command, "lspci") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandGpuPciDetails", command, args...)
	} else if strings.Contains(command, "lsmem") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandMemoryDetails", command, args...)
	} else if strings.Contains(command, "ip") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandIpDetails", command, args...)
	} else if strings.Contains(command, "lsusb") {
		if len(args) == 0 {
			return testCmd("TestGenerateUpdateDeviceRequestCommandUsb", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandUsbVerbose", command, args...)
		}
	} else if strings.Contains(command, "uname") {
		if strings.Contains(args[0], "-r") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelVersion", command, args...)
		} else if strings.Contains(args[0], "-i") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelPlatform", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelOperatingSystem", command, args...)
		}
	} else if strings.Contains(command, "lsb_release") {
		if strings.Contains(args[0], "-i") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseId", command, args...)
		} else if strings.Contains(args[0], "-d") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseVersion", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseMetadata", command, args...)
		}
	} else if strings.Contains(command, "sudo") {
		if strings.Contains(args[0], "lshw") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandGpuDetails", command, args...)
		} else if strings.Contains(args[0], "ipmitool") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandIpmiDetails", command, args...)
		} else if strings.Contains(args[0], "rpc") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandDeviceDetails", command, args...)
		} else {
			if strings.Contains(args[2], "bios-version") {
				return testCmd("TestGenerateUpdateDeviceRequestCommandBiosVersion", command, args...)
			} else if strings.Contains(args[2], "bios-release-date") {
				return testCmd("TestGenerateUpdateDeviceRequestCommandBiosReleaseDate", command, args...)
			} else if strings.Contains(args[2], "bios-vendor") {
				return testCmd("TestGenerateUpdateDeviceRequestCommandBiosVendor", command, args...)
			} else if strings.Contains(args[2], "system-product-name") {
				return testCmd("TestGenerateUpdateDeviceRequestCommandSystemProductName", command, args...)
			} else {
				return testCmd("TestGenerateUpdateDeviceRequestCommandSystemSerialNumber", command, args...)
			}
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedStorageOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "lsblk") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandDiskDetails", command, args...)
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedOsOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "uname") {
		if strings.Contains(args[0], "-r") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelVersion", command, args...)
		} else if strings.Contains(args[0], "-i") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelPlatform", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandOsKernelOperatingSystem", command, args...)
		}
	} else if strings.Contains(command, "lsb_release") {
		if strings.Contains(args[0], "-i") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseId", command, args...)
		} else if strings.Contains(args[0], "-d") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseVersion", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandReleaseMetadata", command, args...)
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedBiosOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "sudo") && strings.Contains(args[0], "dmidecode") {
		if strings.Contains(args[2], "bios-version") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandBiosVersion", command, args...)
		} else if strings.Contains(args[2], "bios-release-date") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandBiosReleaseDate", command, args...)
		} else if strings.Contains(args[2], "bios-vendor") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandBiosVendor", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedCpuOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "lscpu") {
		if len(args) == 0 {
			return testCmd("TestGenerateUpdateDeviceRequestCommandCpuDetails", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandCoreDetails", command, args...)
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedGpuOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "sudo") && strings.Contains(args[0], "lshw") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandGpuDetails", command, args...)
	} else if strings.Contains(command, "lspci") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandGpuPciDetails", command, args...)
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedMemoryOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "lsmem") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandMemoryDetails", command, args...)
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedNetworkOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "sudo") && strings.Contains(args[0], "ipmitool") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandIpmiDetails", command, args...)
	} else if strings.Contains(command, "ip") {
		return testCmd("TestGenerateUpdateDeviceRequestCommandIpDetails", command, args...)
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedUsbOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "lsusb") {
		if len(args) == 0 {
			return testCmd("TestGenerateUpdateDeviceRequestCommandUsb", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandUsbVerbose", command, args...)
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandPassedDeviceOnly(command string, args ...string) *exec.Cmd {
	if strings.Contains(command, "sudo") {
		if strings.Contains(args[0], "rpc") {
			return testCmd("TestGenerateUpdateDeviceRequestCommandDeviceDetails", command, args...)
		} else {
			return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
		}
	} else {
		return testCmd("TestGenerateUpdateDeviceRequestCommandFailed", command, args...)
	}
}

func testCmdExecutorCommandFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestGenerateUpdateDeviceRequestCommandFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestGenerateUpdateDeviceRequestCommandCpuDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandCoreDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandDeviceDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_amtinfo.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandDiskDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_disks.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandGpuDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandGpuPciDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_name.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandMemoryDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_memory.json")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandIpmiDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ipmitool.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandIpDetails(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_ip_addr.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandBiosVersion(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "1.2.3")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandBiosReleaseDate(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "01/01/2000")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandBiosVendor(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "Test Vendor")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandSystemProductName(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "Test Product")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandSystemSerialNumber(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "12A34B5")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandOsKernelVersion(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "5.15.0-82-generic")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandOsKernelPlatform(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "x86_64")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandOsKernelOperatingSystem(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "GNU/Linux")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandReleaseId(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "Distributor ID: Ubuntu")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandReleaseVersion(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "Description:    Ubuntu 20.04 LTS")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandReleaseMetadata(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%v", "Codename:       jammy")
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandUsb(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandUsbVerbose(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_usb_verbose.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGenerateUpdateDeviceRequestCommandFailed(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}
