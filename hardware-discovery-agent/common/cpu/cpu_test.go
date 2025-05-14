// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cpu_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/cpu"
	"github.com/stretchr/testify/assert"
)

var features []string = []string{"fpu", "vme", "de", "pse", "tsc", "msr", "pae", "mce", "cx8", "apic", "sep", "mtrr", "pge", "mca", "cmov", "pat", "pse36", "clflush", "dts", "acpi", "mmx", "fxsr", "sse", "sse2", "ss", "ht", "tm", "pbe", "syscall", "nx", "pdpe1gb", "rdtscp", "lm", "constant_tsc", "arch_perfmon", "pebs", "bts", "rep_good", "nopl", "xtopology", "nonstop_tsc", "cpuid", "aperfmperf", "pni", "pclmulqdq", "dtes64", "monitor", "ds_cpl", "vmx", "smx", "est", "tm2", "ssse3", "sdbg", "fma", "cx16", "xtpr", "pdcm", "pcid", "dca", "sse4_1", "sse4_2", "x2apic", "movbe", "popcnt", "tsc_deadline_timer", "aes", "xsave", "avx", "f16c", "rdrand", "lahf_lm", "abm", "3dnowprefetch", "cpuid_fault", "epb", "cat_l3", "cdp_l3", "invpcid_single", "pti", "ssbd", "ibrs", "ibpb", "stibp", "tpr_shadow", "vnmi", "flexpriority", "ept", "vpid", "ept_ad", "fsgsbase", "tsc_adjust", "bmi1", "hle", "avx2", "smep", "bmi2", "erms", "invpcid", "rtm", "cqm", "rdt_a", "rdseed", "adx", "smap", "intel_pt", "xsaveopt", "cqm_llc", "cqm_occup_llc", "cqm_mbm_total", "cqm_mbm_local", "dtherm", "ida", "arat", "pln", "pts", "md_clear", "flush_l1d"}

var pcoresMultiSocket0 []uint32 = []uint32{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46}
var pcoresMultiSocket1 []uint32 = []uint32{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39, 41, 43, 45, 47}
var pcoresMultiSocket0NoHt []uint32 = []uint32{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22}
var pcoresMultiSocket1NoHt []uint32 = []uint32{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23}
var ecoresMultiSocket0 []uint32 = []uint32{48, 50, 52, 54}
var ecoresMultiSocket1 []uint32 = []uint32{49, 51, 53, 55}
var ecoresMultiSocket0NoHt []uint32 = []uint32{24, 26, 28, 30}
var ecoresMultiSocket1NoHt []uint32 = []uint32{25, 27, 29, 31}

var pcoresSingleSocket []uint32 = []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
var pcoresSingleSocketNoHt []uint32 = []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
var pcoresSingleSocketSameCoreCount []uint32 = []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
var pcoresSingleSocketSameCoreCountNoThreads []uint32 = []uint32{0, 1, 2, 3, 4, 5, 6, 7}
var ecoresSingleSocket []uint32 = []uint32{24, 25, 26, 27, 28, 29, 30, 31}
var ecoresSingleSocketNoHt []uint32 = []uint32{12, 13, 14, 15, 16, 17, 18, 19}
var ecoresSingleSocketSameCoreCount []uint32 = []uint32{16, 17, 18, 19, 20, 21, 22, 23}
var ecoresSingleSocketSameCoreCountNoThreads []uint32 = []uint32{8, 9, 10, 11, 12, 13, 14, 15}

var pcoresMultiSocketInvalidSocketId []uint32 = []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
var pcoresMultiSocketInvalidPCoreId0 []uint32 = []uint32{0, 4, 8, 12, 14, 16, 18, 20, 22}
var pcoresMultiSocketInvalidPCoreId1 []uint32 = []uint32{1, 5, 9, 13, 15, 17, 19, 21, 23}
var ecoresMultiSocketInvalidSocketId []uint32 = []uint32{24, 25, 26, 27, 28, 29, 30, 31}
var ecoresMultiSocketInvalidECoreId0 []uint32 = []uint32{28, 30}
var ecoresMultiSocketInvalidECoreId1 []uint32 = []uint32{29, 31}

