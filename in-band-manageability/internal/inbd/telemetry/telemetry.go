/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryHandler handles different types of telemetry queries
type QueryHandler struct{}

// NewQueryHandler creates a new query handler
func NewQueryHandler() *QueryHandler {
	return &QueryHandler{}
}

// HandleQuery processes query requests and returns appropriate data
func (q *QueryHandler) HandleQuery(option string) (*pb.QueryData, error) {
	timestamp := timestamppb.New(time.Now())

	switch option {
	case "hw", "hardware":
		hw, err := GetHardwareInfo()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "hardware",
			Timestamp: timestamp,
			Values:    &pb.QueryData_Hardware{Hardware: hw},
		}, nil

	case "fw", "firmware":
		fw, err := GetFirmwareInfo()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "firmware",
			Timestamp: timestamp,
			Values:    &pb.QueryData_Firmware{Firmware: fw},
		}, nil

	case "os":
		osInfo, err := GetOSInfo()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "os",
			Timestamp: timestamp,
			Values:    &pb.QueryData_OsInfo{OsInfo: osInfo},
		}, nil

	case "swbom":
		swbom, err := GetSoftwareBOM()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "swbom",
			Timestamp: timestamp,
			Values:    &pb.QueryData_Swbom{Swbom: swbom},
		}, nil

	case "version":
		version, err := GetVersionInfo()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "version",
			Timestamp: timestamp,
			Values:    &pb.QueryData_Version{Version: version},
		}, nil

	case "all":
		all, err := GetAllInfo()
		if err != nil {
			return nil, err
		}
		return &pb.QueryData{
			Type:      "all",
			Timestamp: timestamp,
			Values:    &pb.QueryData_AllInfo{AllInfo: all},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported query option: %s", option)
	}
}
