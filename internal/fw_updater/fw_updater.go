/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater updates the firmware.
package fwupdater

import (
	"log"

	telemetry "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/telemetry"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// FWUpdater is the main struct that contains the methods to update the firmware.
type FWUpdater struct {
	req *pb.UpdateFirmwareRequest
	fs  afero.Fs
}

// NewFWUpdater creates a new FWUpdater instance.
func NewFWUpdater(req *pb.UpdateFirmwareRequest) *FWUpdater {
	return &FWUpdater{
		req: req,
		fs:  afero.NewOsFs(), // Use real filesystem by default
	}
}

// NewFWUpdaterWithFS creates a new FWUpdater instance with a custom filesystem.
// This is primarily used for testing with mocked filesystems.
func NewFWUpdaterWithFS(req *pb.UpdateFirmwareRequest, fs afero.Fs) *FWUpdater {
	return &FWUpdater{
		req: req,
		fs:  fs,
	}
}

// UpdateFirmware updates the firmware based on the request.
func (u *FWUpdater) UpdateFirmware() (*pb.UpdateResponse, error) {
	log.Println("Starting firmware update process.")

	hwInfo, err := telemetry.GetHardwareInfo()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	// Get the firmware update tool info.
	firmwareToolInfo, err := GetFirmwareUpdateToolInfo(u.fs, hwInfo.GetSystemProductName())
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}
	log.Printf("Firmware update tool info: %+v", firmwareToolInfo)

	// Get the firmware information for the release date check.
	fwInfo, err := telemetry.GetFirmwareInfo()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	// Check if the firmware update is required.
	if fwInfo.GetBiosReleaseDate().String() > u.req.ReleaseDate.String() {
		return &pb.UpdateResponse{
			StatusCode: 400,
			Error:      "Firmware update is not required. Current firmware is up to date.",
		}, nil
	}

	// Download the firmware update file.
	// TODO: Download needs to support signature checking
	// and username and password for private repositories.
	log.Printf("Downloading firmware update from URL: %s", u.req.Url)
	downloader := NewDownloader(u.req)
	if err := downloader.download(); err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	// TODO: Perform the firmware update using the downloaded file and
	// the firmware update tool info.

	log.Println("Update completed successfully.")

	// TODO: Remove the artifacts after update success or failure.

	// TODO: Check the incoming request to see if a reboot is desired.
	// Reboot the system if so.  Call the utils.RebootSystem function.

	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}
