/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package inbd provides the implementation of the InbServiceServer interface
// for the INBD service.
package inbd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	fwUpdater "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/fw_updater"
	telemetry "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	osUpdater "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater"
	appSource "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/ubuntu/app_source"
	osSource "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/ubuntu/os_source"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
)

// PowerManager interface for power management operations
type PowerManager interface {
	Reboot() error
	Shutdown() error
}

// DefaultPowerManager implements PowerManager using real system commands
type DefaultPowerManager struct{}

func (dpm *DefaultPowerManager) Reboot() error {
	return utils.RebootSystem(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput))
}

func (dpm *DefaultPowerManager) Shutdown() error {
	return utils.ShutdownSystem(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput))
}

// InbdServer implements the InbServiceServer interface
type InbdServer struct {
	pb.UnimplementedInbServiceServer
	powerManager PowerManager
}

// NewInbdServer creates a new InbdServer with default power manager
func NewInbdServer() *InbdServer {
	return &InbdServer{
		powerManager: &DefaultPowerManager{},
	}
}

// NewInbdServerWithPowerManager creates a new InbdServer with custom power manager (for testing)
func NewInbdServerWithPowerManager(pm PowerManager) *InbdServer {
	return &InbdServer{
		powerManager: pm,
	}
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

// SetPowerState sets the power state of the device
func (s *InbdServer) SetPowerState(ctx context.Context, req *pb.SetPowerStateRequest) (*pb.SetPowerStateResponse, error) {
	log.Printf("Received SetPowerState request")
	if req.Action == pb.SetPowerStateRequest_POWER_ACTION_UNSPECIFIED {
		return &pb.SetPowerStateResponse{StatusCode: 400, Error: "Power action is required"}, nil //nolint:nilerr // gRPC response pattern
	}

	switch req.Action {
	case pb.SetPowerStateRequest_POWER_ACTION_CYCLE:
		if err := s.powerManager.Reboot(); err != nil {
			return &pb.SetPowerStateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
		}
	case pb.SetPowerStateRequest_POWER_ACTION_OFF:
		if err := s.powerManager.Shutdown(); err != nil {
			return &pb.SetPowerStateResponse{StatusCode: 500, Error: fmt.Sprintf("shutdown failed: %s", err)}, nil //nolint:nilerr // gRPC response pattern
		}
	}

	return &pb.SetPowerStateResponse{StatusCode: 200, Error: "SUCCESS"}, nil //nolint:nilerr // gRPC response pattern
}

// UpdateFirmware updates the firmware
func (s *InbdServer) UpdateFirmware(ctx context.Context, req *pb.UpdateFirmwareRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateFirmware request")

	if req.Url == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "URL is required"}, nil //nolint:nilerr // gRPC response pattern
	}
	if err := validateURL(req.Url); err != nil {
		return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	// TODO: Validate signature against expected format
	// TODO: Add unittest test case for invalid signature format

	// Validate hash algorithm, default to sha384 if not provided
	finalHashAlgorithm := "sha384"
	if req.HashAlgorithm != "" {
		switch strings.ToLower(req.HashAlgorithm) {
		case "sha256", "sha384", "sha512":
			finalHashAlgorithm = strings.ToLower(req.HashAlgorithm)
		default:
			return &pb.UpdateResponse{
				StatusCode: 400,
				Error:      "invalid hash algorithm: must be 'sha256', 'sha384', or 'sha512'",
			}, nil //nolint:nilerr // gRPC response pattern
		}
	}
	req.HashAlgorithm = finalHashAlgorithm

	resp, err := fwUpdater.NewFWUpdater(req).UpdateFirmware()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	return &pb.UpdateResponse{StatusCode: resp.StatusCode, Error: resp.Error}, nil //nolint:nilerr // gRPC response pattern
}

// UpdateSystemSoftware updates the system software
func (s *InbdServer) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest) (*pb.UpdateResponse, error) {

	log.Printf("Received UpdateSystemSoftware request")
	os, err := common.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	if req.Url != "" {
		if err := validateURL(req.Url); err != nil {
			return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
		}
	}

	sotaFactory, err := osUpdater.GetOSUpdaterFactory(os)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	resp, err := osUpdater.NewOSUpdater(req).UpdateOS(sotaFactory)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	return &pb.UpdateResponse{StatusCode: resp.StatusCode, Error: resp.Error}, nil //nolint:nilerr // gRPC response pattern
}

