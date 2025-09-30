// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ubuntu

import (
	"errors"
	"testing"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func newTestVerifier(
	undoErr, deleteSnapErr, rebootErr, removeFileErr error,
	networkOK bool,
) *Verifier {
	return &Verifier{
		CommandExecutor: &mockExecutor{},
		fs:              afero.NewMemMapFs(),
		CheckNetworkConnectionFunc: func(_ common.Executor) bool {
			return networkOK
		},
		UndoChangeFunc: func(_ common.Executor, _ int) error {
			return undoErr
		},
		DeleteSnapshotFunc: func(_ common.Executor, _ int) error {
			return deleteSnapErr
		},
		rebootSystemFunc: func(_ common.Executor) error {
			return rebootErr
		},
		RemoveFileFunc: func(_ afero.Fs, _ string) error {
			return removeFileErr
		},
	}
}

func TestVerifier_VerifyUpdateAfterReboot(t *testing.T) {
	t.Run("Network OK", func(t *testing.T) {
		v := newTestVerifier(nil, nil, nil, nil, true)
		state := utils.INBDState{SnapshotNumber: 1}
		err := v.VerifyUpdateAfterReboot(state)
		assert.NoError(t, err)
	})

	t.Run("No Network, All Success", func(t *testing.T) {
		v := newTestVerifier(nil, nil, nil, nil, false)
		state := utils.INBDState{SnapshotNumber: 2}
		err := v.VerifyUpdateAfterReboot(state)
		assert.NoError(t, err)
	})

	t.Run("No Network, Undo Fails", func(t *testing.T) {
		v := newTestVerifier(errors.New("undo failed"), nil, nil, nil, false)
		state := utils.INBDState{SnapshotNumber: 3}
		err := v.VerifyUpdateAfterReboot(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undo failed")
	})

	t.Run("No Network, Delete Snapshot Fails", func(t *testing.T) {
		v := newTestVerifier(nil, errors.New("delete snapshot failed"), nil, nil, false)
		state := utils.INBDState{SnapshotNumber: 4}
		err := v.VerifyUpdateAfterReboot(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete snapshot failed")
	})

	t.Run("No Network, Reboot Fails", func(t *testing.T) {
		v := newTestVerifier(nil, nil, errors.New("reboot failed"), nil, false)
		state := utils.INBDState{SnapshotNumber: 5}
		err := v.VerifyUpdateAfterReboot(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reboot failed")
	})
}
