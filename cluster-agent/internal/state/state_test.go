// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"context"
	"fmt"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/comms"
	"github.com/stretchr/testify/assert"
)

func TestDefaultStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	assert.Equal(t, "INACTIVE", s.State())
}

func TestRegisteringStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.registering)
	assert.Equal(t, "REGISTERING", s.State())
}

func TestInstallInProgressStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.installInProgress)
	assert.Equal(t, "INSTALL_IN_PROGRESS", s.State())
}

func TestActiveStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.active)
	assert.Equal(t, "ACTIVE", s.State())
}

func TestDeregisteringStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.deregistering)
	assert.Equal(t, "DEREGISTERING", s.State())
}

func TestUninstallInProgressStatus(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.uninstallInProgress)
	assert.Equal(t, "UNINSTALL_IN_PROGRESS", s.State())
}

func TestRegisteringIncorrectTransition(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.registering)
	assert.Equal(t, "REGISTERING", s.State())
	assert.Error(t, s.Deregister())
}

func TestInstallInProgressIncorrectTransition(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.installInProgress)
	assert.Equal(t, "INSTALL_IN_PROGRESS", s.State())
	assert.Error(t, s.Deregister())
}

func TestActiveIncorrectTransition(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.active)
	assert.Equal(t, "ACTIVE", s.State())
	assert.Error(t, s.Register())
}

func TestDeregisteringIncorrectTransition(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.deregistering)
	assert.Equal(t, "DEREGISTERING", s.State())
	assert.Error(t, s.Register())
}

func TestUninstallInProgressIncorrectTransition(t *testing.T) {
	s := New(context.TODO(), nil, "", "", nil)
	s.set(s.uninstallInProgress)
	assert.Equal(t, "UNINSTALL_IN_PROGRESS", s.State())
	assert.Error(t, s.Register())
}

//nolint:dupl
func TestInstallError(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		return fmt.Errorf("Installation failed")
	}
	s := New(context.TODO(), orchestratorClient, "", "", execute)

	s.set(s.installInProgress)
	assert.Equal(t, "INSTALL_IN_PROGRESS", s.State())
	assert.Error(t, s.Register())
	assert.Equal(t, "INACTIVE", s.State())
}

//nolint:dupl
func TestUninstallError(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		return fmt.Errorf("Uninstallation failed")
	}
	s := New(context.TODO(), orchestratorClient, "", "", execute)

	s.set(s.uninstallInProgress)
	assert.Equal(t, "UNINSTALL_IN_PROGRESS", s.State())
	assert.Error(t, s.Deregister())
	assert.Equal(t, "INACTIVE", s.State())
}

func TestUninstallCmdNotCached(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		return nil
	}
	s := New(context.TODO(), orchestratorClient, "", "", execute)
	s.set(s.active)
	s.uninstallCmd = ""
	s.cleanupCmd = "echo Hello Test World"

	assert.Equal(t, "ACTIVE", s.State())
	assert.NoError(t, s.Deregister())
	assert.Equal(t, "INACTIVE", s.State())
	assert.Equal(t, s.installCmd, "installCmd")
	assert.Equal(t, s.uninstallCmd, "uninstallCmd")
}

func TestUninstallCmdLvmCleanUpFailure(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		if command == "uninstallCmd" {
			return nil
		} else {
			return fmt.Errorf("LVM clean up failed")
		}
	}
	s := New(context.TODO(), orchestratorClient, "", "", execute)
	s.set(s.active)
	s.cleanupCmd = "echo Hello Test World"

	assert.Equal(t, "ACTIVE", s.State())
	assert.Error(t, s.Deregister())
	assert.Equal(t, "INACTIVE", s.State())
}

func TestInactiveDeregister(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		return nil
	}

	s := New(context.TODO(), orchestratorClient, "", "", execute)
	s.cleanupCmd = "echo Hello Test World"
	assert.Equal(t, "INACTIVE", s.State())
	assert.NoError(t, s.Deregister())
	assert.Equal(t, "INACTIVE", s.State())
}

func TestFullFlowTwice(t *testing.T) {
	orchestratorClient := &comms.Client{}
	orchestratorClient.RegisterToClusterOrch = func(ctx context.Context, guid string) (string, string) {
		return "installCmd", "uninstallCmd"
	}

	execute := func(ctx context.Context, command string) error {
		return nil
	}
	s := New(context.TODO(), orchestratorClient, "", "", execute)
	s.cleanupCmd = "echo Hello Test World"

	assert.Equal(t, "INACTIVE", s.State())
	assert.NoError(t, s.Register())

	assert.Equal(t, "ACTIVE", s.State())
	assert.NoError(t, s.Deregister())

	assert.Equal(t, "INACTIVE", s.State())
	assert.NoError(t, s.Register())

	assert.Equal(t, "ACTIVE", s.State())
	assert.NoError(t, s.Deregister())

	assert.Equal(t, "INACTIVE", s.State())
}