// UpdateOSSource creates a new /etc/apt/sources.list file with only the sources provided
func (s *InbdServer) UpdateOSSource(ctx context.Context, req *pb.UpdateOSSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateOSSource request")

	os, err := common.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Update OS Source is only for Ubuntu."}, nil //nolint:nilerr // gRPC response pattern
	}
	if len(req.SourceList) == 0 {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Source list is empty"}, nil //nolint:nilerr // gRPC response pattern
	}
	err = osSource.NewUpdater().Update(req.SourceList, osSource.UbuntuAptSourcesList)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil //nolint:nilerr // gRPC response pattern
}

// AddApplicationSource adds the source file under /etc/apt/sources.list.d/.
// It optionally adds the GPG key under /usr/share/keyrings/ if the GPG key name is provided.
func (s *InbdServer) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest) (*pb.UpdateResponse, error) {
	os, err := common.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Add Application Source is only for Ubuntu."}, nil //nolint:nilerr // gRPC response pattern
	}
	if req.GpgKeyUri != "" {
		if err := validateURL(req.GpgKeyUri); err != nil {
			return &pb.UpdateResponse{StatusCode: 400, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
		}
	}
	if req.Filename == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Filename is empty"}, nil //nolint:nilerr // gRPC response pattern
	}
	if len(req.Source) == 0 {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Source list is empty"}, nil //nolint:nilerr // gRPC response pattern
	}
	err = appSource.NewAdder().Add(req)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil //nolint:nilerr // gRPC response pattern
}

// RemoveApplicationSource removes the source file from under /etc/apt/sources.list.d/.
// It optionally removes the GPG key under /usr/share/keyrings/ if the GPG key name is provided.
func (s *InbdServer) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest) (*pb.UpdateResponse, error) {
	os, err := common.DetectOS()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 415, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	if os != "Ubuntu" {
		return &pb.UpdateResponse{StatusCode: 415, Error: "Unsupported OS.  Remove Application Source is only for Ubuntu."}, nil //nolint:nilerr // gRPC response pattern
	}
	if req.Filename == "" {
		return &pb.UpdateResponse{StatusCode: 400, Error: "Filename is empty"}, nil //nolint:nilerr // gRPC response pattern
	}
	err = appSource.NewRemover().Remove(req)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil //nolint:nilerr // gRPC response pattern

}

// LoadConfig loads the configuration from the specified URI.
func (s *InbdServer) LoadConfig(ctx context.Context, req *pb.LoadConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("Received LoadConfig request")
	if req.Uri == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "uri is required", Success: false}, nil //nolint:nilerr // gRPC response pattern
	}
	op := &utils.ConfigOperation{}

	// TODO: Validate signature against expected format
	// TODO: Add unittest test case for invalid signature format

	// Validate hash algorithm, default to sha384 if not provided
	finalHashAlgorithm := "sha384"
	if req.HashAlgorithm != "" {
		switch strings.ToLower(req.HashAlgorithm) {
		case "sha256", "sha384", "sha512":
			finalHashAlgorithm = strings.ToLower(req.HashAlgorithm)
		default:
			return &pb.ConfigResponse{
				StatusCode: 400,
				Error:      "invalid hash algorithm: must be 'sha256', 'sha384', or 'sha512'",
				Success:    false,
			}, nil //nolint:nilerr // gRPC response pattern
		}
	}

	err := op.LoadConfigCommand(req.Uri, req.Signature, finalHashAlgorithm)
	if err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil //nolint:nilerr // gRPC response pattern
}

