/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"log"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
)

// Rebooter is the concrete implementation of the Updater interface
// for the Ubuntu OS.
type Rebooter struct {
	CommandExecutor common.Executor
	Request         *pb.UpdateSystemSoftwareRequest
}

// Reboot method for Ubuntu
func (u *Rebooter) Reboot() error {
	if u.Request.DoNotReboot {
		log.Println("Reboot is disabled.  Skipping reboot.")
		return nil
	}

	return utils.RebootSystem(u.CommandExecutor)
}
