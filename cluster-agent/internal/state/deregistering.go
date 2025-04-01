// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

type Deregistering struct {
	sm *StateMachine
	tF string
}

func (s *Deregistering) Register() error {
	return s.sm.incorrectActionRequest()
}

func (s *Deregistering) Deregister() error {
	if s.sm.uninstallCmd == "" {
		s.sm.installCmd, s.sm.uninstallCmd = s.sm.client.RegisterToClusterOrch(utils.GetAuthContext(s.sm.ctx, s.tF), s.sm.guid)
	}

	s.sm.set(s.sm.uninstallInProgress)
	return s.sm.currentState.Deregister()
}

func (s *Deregistering) State() string {
	return "DEREGISTERING"
}
