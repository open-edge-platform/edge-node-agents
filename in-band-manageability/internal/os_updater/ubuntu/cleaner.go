/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	common "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/common"
)

// Cleaner is the concrete implementation of the Cleaner interface
// for the Ubuntu OS.
type Cleaner struct {
	CommandExecutor common.Executor
	Path            string
}

// Clean method for Ubuntu
func (u *Cleaner) Clean() error {
	// No clean up needed for Ubuntu
	return nil
}
