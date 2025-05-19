/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	"fmt"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	emt "github.com/intel/intel-inb-manageability/internal/os_updater/emt"
	ubuntu "github.com/intel/intel-inb-manageability/internal/os_updater/ubuntu"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// UpdaterFactory is an interface that contains the methods to create the concrete classes for the OS updater.
type UpdaterFactory interface {
	CreateDownloader(req *pb.UpdateSystemSoftwareRequest) Downloader
	CreateUpdater(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Updater
	CreateSnapshotter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter
	CreateRebooter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter
}

// GetOSUpdaterFactory returns the correct concrete classes for the OS updater based on the OS type.
func GetOSUpdaterFactory(os string) (UpdaterFactory, error) {
	if os == "EMT" {
		return &EMTFactory{}, nil
	}

	if os == "Ubuntu" {
		return &UbuntuFactory{}, nil
	}
	return nil, fmt.Errorf("Unsupported OS")
}

// EMTFactory represents an EMT factory.
type EMTFactory struct{}

// CreateDownloader creates a downloader concrete class for EMT OS.
func (f *EMTFactory) CreateDownloader(req *pb.UpdateSystemSoftwareRequest) Downloader {
	return emt.NewDownloader(req)
}

// CreateUpdater creates an OS updater concrete class for EMT OS.
func (f *EMTFactory) CreateUpdater(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
	return emt.NewUpdater(commandExecutor, req)
}

// CreateSnapshotter creates a snapshotter concrete class for EMT OS.
func (f *EMTFactory) CreateSnapshotter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
	return emt.NewSnapshotter(commandExecutor, req)
}

// CreateRebooter creates a rebooter concrete class for EMT OS.
func (f *EMTFactory) CreateRebooter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
	return emt.NewRebooter(commandExecutor, req)
}

// UbuntuFactory represents an EMT factory.
type UbuntuFactory struct{}

// CreateDownloader creates a downloader concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateDownloader(req *pb.UpdateSystemSoftwareRequest) Downloader {
	return &ubuntu.Downloader{Request: req}
}

// CreateUpdater creates an OS updater concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateUpdater(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
	return &ubuntu.Updater{
		CommandExecutor:         commandExecutor,
		Request:                 req,
		GetEstimatedSize:        ubuntu.GetEstimatedSize,
		GetFreeDiskSpaceInBytes: utils.GetFreeDiskSpaceInBytes,
	}
}

// CreateSnapshotter creates a snapshotter concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateSnapshotter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
	return &ubuntu.Snapshotter{
		CommandExecutor:         commandExecutor,
		IsBTRFSFileSystemFunc:   utils.IsBTRFSFileSystem,
		IsSnapperInstalledFunc:  ubuntu.IsSnapperInstalled,
		EnsureSnapperConfigFunc: ubuntu.EnsureSnapperConfig,
		ClearStateFileFunc:      utils.ClearStateFile,
		WriteToStateFileFunc:    utils.WriteToStateFile,
		Fs:                      afero.NewOsFs(),
	}
}

// CreateRebooter creates a rebooter concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateRebooter(commandExecutor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
	return &ubuntu.Rebooter{
		CommandExecutor: commandExecutor,
		Request:         req,
	}
}
