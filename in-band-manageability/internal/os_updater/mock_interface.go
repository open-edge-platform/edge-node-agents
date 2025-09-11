/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

import (
	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

// MockDownloader is a mock implementation of the Downloader interface.
type MockDownloader struct {
	DownloadFunc func() error
}

// Download calls the DownloadFunc.
func (m *MockDownloader) Download() error {
	return m.DownloadFunc()
}

// MockUpdater is a mock implementation of the Updater interface.
type MockUpdater struct {
	UpdateFunc func() (bool, error)
}

// Update calls the UpdateFunc.
func (m *MockUpdater) Update() (bool, error) {
	proceedWithReboot, err := m.UpdateFunc()
	return proceedWithReboot, err
}

// MockSnapshotter is a mock implementation of the Snapshotter interface.
type MockSnapshotter struct {
	SnapshotFunc func() error
}

// Snapshot calls the SnapshotFunc.
func (m *MockSnapshotter) Snapshot() error {
	return m.SnapshotFunc()
}

// MockCleaner is a mock implementation of the Cleaner interface.
type MockCleaner struct {
	CleanFunc func() error
}

// Clean calls the CleanFunc.
func (m *MockCleaner) Clean() error {
	return m.CleanFunc()
}

// MockRebooter is a mock implementation of the Rebooter interface.
type MockRebooter struct {
	RebootFunc func() error
}

// Reboot calls the RebootFunc.
func (m *MockRebooter) Reboot() error {
	return m.RebootFunc()
}

// MockUpdaterFactory is a mock implementation of the UpdaterFactory interface.
type MockUpdaterFactory struct {
	CreateDownloaderFunc  func(*pb.UpdateSystemSoftwareRequest) Downloader
	CreateUpdaterFunc     func(common.Executor, *pb.UpdateSystemSoftwareRequest) Updater
	CreateCleanerFunc     func(common.Executor, string) Cleaner
	CreateRebooterFunc    func(common.Executor, *pb.UpdateSystemSoftwareRequest) Rebooter
	CreateSnapshotterFunc func(common.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter
}

// CreateDownloader calls the CreateDownloaderFunc.
func (m *MockUpdaterFactory) CreateDownloader(req *pb.UpdateSystemSoftwareRequest) Downloader {
	return m.CreateDownloaderFunc(req)
}

// CreateUpdater calls the CreateUpdaterFunc.
func (m *MockUpdaterFactory) CreateUpdater(cmdExec common.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
	return m.CreateUpdaterFunc(cmdExec, req)
}

// CreateCleaner calls the CreateCleanerFunc.
func (m *MockUpdaterFactory) CreateCleaner(cmdExec common.Executor, path string) Cleaner {
	return m.CreateCleanerFunc(cmdExec, path)
}

// CreateRebooter calls the CreateRebooterFunc.
func (m *MockUpdaterFactory) CreateRebooter(cmdExec common.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
	return m.CreateRebooterFunc(cmdExec, req)
}

// CreateSnapshotter calls the CreateSnapshotterFunc.
func (m *MockUpdaterFactory) CreateSnapshotter(cmdExec common.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
	return m.CreateSnapshotterFunc(cmdExec, req)
}
