/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"log"
	"os/exec"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
)

// Verifier is the concrete implementation of the Verifier interface
type Verifier struct {
	CommandExecutor            common.Executor
	fs                         afero.Fs
	CheckNetworkConnectionFunc func(common.Executor) bool
	UndoChangeFunc             func(common.Executor, int) error
	DeleteSnapshotFunc         func(common.Executor, int) error
	rebootSystemFunc           func(common.Executor) error
	RemoveFileFunc             func(afero.Fs, string) error
}

// NewVerifier creates a new instance of Verifier with a command executor and file system.
func NewVerifier() *Verifier {
	return &Verifier{
		CommandExecutor:            common.NewExecutor(exec.Command, common.ExecuteAndReadOutput),
		fs:                         afero.NewOsFs(),
		CheckNetworkConnectionFunc: CheckNetworkConnection,
		UndoChangeFunc:             UndoChange,
		DeleteSnapshotFunc:         DeleteSnapshot,
		rebootSystemFunc:           utils.RebootSystem,
		RemoveFileFunc:             utils.RemoveFile,
	}
}

// VerifyUpdateAfterReboot verifies the update after a reboot.
// It checks if the state file exists and compares the previous and current versions.
// If the versions are different, it commits the update; otherwise, it reverts to the previous image.
func (v *Verifier) VerifyUpdateAfterReboot(state utils.INBDState) error {
	snapshotNumber := state.SnapshotNumber
	log.Printf("Snapshot number: %v", snapshotNumber)

	// Network check
	if !v.CheckNetworkConnectionFunc(v.CommandExecutor) {
		log.Println("No network connection detected.  Reverting to previous snapshot.")
		if err := v.UndoChangeFunc(v.CommandExecutor, snapshotNumber); err != nil {
			log.Printf("Failed to revert to previous snapshot: %v", err)
			return err
		}
		if err := v.DeleteSnapshotFunc(v.CommandExecutor, snapshotNumber); err != nil {
			log.Printf("Failed to delete snapshot: %v", err)
			return err
		}
		if err := v.rebootSystemFunc(v.CommandExecutor); err != nil {
			log.Printf("Failed to reboot system: %v", err)
			return err
		}
		log.Println("Reverted to previous snapshot and deleted it.")
		return nil

	}
	log.Println("Network connection detected.  Proceeding with update verification.")

	// Remove state file after checks
	err := v.RemoveFileFunc(v.fs, utils.StateFilePath)
	if err != nil {
		log.Printf("[Warning] Error removing state file: %v", err)
	}

	return nil
}
