// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ubuntu

import (
	"testing"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
)

func TestDownloader_Download(t *testing.T) {
	// Create a Downloader instance
	downloader := Downloader{
		Request: &pb.UpdateSystemSoftwareRequest{},
	}

	// Call the Download method
	err := downloader.Download()

	// Assertions
	assert.NoError(t, err, "Download should not return an error")
}
