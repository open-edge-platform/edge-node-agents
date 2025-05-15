// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package gpu_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/gpu"
)

var testPciAddr1 = "03:00.0"
var testPciAddr2 = "57:00.0"
var testPciAddr3 = "5b:00.0"
var testProduct1 = "Graphics Controller"
var testProduct2 = "Graphics Card"
var testProduct3 = "Graphics Card"
var testVendor1 = "Graphics"
var testVendor2 = "PCI"
var testVendor3 = "PCI"
var testName = "Graphics Controller"
var testDescription1 = "VGA compatible controller"
var testDescription2 = "PCI graphics card"
var testDescription3 = "PCI graphics card"
var testFeatures1 = []string{"pm", "vga_controller", "bus_master", "cap_list", "rom", "fb"}
var testFeatures2 = []string{"pciexpress", "msi", "pm", "bus_master", "cap_list"}
var testFeatures3 = []string{"pciexpress", "msi", "pm", "bus_master", "cap_list"}
var testNotFoundMsg = "Not Available"
var testNameNotFound = "Info Not Available"

func expectedOutput(expect []*gpu.Gpu, pci, prod, vendor, name, desc string, features []string) []*gpu.Gpu {
	return append(expect, &gpu.Gpu{
		PciID:       pci,
		Product:     prod,
		Vendor:      vendor,
		Name:        name,
		Description: desc,
		Features:    features,
	})
}

func TestGetGpuList(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorSuccess)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testName, testDescription1, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListFailed(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorLshwFailed)
	require.Error(t, err)
	assert.Equal(t, []*gpu.Gpu{}, out)
}

func TestGetGpuListLspciFailed(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorLspciFailed)
	require.Error(t, err)
	assert.Equal(t, []*gpu.Gpu{}, out)
}

func TestGetGpuListMultiDevicesSuccess(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorMultiDevSuccess)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testName, testDescription1, testFeatures1)
	expect = expectedOutput(expect, testPciAddr2, testProduct2,
		testVendor2, testName, testDescription2, testFeatures2)
	expect = expectedOutput(expect, testPciAddr3, testProduct3,
		testVendor3, testName, testDescription3, testFeatures3)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyPciInfo(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyPciInfo)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testNotFoundMsg, testProduct1,
		testVendor1, testNameNotFound, testDescription1, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyProductName(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyProductName)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testNotFoundMsg,
		testVendor1, testName, testDescription1, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyVendor(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyVendorName)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testNotFoundMsg, testName, testDescription1, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyDescription(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyDescription)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testName, testNotFoundMsg, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyFeatures(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyFeatures)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testName, testDescription1, []string{testNotFoundMsg})
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListEmptyDeviceName(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorEmptyDeviceName)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testNameNotFound, testDescription1, testFeatures1)
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func testCmdReceived(args ...string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "lshw") {
			return true
		}
	}
	return false
}

func testCmdExecutorSuccess(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionLshwSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorLshwFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestGpuListExecutionFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorLspciFailed(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionLshwSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorMultiDevSuccess(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionMultiDevSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyPciInfo(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestGpuListExecutionEmptyPciInfo", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyProductName(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionEmptyProductName", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyVendorName(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionEmptyVendor", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyDescription(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionEmptyDescription", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyFeatures(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionEmptyFeatures", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdExecutorEmptyDeviceName(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionLshwSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}

	cs := []string{"-test.run=TestGpuListExecutionLspciNoDeviceName", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestGpuListExecutionLshwSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionLspciSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_name.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionFailed(_ *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "failed to execute command")
	os.Exit(1)
}

func TestGpuListExecutionMultiDevSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_multi_gpu.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionEmptyPciInfo(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_empty_pci_info.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionEmptyProductName(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_empty_product.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionEmptyVendor(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_empty_vendor.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionEmptyDescription(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_empty_description.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionEmptyFeatures(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_empty_features.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionLspciNoDeviceName(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_no_name.txt")
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
