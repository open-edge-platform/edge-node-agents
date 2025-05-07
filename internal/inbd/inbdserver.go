/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package inbd

import (
	"context"
	"log"

	osUpdater "github.com/intel/intel-inb-manageability/internal/os_updater"
	appSource "github.com/intel/intel-inb-manageability/internal/os_updater/ubuntu/app_source"
	osSource "github.com/intel/intel-inb-manageability/internal/os_updater/ubuntu/os_source"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
)

type InbdServer struct {
	pb.UnimplementedInbServiceServer
}

// UpdateSystemSoftware updates the system software
func (s *InbdServer) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateSystemSoftware request")
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}

	sotaFactory, err := osUpdater.GetOSUpdaterFactory(os)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}

	resp, err := osUpdater.UpdateOS(req, sotaFactory)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	
	return &pb.UpdateResponse{StatusCode: resp.StatusCode, Error: resp.Error}, nil
}

// UpdateOSSource creates a new /etc/apt/sources.list file with only the sources provided
func (s *InbdServer) UpdateOSSource(ctx context.Context, req *pb.UpdateOSSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateOSSource request")
	
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Update OS Source is only for Ubuntu."}, nil
	}
	if len(req.SourceList) == 0 {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Source list is empty"}, nil
	}
	err = osSource.NewUpdater().Update(req.SourceList, osSource.UbuntuAptSourcesList)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}

func (s *InbdServer) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest) (*pb.UpdateResponse, error) {
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Add Application Source is only for Ubuntu."}, nil
	}
	err = appSource.NewAdder().Add(req)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}

func (s *InbdServer) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received RemoveApplicationSource request")
	return &pb.UpdateResponse{StatusCode: 501, Error: "Not implemented"}, nil
}