func getSocketResult(socketId uint32, pCores, eCores []uint32) *cpu.Socket {
	coreGroups := []*cpu.CoreGroup{}
	coreGroups = append(coreGroups, &cpu.CoreGroup{
		Type: "P-Core",
		List: pCores,
	})
	if len(eCores) > 0 {
		coreGroups = append(coreGroups, &cpu.CoreGroup{
			Type: "E-Core",
			List: eCores,
		})
	}

	return &cpu.Socket{SocketId: socketId, CoreGroups: coreGroups}
}

func getExpectedResultMultiSocket(cores, threads uint32, pCoresSocket0, pCoresSocket1, eCoresSocket0, eCoresSocket1 []uint32) *cpu.Cpu {
	socket := []*cpu.Socket{}
	socket0 := getSocketResult(0, pCoresSocket0, eCoresSocket0)
	socket1 := getSocketResult(1, pCoresSocket1, eCoresSocket1)
	socket = append(socket, socket0)
	socket = append(socket, socket1)
	expected := &cpu.Cpu{
		Arch:     "x86_64",
		Sockets:  2,
		Vendor:   "GenuineIntel",
		Model:    "Intel(R) Xeon(R) CPU E5-2687W v4 @ 3.00GHz",
		Cores:    cores,
		Threads:  threads,
		Features: features,
		Topology: &cpu.CpuTopology{Sockets: socket},
	}
	return expected
}

func getExpectedResultSingleSocket(cores, threads uint32, pCores, eCores []uint32) *cpu.Cpu {
	socket := []*cpu.Socket{}
	socket0 := getSocketResult(0, pCores, eCores)
	socket = append(socket, socket0)
	expected := &cpu.Cpu{
		Arch:     "x86_64",
		Sockets:  1,
		Vendor:   "GenuineIntel",
		Model:    "Intel(R) Xeon(R) CPU E5-2687W v4 @ 3.00GHz",
		Cores:    cores,
		Threads:  threads,
		Features: features,
		Topology: &cpu.CpuTopology{Sockets: socket},
	}
	return expected
}

func getExpectedResultFailure(sockets, cores, threads uint32) *cpu.Cpu {
	expected := &cpu.Cpu{
		Arch:     "x86_64",
		Sockets:  sockets,
		Vendor:   "GenuineIntel",
		Model:    "Intel(R) Xeon(R) CPU E5-2687W v4 @ 3.00GHz",
		Cores:    cores,
		Threads:  threads,
		Features: features,
	}
	return expected
}