// GetConfig retrieves configuration values from the configuration file
func (s *InbdServer) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	log.Printf("Received GetConfig request")
	if strings.TrimSpace(req.Path) == "" {
		return &pb.GetConfigResponse{StatusCode: 400, Error: "path is required", Success: false, Value: ""}, nil //nolint:nilerr // gRPC response pattern
	}
	op := &utils.ConfigOperation{}
	val, errStr, err := op.GetConfigCommand(req.Path)
	if err != nil && val == "" {
		return &pb.GetConfigResponse{
			StatusCode: 500,
			Error:      errStr,
			Success:    false,
			Value:      "",
		}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.GetConfigResponse{
		StatusCode: 200,
		Error:      errStr,
		Success:    errStr == "",
		Value:      val,
	}, nil //nolint:nilerr // gRPC response pattern
}

// handleConfigOperation is a generic helper for config operations
func (s *InbdServer) handleConfigOperation(operationName, path string, operation func(string) error) (*pb.ConfigResponse, error) {
	log.Printf("Received %s request", operationName)
	if strings.TrimSpace(path) == "" {
		return &pb.ConfigResponse{StatusCode: 400, Error: "path is required", Success: false}, nil //nolint:nilerr // gRPC response pattern
	}
	if err := operation(path); err != nil {
		return &pb.ConfigResponse{StatusCode: 500, Error: err.Error(), Success: false}, nil //nolint:nilerr // gRPC response pattern
	}
	return &pb.ConfigResponse{StatusCode: 200, Error: "", Success: true}, nil //nolint:nilerr // gRPC response pattern
}

func (s *InbdServer) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.ConfigResponse, error) {
	return s.handleConfigOperation("SetConfig", req.Path, func(path string) error {
		op := &utils.ConfigOperation{}
		return op.SetConfigCommand(path)
	})
}

func (s *InbdServer) AppendConfig(ctx context.Context, req *pb.AppendConfigRequest) (*pb.ConfigResponse, error) {
	return s.handleConfigOperation("AppendConfig", req.Path, func(path string) error {
		op := &utils.ConfigOperation{}
		return op.AppendConfigCommand(path)
	})
}

func (s *InbdServer) RemoveConfig(ctx context.Context, req *pb.RemoveConfigRequest) (*pb.ConfigResponse, error) {
	return s.handleConfigOperation("RemoveConfig", req.Path, func(path string) error {
		op := &utils.ConfigOperation{}
		return op.RemoveConfigCommand(path)
	})
}

// Query returns system information based on the query option
func (s *InbdServer) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {

	if req == nil {
		return &pb.QueryResponse{
			StatusCode: 400,
			Error:      "request is required",
			Success:    false,
			Data:       nil,
		}, nil //nolint:nilerr // gRPC response pattern
	}

	log.Printf("Received Query request for option: %s", req.Option)

	if req.Option == pb.QueryOption_QUERY_OPTION_UNSPECIFIED {
		return &pb.QueryResponse{
			StatusCode: 400,
			Error:      "invalid query option",
			Success:    false,
			Data:       nil,
		}, nil //nolint:nilerr // gRPC response pattern
	}

	// Convert enum to string
	optionStr := convertQueryOptionToString(req.Option)

	queryHandler := telemetry.NewQueryHandler()
	data, err := queryHandler.HandleQuery(optionStr)
	if err != nil {
		return &pb.QueryResponse{ //nolint:nilerr // gRPC response pattern
			StatusCode: 500,
			Error:      err.Error(),
			Success:    false,
			Data:       nil,
		}, nil
	}

	return &pb.QueryResponse{
		StatusCode: 200,
		Error:      "",
		Success:    true,
		Data:       data,
	}, nil //nolint:nilerr // gRPC response pattern
}

// convertQueryOptionToString converts QueryOption enum to string
func convertQueryOptionToString(option pb.QueryOption) string {
	switch option {
	case pb.QueryOption_QUERY_OPTION_HARDWARE:
		return "hw"
	case pb.QueryOption_QUERY_OPTION_FIRMWARE:
		return "fw"
	case pb.QueryOption_QUERY_OPTION_OS:
		return "os"
	case pb.QueryOption_QUERY_OPTION_SWBOM:
		return "swbom"
	case pb.QueryOption_QUERY_OPTION_VERSION:
		return "version"
	case pb.QueryOption_QUERY_OPTION_ALL:
		return "all"
	default:
		return "all" // Default to "all" for unknown options
	}
}
