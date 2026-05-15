// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import "context"

type InstallInProgress struct {
	sm      *StateMachine
	execute func(ctx context.Context, command string) error
}

func (s *InstallInProgress) Register() error {
	log.Info("Start kubernetes engine installation script")

	resolvedCmd, err := ResolveReservedCPUPolicy(s.sm.installCmd)
	if err != nil {
		log.Warnf("Failed to resolve reserved CPU policy, proceeding with original command: %v", err)
		resolvedCmd = s.sm.installCmd
	}

	err = s.execute(s.sm.ctx, resolvedCmd)
	if err != nil {
		s.sm.set(s.sm.inactive)
		return err
	}

	log.Info("kubernetes engine installation script executed successfully")

	s.sm.set(s.sm.active)
	return nil
}

func (s *InstallInProgress) Deregister() error {
	return s.sm.incorrectActionRequest()
}

func (s *InstallInProgress) State() string {
	return "INSTALL_IN_PROGRESS"
}
