/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	"log"
	"os/exec"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"	
)

// UpdateOS updates the OS depending on the OS type.
func UpdateOS(req *pb.UpdateSystemSoftwareRequest, factory UpdaterFactory) (*pb.UpdateResponse, error) {
	log.Printf("Request Mode: %v\n", req.Mode)

	if req.Mode != pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
		// Download the update
		downloader := factory.CreateDownloader(req)
		err := downloader.Download()
		if err != nil {
			return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
		}
	}
	execCmd := utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)
	cleaner := factory.CreateCleaner(execCmd, utils.DownloadDir + "/")

	snapshot := factory.CreateSnapshotter(execCmd, req)
	// Create a snapshot of the current system
	err := snapshot.Snapshot()
	if err != nil {
		if errClean := cleaner.Clean(); errClean != nil {
			log.Printf("[Warning] unable to cleanup files: %v", errClean.Error())
		}
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	// Update the OS
	updater := factory.CreateUpdater(utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput), req)
	proceedWithReboot, err := updater.Update()
	if err != nil {
		// Remove the artifacts if failure happens.
		if errClean := cleaner.Clean(); errClean != nil {
			log.Printf("[Warning] %v", errClean.Error())
		}
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	log.Println("Update completed successfully.")
      
  	// Remove the artifacts after update success.
	if err = cleaner.Clean(); err != nil {
		log.Printf("[Warning] %v", err.Error())
	}
      
	if proceedWithReboot {
		if req.Mode != pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY {
			// Reboot the system
			rebooter := factory.CreateRebooter(utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput), req)
			if err =rebooter.Reboot(); err != nil {
				return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
			}
		}
	}

	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}
