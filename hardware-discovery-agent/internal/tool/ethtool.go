// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package tool

import (
	"sort"

	"github.com/safchain/ethtool"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/logger"
)

var ETHTOOL_LINK_MODE_MAP = map[int]string{
	0x001:              "10baseT Half",
	0x002:              "10baseT Full",
	0x004:              "100baseT Half",
	0x008:              "100baseT Full",
	0x010:              "1000baseT Half (not supported by IEEE standards)",
	0x020:              "1000baseT Full",
	0x20000:            "1000baseKX Full",
	0x20000000000:      "1000baseX Full",
	0x800000000000:     "2500baseT Full",
	0x8000:             "2500baseX Full (not supported by IEEE standards)",
	0x1000000000000:    "5000baseT Full",
	0x1000:             "10000baseT Full",
	0x40000:            "10000baseKX4 Full",
	0x80000:            "10000baseKR Full",
	0x100000:           "10000baseR_FEC",
	0x40000000000:      "10000baseCR  Full",
	0x80000000000:      "10000baseSR  Full",
	0x100000000000:     "10000baseLR  Full",
	0x200000000000:     "10000baseLRM Full",
	0x400000000000:     "10000baseER  Full",
	0x200000:           "20000baseMLD2 Full (not supported by IEEE standards)",
	0x400000:           "20000baseKR2 Full (not supported by IEEE standards)",
	0x80000000:         "25000baseCR Full",
	0x100000000:        "25000baseKR Full",
	0x200000000:        "25000baseSR Full",
	0x800000:           "40000baseKR4 Full",
	0x1000000:          "40000baseCR4 Full",
	0x2000000:          "40000baseSR4 Full",
	0x4000000:          "40000baseLR4 Full",
	0x400000000:        "50000baseCR2 Full",
	0x800000000:        "50000baseKR2 Full",
	0x10000000000:      "50000baseSR2 Full",
	0x10000000000000:   "50000baseKR Full",
	0x20000000000000:   "50000baseSR Full",
	0x40000000000000:   "50000baseCR Full",
	0x80000000000000:   "50000baseLR_ER_FR Full",
	0x100000000000000:  "50000baseDR Full",
	0x8000000:          "56000baseKR4 Full",
	0x10000000:         "56000baseCR4 Full",
	0x20000000:         "56000baseSR4 Full",
	0x40000000:         "56000baseLR4 Full",
	0x1000000000:       "100000baseKR4 Full",
	0x2000000000:       "100000baseSR4 Full",
	0x4000000000:       "100000baseCR4 Full",
	0x8000000000:       "100000baseLR4_ER4 Full",
	0x200000000000000:  "100000baseKR2 Full",
	0x400000000000000:  "100000baseSR2 Full",
	0x800000000000000:  "100000baseCR2 Full",
	0x1000000000000000: "100000baseLR2_ER2_FR2 Full",
	0x2000000000000000: "100000baseDR2 Full",
	0x4000000000000000: "200000baseKR4 Full",
}

var log = logger.Logger

type EthtoolValues struct {
	LinkState           bool
	SupportedLinkMode   []string
	AdvertisingLinkMode []string
	CurrentSpeed        uint64
	CurrentDuplex       string
	Features            []string
}

func ParseLinkModeCode(linkMode uint64) []string {
	modes := make([]string, 0)
	for k, v := range ETHTOOL_LINK_MODE_MAP {
		if linkMode&uint64(k) != 0 {
			modes = append(modes, v)
		}
	}
	sort.Strings(modes)
	return modes
}

func ParseDuplexMode(duplex uint64) string {
	switch duplex {
	case 0:
		return "Half"
	case 1:
		return "Full"
	}
	return "Unknown"
}

func ParseLinkState(linkstate uint32) bool {
	switch linkstate {
	case 0:
		return false
	case 1:
		return true
	}
	return false
}

func ParseFeatureList(features map[string]bool) []string {
	featureList := make([]string, 0)
	for k, v := range features {
		if v {
			featureList = append(featureList, k)
		}
	}
	sort.Strings(featureList)
	return featureList
}

func CollectEthtoolData(nicName string) (*EthtoolValues, error) {

	// ethHandle needs to be initialized every time on every interface
	ethHandle, ethHandleErr := ethtool.NewEthtool()
	if ethHandleErr != nil {
		log.Errorf("ethtool initialization error")
		return nil, ethHandleErr
	}
	defer ethHandle.Close()

	// Retrieving the linkstate information from ioctl
	linkstate, err := ethHandle.LinkState(nicName)
	if err != nil {
		log.Errorf("Collecting ethtool linkstate for %s failed\n : %v", nicName, err)
		return nil, err
	}

	// Retrieving the features information from ioctl
	features, err := ethHandle.Features(nicName)
	if err != nil {
		log.Errorf("Collecting ethtool features for %s failed\n : %v", nicName, err)
		return nil, err
	}

	cmd, err := ethHandle.CmdGetMapped(nicName)
	if err != nil {
		log.Errorf("Collecting ethtool cmd for %s failed\n : %v", nicName, err)
		return nil, err
	}

	var currentSpeed int
	if ParseLinkState(linkstate) {
		// The speed value is only valid when the link is up
		currentSpeed = int(cmd["Speed"])
	} else {
		currentSpeed = 0
	}

	var currentDuplex string
	if ParseLinkState(linkstate) {
		// The duplex value is only valid when the link is up
		currentDuplex = ParseDuplexMode(cmd["Duplex"])
	} else {
		currentDuplex = "not applicable"
	}

	return &EthtoolValues{
		LinkState:           ParseLinkState(linkstate),
		SupportedLinkMode:   ParseLinkModeCode(cmd["Supported"]),
		AdvertisingLinkMode: ParseLinkModeCode(cmd["Advertising"]),
		CurrentSpeed:        uint64(currentSpeed),
		CurrentDuplex:       currentDuplex,
		Features:            ParseFeatureList(features),
	}, nil
}
