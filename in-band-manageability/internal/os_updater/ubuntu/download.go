/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"fmt"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
)

// Downloader is the concrete implementation of the IDownloader interface
// for the Ubuntu OS.
type Downloader struct {
	Request *pb.UpdateSystemSoftwareRequest
}

// Download method for Ubuntu
func (u *Downloader) Download() error {
	fmt.Println("Debian-based OS does not require a file download to perform a software update")
	return nil
}
