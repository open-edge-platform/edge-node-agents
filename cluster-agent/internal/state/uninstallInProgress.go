// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"context"
	"os"
	"strings"
)

type UninstallInProgress struct {
	sm      *StateMachine
	execute func(ctx context.Context, command string) error
}

func (s *UninstallInProgress) Register() error {
	return s.sm.incorrectActionRequest()
}

func (s *UninstallInProgress) Deregister() error {
	log.Info("Start kubernetes engine uninstallation script")

	lsbRelease, err := os.ReadFile("/etc/lsb-release")
	if err != nil {
		log.Error("failed to read /etc/lsb-release, err:", err)
		return err
	}

	isEmt := strings.Contains(string(lsbRelease), `DISTRIB_ID="Edge Microvisor Toolkit"`)
	if isEmt {
		err = s.execute(s.sm.ctx, `sudo sed -i 's|\/etc\/cni$|\/etc\/cni\/*|g' /opt/rke2/bin/rke2-uninstall.sh`)
		if err != nil {
			log.Info("Failed to patch uninstall script, err: ", err)
		}
	}

	err = s.execute(s.sm.ctx, s.sm.uninstallCmd)
	if err != nil {

		s.sm.uninstallCmd = "" // trigger fetching uninstallCmd from cluster orchestrator
		s.sm.set(s.sm.inactive)
		return err
	}

	log.Info("kubernetes engine uninstallation script executed successfully")

	err = s.execute(s.sm.ctx, s.sm.cleanupCmd)
	if err != nil {
		log.Info("Failed to clean up cluster volume mounts")
		s.sm.set(s.sm.inactive)
		return err
	}
	s.sm.set(s.sm.inactive)
	return nil
}

func (s *UninstallInProgress) State() string {
	return "UNINSTALL_IN_PROGRESS"
}
