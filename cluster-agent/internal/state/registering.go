// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

type Registering struct {
	sm *StateMachine
	tF string
}

func (s *Registering) Register() error {
	s.sm.installCmd, s.sm.uninstallCmd = s.sm.client.RegisterToClusterOrch(utils.GetAuthContext(s.sm.ctx, s.tF), s.sm.guid)

	s.sm.set(s.sm.installInProgress)
	return s.sm.currentState.Register()
}

func (s *Registering) Deregister() error {
	return s.sm.incorrectActionRequest()
}

func (s *Registering) State() string {
	return "REGISTERING"
}
