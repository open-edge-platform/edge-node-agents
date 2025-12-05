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
	log.Printf("[REBOOT DEBUG] Ubuntu Rebooter.Reboot() called - DoNotReboot=%v", u.Request.DoNotReboot)
	if u.Request.DoNotReboot {
		log.Println("[REBOOT DEBUG] Reboot is disabled (DoNotReboot=true). Skipping reboot.")
		return nil
	}

	log.Println("[REBOOT DEBUG] Calling utils.RebootSystem() to execute /usr/sbin/reboot...")
	err := utils.RebootSystem(u.CommandExecutor)
	if err != nil {
		log.Printf("[REBOOT DEBUG] utils.RebootSystem() returned error: %v", err)
	} else {
		log.Println("[REBOOT DEBUG] utils.RebootSystem() returned successfully (system should be rebooting now)")
	}
	return err
}
