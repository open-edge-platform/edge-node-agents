// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package gpu_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/gpu"
	"github.com/stretchr/testify/assert"
)

var testPciAddr1 string = "03:00.0"
var testPciAddr2 string = "57:00.0"
var testPciAddr3 string = "5b:00.0"
var testProduct1 string = "Graphics Controller"
var testProduct2 string = "Graphics Card"
var testProduct3 string = "Graphics Card"
var testVendor1 string = "Graphics"
var testVendor2 string = "PCI"
var testVendor3 string = "PCI"
var testDescription1 string = "VGA compatible controller"
var testDescription2 string = "PCI graphics card"
var testDescription3 string = "PCI graphics card"
var testFeatures1 []string = []string{"pm", "vga_controller", "bus_master", "cap_list", "rom", "fb"}
var testFeatures2 []string = []string{"pciexpress", "msi", "pm", "bus_master", "cap_list"}
var testFeatures3 []string = []string{"pciexpress", "msi", "pm", "bus_master", "cap_list"}

func expectedOutput(expect []*gpu.Gpu, pci, prod, vendor, desc string, features []string) []*gpu.Gpu {
	return append(expect, &gpu.Gpu{
		PciId:       pci,
		Product:     prod,
		Vendor:      vendor,
		Name:        "Graphics Controller",
		Description: desc,
		Features:    features,
	})
}

func TestGetGpuList(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorSuccess)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testDescription1, testFeatures1)
	assert.Nil(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, expect, out)
}

func TestGetGpuListFailed(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorLshwFailed)
	assert.NotNil(t, err)
	assert.Equal(t, []*gpu.Gpu{}, out)
}

func TestGetCpuListLspciFailed(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorLspciFailed)
	assert.NotNil(t, err)
	assert.Equal(t, []*gpu.Gpu{}, out)
}

func TestGetGpuListMultiDevicesSuccess(t *testing.T) {
	out, err := gpu.GetGpuList(testCmdExecutorMultiDevSuccess)
	expect := []*gpu.Gpu{}
	expect = expectedOutput(expect, testPciAddr1, testProduct1,
		testVendor1, testDescription1, testFeatures1)
	expect = expectedOutput(expect, testPciAddr2, testProduct2,
		testVendor2, testDescription2, testFeatures2)
	expect = expectedOutput(expect, testPciAddr3, testProduct3,
		testVendor3, testDescription3, testFeatures3)
	assert.Nil(t, err)
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
	} else {
		cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
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
	} else {
		cs := []string{"-test.run=TestGpuListExecutionFailed", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func testCmdExecutorMultiDevSuccess(command string, args ...string) *exec.Cmd {
	if testCmdReceived(args...) {
		cs := []string{"-test.run=TestGpuListExecutionMultiDevSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	} else {
		cs := []string{"-test.run=TestGpuListExecutionLspciSuccess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_PROCESS=1"}
		return cmd
	}
}

func TestGpuListExecutionLshwSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionLspciSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	testData, err := os.ReadFile("../../test/data/mock_gpu_name.txt")
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}

func TestGpuListExecutionFailed(t *testing.T) {
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
	if err != nil {
		log.Fatal()
	}
	fmt.Fprintf(os.Stdout, "%v", string(testData))
	os.Exit(0)
}
