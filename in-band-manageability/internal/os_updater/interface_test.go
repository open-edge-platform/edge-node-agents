/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
package osupdater

import (
	"os/exec"
	"testing"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	emt "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/emt"
	ubuntu "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/ubuntu"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetOSUpdaterFactory(t *testing.T) {
	t.Run("returns EMTUpdater for EMT OS", func(t *testing.T) {
		factory, err := GetOSUpdaterFactory("EMT")
		assert.NoError(t, err)
		assert.IsType(t, &EMTFactory{}, factory)
	})

	t.Run("returns UbuntuUpdater for Ubuntu OS", func(t *testing.T) {
		factory, err := GetOSUpdaterFactory("Ubuntu")
		assert.NoError(t, err)
		assert.IsType(t, &UbuntuFactory{}, factory)
	})

	t.Run("returns error for unsupported OS", func(t *testing.T) {
		factory, err := GetOSUpdaterFactory("UnsupportedOS")
		assert.Error(t, err)
		assert.Nil(t, factory)
	})
}

func TestEMTUpdater(t *testing.T) {
	emtUpdater := &EMTFactory{}
	req := &pb.UpdateSystemSoftwareRequest{
		Mode:        pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY,
		DoNotReboot: true,
		Signature:   "signature",
	}

	t.Run("createDownloader returns Downloader", func(t *testing.T) {
		downloader := emtUpdater.CreateDownloader(req)
		assert.IsType(t, &emt.Downloader{}, downloader)
	})

	t.Run("createUpdater returns EMTUpdater", func(t *testing.T) {
		updater := emtUpdater.CreateUpdater(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), req)
		assert.IsType(t, &emt.Updater{}, updater)
	})

	t.Run("createSnapshotter returns EMTUpdater", func(t *testing.T) {
		snapshotter := emtUpdater.CreateSnapshotter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), req)
		assert.IsType(t, &emt.Snapshotter{}, snapshotter)
	})

	t.Run("createRebooter returns EMTRebooter", func(t *testing.T) {
		rebooter := emtUpdater.CreateRebooter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), req)
		assert.IsType(t, &emt.Rebooter{}, rebooter)
	})
}

func TestUbuntuUpdater(t *testing.T) {
	ubuntuUpdater := &UbuntuFactory{}
	req := pb.UpdateSystemSoftwareRequest{
		Mode:        pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY,
		DoNotReboot: true,
		Signature:   "signature",
	}

	t.Run("createDownloader returns UbuntuDownloader", func(t *testing.T) {
		downloader := ubuntuUpdater.CreateDownloader(&req)
		assert.IsType(t, &ubuntu.Downloader{}, downloader)
	})

	t.Run("createUpdater returns UbuntuUpdater", func(t *testing.T) {
		updater := ubuntuUpdater.CreateUpdater(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), &req)
		assert.IsType(t, &ubuntu.Updater{}, updater)
	})

	t.Run("createSnapshotter returns UbuntuSnapshotter", func(t *testing.T) {
		snapshotter := ubuntuUpdater.CreateSnapshotter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), &req)
		assert.IsType(t, &ubuntu.Snapshotter{}, snapshotter)
	})

	t.Run("createRebooter returns UbuntuRebooter", func(t *testing.T) {
		rebooter := ubuntuUpdater.CreateRebooter(common.NewExecutor(exec.Command, common.ExecuteAndReadOutput), &req)
		assert.IsType(t, &ubuntu.Rebooter{}, rebooter)
	})
}
