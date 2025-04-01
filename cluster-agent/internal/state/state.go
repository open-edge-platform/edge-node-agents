// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// package state implements thread safe state machine which represents Cluster Agent overall progress on k8s bootstrap
package state

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/logger"
)

var ErrIncorrectActionRequest = errors.New("incorrect ActionRequest")
var log = logger.Logger

type State interface {
	Register() error
	Deregister() error
	State() string
}

// StateMachine represents Cluster Agent overall progress. It can be found here
// http://edge-node.app.intel.com/docs/implementation-designs/bare_metal_agents/cluster-agent-design/#cluster-agent-state-machine
// ERROR should be used to indicate persistent errors from which CA can't recover
type StateMachine struct {
	inactive            State
	registering         State
	installInProgress   State
	active              State
	deregistering       State
	uninstallInProgress State
	currentState        State

	ctx          context.Context
	client       *comms.Client
	guid         string
	installCmd   string
	uninstallCmd string
	cleanupCmd   string

	mu sync.RWMutex
}

func New(ctx context.Context, c *comms.Client, guid string, accessTokenPath string, execute func(ctx context.Context, command string) error) *StateMachine {
	sm := &StateMachine{ctx: ctx, client: c, guid: guid}
	sm.inactive = &Inactive{sm: sm}
	sm.registering = &Registering{sm: sm, tF: accessTokenPath}
	sm.installInProgress = &InstallInProgress{sm: sm, execute: execute}
	sm.active = &Active{sm: sm}
	sm.deregistering = &Deregistering{sm: sm, tF: accessTokenPath}
	sm.uninstallInProgress = &UninstallInProgress{sm: sm, execute: execute}

	sm.currentState = sm.inactive
	sm.cleanupCmd = "for lvname in $(sudo lvs --noheadings -o lv_name lvmvg); do sudo lvremove /dev/lvmvg/${lvname} -y; done;"
	return sm
}

func (sm *StateMachine) Register() error {
	return sm.currentState.Register()
}

func (sm *StateMachine) Deregister() error {
	return sm.currentState.Deregister()
}

func (sm *StateMachine) State() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.currentState.State()
}

func (sm *StateMachine) set(s State) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log.Infof("Changing Cluster Agent state from %s to %s", sm.currentState.State(), s.State())
	sm.currentState = s
}

func (sm *StateMachine) incorrectActionRequest() error {
	return fmt.Errorf("%w for current state: %s", ErrIncorrectActionRequest, sm.currentState.State())
}
