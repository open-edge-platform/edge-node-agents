/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/emt"
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
	// Check for package installation verification first
	if state.PackageList != "" {
		log.Printf("Found PackageList in state file: %s", state.PackageList)
		if err := v.verifyPackageInstallation(state); err != nil {
			log.Printf("Package verification failed: %v", err)
			return err
		}
		// If only packages (no snapshot), we're done - remove state file
		if state.SnapshotNumber == 0 {
			log.Println("Package-only installation verified successfully")
			// Remove state file after successful verification
			if err := v.RemoveFileFunc(v.fs, utils.StateFilePath); err != nil {
				log.Printf("[Warning] Error removing state file: %v", err)
			}
			return nil
		}
		// Continue to OS verification if snapshot exists
		log.Println("Package verification complete, proceeding with OS verification")
	}

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

	// Write SUCCESS status to update log files
	emt.WriteUpdateStatus(v.fs, emt.SUCCESS, "", "")
	emt.WriteGranularLogWithOSType(v.fs, emt.SUCCESS, "", "ubuntu")

	// Remove state file after checks
	err := v.RemoveFileFunc(v.fs, utils.StateFilePath)
	if err != nil {
		log.Printf("[Warning] Error removing state file: %v", err)
	}

	return nil
}

// verifyPackageInstallation verifies that packages in PackageList are installed
func (v *Verifier) verifyPackageInstallation(state utils.INBDState) error {
	if state.PackageList == "" {
		return nil
	}

	log.Printf("Verifying package installation: %s", state.PackageList)
	packages := strings.Split(state.PackageList, ",")

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		// Check if package is installed using dpkg
		cmd := []string{common.DpkgCmd, "-l", pkg}
		stdout, stderr, err := v.CommandExecutor.Execute(cmd)

		if err != nil || len(stderr) > 0 {
			errMsg := fmt.Sprintf("Package %s verification failed: %v, stderr: %s", pkg, err, string(stderr))
			log.Println(errMsg)
			emt.WriteUpdateStatus(v.fs, emt.FAIL, "", errMsg)
			emt.WriteGranularLogWithOSType(v.fs, emt.FAIL, emt.FAILURE_REASON_UPDATE_TOOL, "ubuntu")
			return fmt.Errorf("%s", errMsg)
		}

		// Check if output contains "ii" status (installed)
		if !strings.Contains(string(stdout), "ii  "+pkg) {
			errMsg := fmt.Sprintf("Package %s not found in installed packages", pkg)
			log.Println(errMsg)
			emt.WriteUpdateStatus(v.fs, emt.FAIL, "", errMsg)
			emt.WriteGranularLogWithOSType(v.fs, emt.FAIL, emt.FAILURE_REASON_UPDATE_TOOL, "ubuntu")
			return fmt.Errorf("%s", errMsg)
		}

		log.Printf("Package %s verified successfully", pkg)
	}

	// All packages verified - write SUCCESS
	log.Println("All packages verified successfully")
	emt.WriteUpdateStatus(v.fs, emt.SUCCESS, "", "")
	emt.WriteGranularLogWithOSType(v.fs, emt.SUCCESS, "", "ubuntu")

	return nil
}
