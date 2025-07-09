/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package inbd

import (
	"context"
	"log"
	"net/url"
	"errors"
  "strings"

	fwUpdater "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/fw_updater"
	utils "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	osUpdater "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/os_updater"
	appSource "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/os_updater/ubuntu/app_source"
	osSource "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/os_updater/ubuntu/os_source"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

// InbdServer implements the InbServiceServer interface
type InbdServer struct {
	pb.UnimplementedInbServiceServer
}

// validateURL checks if the given URL is non-empty, well-formed, and uses http or https scheme.
func validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL is empty")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return errors.New("URL is not valid: " + err.Error())
	}
	if parsed.Scheme != "https" {
		return errors.New("URL must use https scheme")
	}
	if parsed.Host == "" {
		return errors.New("URL must have a host")
	}
	return nil
}
// UpdateFirmware updates the firmware
func (s *InbdServer) UpdateFirmware(ctx context.Context, req *pb.UpdateFirmwareRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateFirmware request")

	if req.Url == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "URL is required"}, nil
	}
	if err := validateURL(req.Url); err != nil {
		return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil
	}

	resp, err := fwUpdater.NewFWUpdater(req).UpdateFirmware()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}	

	return &pb.UpdateResponse{StatusCode: resp.StatusCode, Error: resp.Error}, nil
}

// UpdateSystemSoftware updates the system software
func (s *InbdServer) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest) (*pb.UpdateResponse, error) {

	log.Printf("Received UpdateSystemSoftware request")
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}
	if req.Url != "" {
		if err := validateURL(req.Url); err != nil {
			return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil
		}
	}

	sotaFactory, err := osUpdater.GetOSUpdaterFactory(os)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}

	resp, err := osUpdater.NewOSUpdater(req).UpdateOS(sotaFactory)
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

// AddApplicationSource adds the source file under /etc/apt/sources.list.d/.
// It optionally adds the GPG key under /usr/share/keyrings/ if the GPG key name is provided.
func (s *InbdServer) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest) (*pb.UpdateResponse, error) {
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Add Application Source is only for Ubuntu."}, nil
	}
	if req.GpgKeyUri != "" {
		if err := validateURL(req.GpgKeyUri); err != nil {
			return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil
		}
	}
	if req.Filename == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Filename is empty"}, nil
	}
	if len(req.Source) == 0 {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Source list is empty"}, nil
	}
	err = appSource.NewAdder().Add(req)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}

// RemoveApplicationSource removes the source file from under /etc/apt/sources.list.d/.
// It optionally removes the GPG key under /usr/share/keyrings/ if the GPG key name is provided.
func (s *InbdServer) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest) (*pb.UpdateResponse, error) {
	os, err := osUpdater.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Remove Application Source is only for Ubuntu."}, nil
	}
	if req.Filename == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Filename is empty"}, nil
	}
	err = appSource.NewRemover().Remove(req)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil

}

func (s *InbdServer) LoadConfig(ctx context.Context, req *pb.LoadConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("Received LoadConfig request")
	if req.Uri == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "uri is required", Success: false}, nil
	}
	op := &utils.ConfigOperation{}
	err := op.LoadConfigCommand(req.Uri, req.Signature)
	if err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil
}

func (s *InbdServer) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	log.Printf("Received GetConfig request")
	if strings.TrimSpace(req.Path) == "" {
		return &pb.GetConfigResponse{StatusCode: 400, Error: "path is required", Success: false, Value: ""}, nil
	}
	op := &utils.ConfigOperation{}
	val, errStr, err := op.GetConfigCommand(req.Path)
	if err != nil && val == "" {
		return &pb.GetConfigResponse{
			StatusCode: 500,
			Error:      errStr,
			Success:    false,
			Value:      "",
		}, nil
	}
	return &pb.GetConfigResponse{
		StatusCode: 200,
		Error:      errStr,
		Success:    errStr == "",
		Value:      val,
	}, nil
}

func (s *InbdServer) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("Received SetConfig request")
	if strings.TrimSpace(req.Path) == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "path is required", Success: false}, nil
	}
	op := &utils.ConfigOperation{}
	if err := op.SetConfigCommand(req.Path); err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil
}

func (s *InbdServer) AppendConfig(ctx context.Context, req *pb.AppendConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("Received AppendConfig request")
	if strings.TrimSpace(req.Path) == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "path is required", Success: false}, nil
	}
	op := &utils.ConfigOperation{}
	if err := op.AppendConfigCommand(req.Path); err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil
}

func (s *InbdServer) RemoveConfig(ctx context.Context, req *pb.RemoveConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("Received RemoveConfig request")
	if strings.TrimSpace(req.Path) == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "path is required", Success: false}, nil
	}
	op := &utils.ConfigOperation{}
	if err := op.RemoveConfigCommand(req.Path); err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil
}

// Query returns system information based on the query option
func (s *InbdServer) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	log.Printf("Received Query request for option: %s", req.Option)

	return &pb.QueryResponse{
		StatusCode: 501,
		Error:      "Not Implemented",
		Success:    false,
		Data:       nil,
	}, nil
}
