/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	"fmt"

	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	emt "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/os_updater/emt"
	ubuntu "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/os_updater/ubuntu"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// UpdaterFactory is an interface that contains the methods to create the concrete classes for the OS updater.
type UpdaterFactory interface {
	CreateDownloader(*pb.UpdateSystemSoftwareRequest) Downloader
	CreateUpdater(common.Executor, *pb.UpdateSystemSoftwareRequest) Updater
	CreateSnapshotter(common.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter
	CreateCleaner(common.Executor, string) Cleaner
	CreateRebooter(common.Executor, *pb.UpdateSystemSoftwareRequest) Rebooter
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
func (f *EMTFactory) CreateUpdater(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
	return emt.NewUpdater(commandExecutor, req)
}

// CreateSnapshotter creates a snapshotter concrete class for EMT OS.
func (f *EMTFactory) CreateSnapshotter(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
	return emt.NewSnapshotter(commandExecutor, req)
}

// CreateCleaner creates a cleaner concrete class for EMT OS.
func (f *EMTFactory) CreateCleaner(commandExecutor common.Executor, path string) Cleaner {
	return emt.NewCleaner(commandExecutor, path)
}

// CreateRebooter creates a rebooter concrete class for EMT OS.
func (f *EMTFactory) CreateRebooter(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
	return emt.NewRebooter(commandExecutor, req)
}

// UbuntuFactory represents an EMT factory.
type UbuntuFactory struct{}

// CreateDownloader creates a downloader concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateDownloader(req *pb.UpdateSystemSoftwareRequest) Downloader {
	return &ubuntu.Downloader{Request: req}
}

// CreateUpdater creates an OS updater concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateUpdater(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
	return &ubuntu.Updater{
		CommandExecutor:         commandExecutor,
		Request:                 req,
		GetEstimatedSize:        ubuntu.GetEstimatedSize,
		GetFreeDiskSpaceInBytes: utils.GetFreeDiskSpaceInBytes,
	}
}

// CreateSnapshotter creates a snapshotter concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateSnapshotter(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
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

// CreateCleaner creates a cleaner concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateCleaner(commandExecutor common.Executor, path string) Cleaner {
	return &ubuntu.Cleaner{
		CommandExecutor: commandExecutor,
		Path:            path,
	}
}

// CreateRebooter creates a rebooter concrete class for Ubuntu OS.
func (f *UbuntuFactory) CreateRebooter(commandExecutor common.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
	return &ubuntu.Rebooter{
		CommandExecutor: commandExecutor,
		Request:         req,
	}
}
