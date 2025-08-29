/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	"fmt"
	"log"
	"os/exec"

	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// OSUpdater is the main struct that contains the methods to update the OS.
type OSUpdater struct {
	req                          *pb.UpdateSystemSoftwareRequest
	isProceedWithoutRollbackFunc func(*utils.Configurations) bool
	loadConfigFunc               func(afero.Fs, string) (*utils.Configurations, error)
}

// NewOSUpdater creates a new OSUpdater instance.
func NewOSUpdater(req *pb.UpdateSystemSoftwareRequest) *OSUpdater {
	return &OSUpdater{
		req:                          req,
		isProceedWithoutRollbackFunc: utils.IsProceedWithoutRollback,
		loadConfigFunc:               utils.LoadConfig,
	}
}

// UpdateOS updates the OS based on the request.
func (u *OSUpdater) UpdateOS(factory UpdaterFactory) (*pb.UpdateResponse, error) {
	log.Printf("Request Mode: %v\n", u.req.Mode)

	if u.req.Mode != pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
		// Download the update
		downloader := factory.CreateDownloader(u.req)
		err := downloader.Download()
		if err != nil {
			return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
		}
	}
	execCmd := common.NewExecutor(exec.Command, common.ExecuteAndReadOutput)
	cleaner := factory.CreateCleaner(execCmd, utils.SOTADownloadDir+"/")

	snapshot := factory.CreateSnapshotter(execCmd, u.req)
	// Create a snapshot of the current system
	err := snapshot.Snapshot()
	if err != nil {
		// Get the ProceedWithoutRollback flag from the config file to see if we should proceed with the update
		config, loadErr := u.loadConfigFunc(afero.NewOsFs(), utils.ConfigFilePath)
		if loadErr != nil {
			cleanFiles(cleaner)
			return &pb.UpdateResponse{StatusCode: 500, Error: loadErr.Error()}, nil
		}
		proceedWithoutRollback := u.isProceedWithoutRollbackFunc(config)
		if !proceedWithoutRollback {
			// If we are not proceeding with rollback, clean up the files
			cleanFiles(cleaner)
			return &pb.UpdateResponse{StatusCode: 500, Error: fmt.Sprintf("proceedWithoutRollback configuration flag is false; can not proceed as snapshot failed: %v", err.Error())}, nil
		}
	}

	// Update the OS
	updater := factory.CreateUpdater(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), u.req)
	proceedWithReboot, err := updater.Update()
	if err != nil {
		// Remove the artifacts if failure happens.
		cleanFiles(cleaner)
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
	}

	log.Println("Update completed successfully.")

	// Remove the artifacts after update success.
	cleanFiles(cleaner)

	if proceedWithReboot {
		if u.req.Mode != pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY {
			// Reboot the system
			rebooter := factory.CreateRebooter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), u.req)
			if err = rebooter.Reboot(); err != nil {
				return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil
			}
		}
	}

	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}

func cleanFiles(cleaner Cleaner) {
	if err := cleaner.Clean(); err != nil {
		log.Printf("[Warning] unable to cleanup files: %v", err.Error())
	}
}
