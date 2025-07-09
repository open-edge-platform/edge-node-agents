/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater updates the firmware.
package fwupdater

import (
	"log"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

// FWUpdater is the main struct that contains the methods to update the firmware.
type FWUpdater struct {
	req *pb.UpdateFirmwareRequest
}

// NewFWUpdater creates a new FWUpdater instance.
func NewFWUpdater(req *pb.UpdateFirmwareRequest) *FWUpdater {
	return &FWUpdater{
		req: req,
	}
}

// UpdateFirmware updates the firmware based on the request.
func (u *FWUpdater) UpdateFirmware() (*pb.UpdateResponse, error) {
	// TODO: Check eligible for update

	downloader := NewDownloader(u.req)
	if err := downloader.download(); err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	// TODO: Get Configuration

	log.Println("Update completed successfully.")

	// TODO: Remove the artifacts after update success or failure.

	// TODO: Reboot the system if required.

	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}
