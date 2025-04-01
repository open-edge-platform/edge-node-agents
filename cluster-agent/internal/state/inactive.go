// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

type Inactive struct {
	sm *StateMachine
}

func (s *Inactive) Register() error {
	s.sm.set(s.sm.registering)
	return s.sm.currentState.Register()
}

func (s *Inactive) Deregister() error {
	s.sm.set(s.sm.deregistering)
	return s.sm.currentState.Deregister()
}

func (s *Inactive) State() string {
	return "INACTIVE"
}
