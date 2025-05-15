// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package usb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/common/utils"
)

type Interface struct {
	Class string
}

type Usb struct {
	Class       string
	VendorID    string
	ProductID   string
	Bus         uint32
	Address     uint32
	Description string
	Serial      string
	Interfaces  []*Interface
}

func GetUsbList(executor utils.CmdExecutor) ([]*Usb, error) {
	usbDevList, err := utils.ReadFromCommand(executor, "lsusb")
	if err != nil {
		return []*Usb{}, fmt.Errorf("failed to read data from command; error: %w", err)
	}
	parseUsbDevList := strings.SplitAfter(string(usbDevList), "\n")

	usbList := []*Usb{}
	for _, usbDevData := range parseUsbDevList {
		if !strings.Contains(usbDevData, "Bus") {
			continue
		}

		var usb Usb
		usbDeviceDetails := strings.SplitAfter(usbDevData, " ")
		usbDeviceID := strings.Split(usbDeviceDetails[3], ":")
		usbBusInfo := strings.Split(usbDeviceDetails[1], " ")

		usbDevAddr := usbBusInfo[0] + ":" + usbDeviceID[0]
		usbDeviceInfo, err := utils.ReadFromCommand(executor, "lsusb", "-v", "-s", usbDevAddr)
		if err != nil {
			return []*Usb{}, fmt.Errorf("failed to read data from command; error: %w", err)
		}
		parseUsbDeviceInfo := string(usbDeviceInfo)

		usb.Class = getDeviceClass(parseUsbDeviceInfo)
		usb.VendorID = getID(parseUsbDeviceInfo, "idVendor")
		usb.ProductID = getID(parseUsbDeviceInfo, "idProduct")

		usb.Bus, err = getAddr(usbBusInfo[0])
		if err != nil {
			return []*Usb{}, fmt.Errorf("failed to read data from command; error: %w", err)
		}

		usb.Address, err = getAddr(usbDeviceID[0])
		if err != nil {
			return []*Usb{}, fmt.Errorf("failed to read data from command; error: %w", err)
		}

		usb.Description = getDeviceDescription(usbDevData)
		usb.Serial = getSerial(parseUsbDeviceInfo)
		usb.Interfaces = getInterfaces(parseUsbDeviceInfo)

		usbList = append(usbList, &usb)
	}

	return usbList, nil
}

func getDeviceClass(usbDeviceInfo string) string {
	devClass := strings.SplitAfter(usbDeviceInfo, "bDeviceClass")
	device := strings.Split(devClass[1], "\n")
	dev := strings.SplitAfter(device[0], " ")
	devLen := len(dev)
	return dev[devLen-1]
}

func getID(usbDeviceInfo string, idType string) string {
	usbDeviceID := strings.SplitAfter(usbDeviceInfo, idType)
	deviceID := strings.Split(usbDeviceID[1], "\n")
	devID := strings.SplitAfter(deviceID[0], "0x")
	id := strings.Split(devID[1], " ")
	return id[0]
}

func getAddr(usbAddrInfo string) (uint32, error) {
	addr, err := strconv.ParseUint(usbAddrInfo, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint32(addr), err
}

func getDeviceDescription(usbDeviceInfo string) string {
	deviceDescription := strings.Split(usbDeviceInfo, "\n")
	devDescription := strings.SplitAfter(deviceDescription[0], ":")
	description := strings.SplitAfter(devDescription[2], " ")
	return strings.Join(description[1:], "")
}

func getSerial(usbDeviceInfo string) string {
	serialData := strings.SplitAfter(usbDeviceInfo, "iSerial")
	serData := strings.Split(serialData[1], "\n")
	serial := strings.SplitAfter(serData[0], ":")
	serialInfo := strings.Split(serial[0], ":")
	if len(serialInfo) > 1 {
		deviceSerial := strings.SplitAfter(serialInfo[0], " ")
		length := len(deviceSerial)
		return deviceSerial[length-1] + ":" + serial[1] + serial[2]
	}
	return "Not available"
}

func getInterfaces(usbDeviceInfo string) []*Interface {
	interfaces := []*Interface{}
	var iface Interface

	interfaceClass := strings.SplitAfter(usbDeviceInfo, "bInterfaceClass         ")
	for interfaceLen, interfaceData := range interfaceClass {
		if interfaceLen == 0 {
			continue
		}

		ifaceClass := strings.Split(interfaceData, "\n")
		iClass := strings.SplitAfter(ifaceClass[0], " ")
		iface.Class = strings.Join(iClass[1:], "")
		interfaces = append(interfaces, &iface)
	}
	return interfaces
}
