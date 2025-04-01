// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

type Active struct {
	sm *StateMachine
}

func (s *Active) Register() error {
	return s.sm.incorrectActionRequest()
}

func (s *Active) Deregister() error {
	s.sm.set(s.sm.deregistering)
	return s.sm.currentState.Deregister()
}

func (s *Active) State() string {
	return "ACTIVE"
}