func Test_GetCpuList(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccess)
	expected := getExpectedResultMultiSocket(32, 56, pcoresMultiSocket0, pcoresMultiSocket1, ecoresMultiSocket0, ecoresMultiSocket1)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListNoThreads(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoHT)
	expected := getExpectedResultMultiSocket(32, 32, pcoresMultiSocket0NoHt, pcoresMultiSocket1NoHt, ecoresMultiSocket0NoHt, ecoresMultiSocket1NoHt)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListNoEcores(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoEcores)
	expected := getExpectedResultMultiSocket(24, 48, pcoresMultiSocket0, pcoresMultiSocket1, []uint32{}, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListNoEcoresNoThreads(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoEcoresNoHT)
	expected := getExpectedResultMultiSocket(24, 24, pcoresMultiSocket0NoHt, pcoresMultiSocket1NoHt, []uint32{}, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocket(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessOneSocket)
	expected := getExpectedResultSingleSocket(20, 32, pcoresSingleSocket, ecoresSingleSocket)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocketNoThreads(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoHTOneSocket)
	expected := getExpectedResultSingleSocket(20, 20, pcoresSingleSocketNoHt, ecoresSingleSocketNoHt)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocketNoEcores(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoEcoresOneSocket)
	expected := getExpectedResultSingleSocket(12, 24, pcoresSingleSocket, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocketNoEcoresNoThreads(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessNoEcoresNoHTOneSocket)
	expected := getExpectedResultSingleSocket(12, 12, pcoresSingleSocketNoHt, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocketSamePcoreEcoreCount(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessSamePcoreEcoreCount)
	expected := getExpectedResultSingleSocket(16, 24, pcoresSingleSocketSameCoreCount, ecoresSingleSocketSameCoreCount)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListSingleSocketSamePcoreEcoreCountNoThreads(t *testing.T) {
	out, err := cpu.GetCpuList(testSuccessSamePcoreEcoreCountNoHT)
	expected := getExpectedResultSingleSocket(16, 16, pcoresSingleSocketSameCoreCountNoThreads, ecoresSingleSocketSameCoreCountNoThreads)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListFailedFirstLscpu(t *testing.T) {
	out, err := cpu.GetCpuList(testFailureFirstLscpu)
	assert.Equal(t, &cpu.Cpu{}, out)
	assert.Error(t, err)
}

func Test_GetCpuListFailedSecondLscpu(t *testing.T) {
	out, err := cpu.GetCpuList(testFailureSecondLscpu)
	expected := getExpectedResultFailure(2, 32, 56)
	assert.Equal(t, expected, out)
	assert.Error(t, err)
}

func Test_GetCpuListInvalidSocket(t *testing.T) {
	out, err := cpu.GetCpuList(testFailureSocketParse)
	expected := getExpectedResultFailure(0, 0, 56)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidCpus(t *testing.T) {
	out, err := cpu.GetCpuList(testInvalidCpuParse)
	expected := getExpectedResultMultiSocket(32, 0, pcoresMultiSocket0NoHt, pcoresMultiSocket1NoHt, ecoresMultiSocket0NoHt, ecoresMultiSocket1NoHt)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidCoreCount(t *testing.T) {
	out, err := cpu.GetCpuList(testInvalidCoreParse)
	expected := getExpectedResultMultiSocket(0, 56, pcoresMultiSocket0NoHt, pcoresMultiSocket1NoHt, ecoresMultiSocket0NoHt, ecoresMultiSocket1NoHt)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidSocketId(t *testing.T) {
	out, err := cpu.GetCpuList(testSocketIdParse)
	expected := getExpectedResultMultiSocket(32, 32, pcoresMultiSocketInvalidSocketId, []uint32{}, ecoresMultiSocketInvalidSocketId, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidPCoreId(t *testing.T) {
	out, err := cpu.GetCpuList(testPCoreIdParse)
	expected := getExpectedResultMultiSocket(32, 32, pcoresMultiSocketInvalidPCoreId0, pcoresMultiSocketInvalidPCoreId1, ecoresMultiSocket0NoHt, ecoresMultiSocket1NoHt)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidECoreId(t *testing.T) {
	out, err := cpu.GetCpuList(testECoreIdParse)
	expected := getExpectedResultMultiSocket(32, 32, pcoresMultiSocket0NoHt, pcoresMultiSocket1NoHt, ecoresMultiSocketInvalidECoreId0, ecoresMultiSocketInvalidECoreId1)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListNoMaxMhzDetected(t *testing.T) {
	out, err := cpu.GetCpuList(testNoMaxMhzParse)
	expected := getExpectedResultSingleSocket(12, 24, pcoresSingleSocket, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func Test_GetCpuListInvalidMaxMhzDetected(t *testing.T) {
	out, err := cpu.GetCpuList(testInvalidMaxMhzParse)
	expected := getExpectedResultSingleSocket(12, 24, []uint32{}, []uint32{})
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}

func testLscpuFlagsReceived(args ...string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "--extended") {
			return true
		}
	}
	return false
}

func testCmd(testFunc string, command string, args ...string) *exec.Cmd {
	cs := []string{fmt.Sprintf("-test.run=%s", testFunc), "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testLscpuCmd(firstTestFunc string, secondTestFunc string, command string, args ...string) *exec.Cmd {
	if !testLscpuFlagsReceived(args...) {
		return testCmd(firstTestFunc, command, args...)
	} else {
		return testCmd(secondTestFunc, command, args...)
	}
}

func testSuccess(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpu", "TestSuccessSecondLscpu", command, args...)
}

func testSuccessNoHT(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoHT", "TestSuccessSecondLscpuNoHT", command, args...)
}

func testSuccessNoEcores(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoEcores", "TestSuccessSecondLscpuNoEcores", command, args...)
}

func testSuccessNoEcoresNoHT(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoEcoresNoHT", "TestSuccessSecondLscpuNoEcoresNoHT", command, args...)
}

func testSuccessOneSocket(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuOneSocket", "TestSuccessSecondLscpuOneSocket", command, args...)
}

func testSuccessNoHTOneSocket(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoHTOneSocket", "TestSuccessSecondLscpuNoHTOneSocket", command, args...)
}

func testSuccessNoEcoresOneSocket(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoEcoresOneSocket", "TestSuccessSecondLscpuNoEcoresOneSocket", command, args...)
}

func testSuccessNoEcoresNoHTOneSocket(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoEcoresNoHTOneSocket", "TestSuccessSecondLscpuNoEcoresNoHTOneSocket", command, args...)
}

func testSuccessSamePcoreEcoreCount(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuSamePcoreEcoreCount", "TestSuccessSecondLscpuSamePcoreEcoreCount", command, args...)
}

func testSuccessSamePcoreEcoreCountNoHT(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuSamePcoreEcoreCountNoHT", "TestSuccessSecondLscpuSamePcoreEcoreCountNoHT", command, args...)
}

func testFailureFirstLscpu(command string, args ...string) *exec.Cmd {
	return testCmd("TestFailureCommand", command, args...)
}

func testFailureSecondLscpu(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpu", "TestFailureCommand", command, args...)
}

func testFailureSocketParse(command string, args ...string) *exec.Cmd {
	return testCmd("TestFailureSocketParse", command, args...)
}

func testInvalidCpuParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestInvalidCpuParse", "TestSuccessSecondLscpuNoHT", command, args...)
}

func testInvalidCoreParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestInvalidCoreParse", "TestSuccessSecondLscpuNoHT", command, args...)
}

func testSocketIdParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoHT", "TestSuccessLscpuInvalidSocketId", command, args...)
}

func testPCoreIdParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoHT", "TestSuccessLscpuInvalidPCoreId", command, args...)
}

func testECoreIdParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoHT", "TestSuccessLscpuInvalidECoreId", command, args...)
}

func testNoMaxMhzParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoMaxMhz", "TestSuccessSecondLscpuNoMaxMhz", command, args...)
}

func testInvalidMaxMhzParse(command string, args ...string) *exec.Cmd {
	return testLscpuCmd("TestSuccessFirstLscpuNoMaxMhz", "TestSuccessSecondLscpuInvalidMaxMhz", command, args...)
}

func TestSuccessFirstLscpu(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoEcores(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noecores.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoEcoresNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noecores_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoHTOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noHT_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoEcoresOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noecores_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoEcoresNoHTOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_noecores_noHT_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuSamePcoreEcoreCount(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_same_pcore_ecore_count.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuSamePcoreEcoreCountNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_same_pcore_ecore_count_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessFirstLscpuNoMaxMhz(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_no_max_mhz.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpu(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoEcores(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noecores.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoEcoresNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noecores_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoHTOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noHT_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoEcoresOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noecores_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoEcoresNoHTOneSocket(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_noecores_noHT_onesocket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuSamePcoreEcoreCount(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_same_pcore_ecore_count.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuSamePcoreEcoreCountNoHT(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_same_pcore_ecore_count_noHT.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuNoMaxMhz(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_details_no_max_mhz.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestFailureCommand(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "failed to execute command")
	os.Exit(1)
}

func TestFailureSocketParse(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_socket.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestInvalidCpuParse(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_cpu.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestInvalidCoreParse(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_core.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessLscpuInvalidSocketId(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_socket_id.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessLscpuInvalidPCoreId(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_pcore_id.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessLscpuInvalidECoreId(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_ecore_id.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestSuccessSecondLscpuInvalidMaxMhz(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_cpu_invalid_max_mhz.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
