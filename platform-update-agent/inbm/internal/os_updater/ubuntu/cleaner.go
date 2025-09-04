/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"

// Cleaner is the concrete implementation of the Cleaner interface
// for the Ubuntu OS.
type Cleaner struct {
	CommandExecutor utils.Executor
	Path            string
}

// Clean method for Ubuntu
func (u *Cleaner) Clean() error {
	// No clean up needed for Ubuntu
	return nil
}
