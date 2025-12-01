/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/os_updater/emt"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/encoding/protojson"
)

// Updater is the concrete implementation of the Updater interface
// for the Ubuntu OS.
type Updater struct {
	CommandExecutor         common.Executor
	Request                 *pb.UpdateSystemSoftwareRequest
	GetFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error)
	Fs                      afero.Fs
}

// Update method for Ubuntu
func (u *Updater) Update() (bool, error) {
	fs := u.Fs
	if fs == nil {
		fs = afero.NewOsFs()
	}

	// Get the request details for logging
	jsonString, err := protojson.Marshal(u.Request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	// Set the environment variable DEBIAN_FRONTEND to non-interactive
	err = os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	if err != nil {
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INBM, "ubuntu")
		return false, fmt.Errorf("SOTA Aborted: Failed to set environment variable: %v", err)
	}

	err = os.Setenv("PATH", os.Getenv("PATH")+":/usr/bin:/bin")
	if err != nil {
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INBM, "ubuntu")
		return false, fmt.Errorf("SOTA Aborted: Failed to set environment variable: %v", err)
	}

	isUpdateAvail, updateSize, err := GetEstimatedSize(u.CommandExecutor, u.Request.PackageList)
	if err != nil {
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_DOWNLOAD, "ubuntu")
		return false, fmt.Errorf("SOTA Aborted: Update Failed: %s", err)
	}
	if !isUpdateAvail {
		log.Println("No update available.  System is up to date.")
		// If specific packages were requested and they're already installed, that's success
		if len(u.Request.PackageList) > 0 && u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
			log.Println("Package(s) already installed - treating as success")
			// Write state file for post-boot verification even if already installed
			if err := writeStateFileForPackageInstallation(fs, u.Request.PackageList); err != nil {
				log.Printf("WARNING: Failed to write state file for package installation: %v", err)
			}
			emt.WriteUpdateStatus(fs, emt.SUCCESS, string(jsonString), "")
			emt.WriteGranularLogWithOSType(fs, emt.SUCCESS, "", "ubuntu")
			return true, nil
		}
		return false, nil
	}

	log.Printf("Estimated update size: %d bytes", updateSize)

	freeSpace, err := u.GetFreeDiskSpaceInBytes("/", unix.Statfs)
	if err != nil {
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INSUFFICIENT_STORAGE, "ubuntu")
		return false, fmt.Errorf("SOTA Aborted: Failed to get free disk space: %v", err)
	}
	log.Printf("Free disk space: %d bytes", freeSpace)
	if freeSpace < updateSize {
		err := fmt.Errorf("SOTA Aborted: Not enough free disk space.  Free: %d bytes, Required: %d bytes", freeSpace, updateSize)
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INSUFFICIENT_STORAGE, "ubuntu")
		return false, err
	}

	// Take snapshot before applying updates (for FULL and NO_DOWNLOAD modes)
	// Skip snapshot for package-only installations as they don't need rollback capability
	if u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL ||
		u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
		// Only take snapshot for system-wide upgrades, not for specific package installations
		if len(u.Request.PackageList) == 0 {
			log.Println("Save snapshot before applying the update.")
			if err := NewSnapshotter(u.CommandExecutor, fs).Snapshot(); err != nil {
				errMsg := fmt.Sprintf("Error taking snapshot: %v", err)
				emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), errMsg)
				emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INBM, "ubuntu")
				return false, fmt.Errorf("failed to take snapshot before applying the update: %v", err)
			}
		} else {
			log.Printf("Skipping snapshot for specific package installation: %v", u.Request.PackageList)
		}
	}

	var cmds [][]string
	switch u.Request.Mode {
	case pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL:
		cmds = fullInstall(u.Request.PackageList)
	case pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD:
		cmds = noDownload(u.Request.PackageList)
	case pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY:
		cmds = downloadOnly(u.Request.PackageList)
	default:
		return false, fmt.Errorf("SOTA Aborted: Invalid mode")
	}

	for _, cmd := range cmds {
		log.Printf("Executing command: %s", cmd)
		_, stderr, err := u.CommandExecutor.Execute(cmd)
		if err != nil {
			emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
			emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_UPDATE_TOOL, "ubuntu")
			return false, fmt.Errorf("SOTA Aborted: Command execution error: %v", err)
		}
		if len(stderr) > 0 {
			errMsg := fmt.Sprintf("SOTA Aborted: Command failed: %s", string(stderr))
			emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), errMsg)
			emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_UPDATE_TOOL, "ubuntu")
			return false, fmt.Errorf("%s", errMsg)
		}
	}

	// For NO_DOWNLOAD mode with packages, write state file before SUCCESS
	if u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD && len(u.Request.PackageList) > 0 {
		log.Println("State file check: NO_DOWNLOAD mode with packages, writing state file")
		if err := writeStateFileForPackageInstallation(fs, u.Request.PackageList); err != nil {
			log.Printf("WARNING: Failed to write state file for package installation: %v", err)
		}
	}

	// Success - write success status (will be verified after reboot)
	emt.WriteUpdateStatus(fs, emt.SUCCESS, string(jsonString), "")
	emt.WriteGranularLogWithOSType(fs, emt.SUCCESS, "", "ubuntu")

	return true, nil
}

// GetEstimatedSize returns the estimated size of the update
// and whether an update is available.
func GetEstimatedSize(cmdExec common.Executor, packageList []string) (bool, uint64, error) {
	var cmd []string

	// If specific packages are requested, check those; otherwise check system-wide upgrade
	if len(packageList) > 0 {
		// For specific packages: apt-get install --dry-run <packages>
		cmd = append([]string{common.AptGetCmd, "-o", "Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold", "-u", "install", "--assume-no"}, packageList...)
	} else {
		// For system-wide upgrade
		cmd = []string{common.AptGetCmd, "-o", "Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold", "--with-new-pkgs", "-u", "upgrade", "--assume-no"}
	}

	// Ignore the error as the command will return a non-zero exit code
	stdout, stderr, err := cmdExec.Execute(cmd)
	if err != nil {
		// Log the error, but continue processing as output may still be useful
		log.Printf("Warning: command execution returned error: %v", err)
	}
	if len(stderr) > 0 {
		return false, 0, fmt.Errorf("SOTA Aborted: command execution for update size determination failed: %s", string(stderr))
	}
	if len(stdout) == 0 {
		return false, 0, fmt.Errorf("SOTA Aborted: no output from command to determine update size")
	}

	return getEstimatedSizeInBytesFromAptGetUpgrade(string(stdout))
}

func sizeToBytes(size string, unit string) uint64 {
	parsedSize, err := strconv.ParseFloat(size, 64)
	if err != nil {
		log.Printf("Error parsing size: %v", err)
		return 0
	}

	switch unit {
	case "kB":
		return uint64(parsedSize * 1024)
	case "MB":
		return uint64(parsedSize * 1024 * 1024)
	case "GB":
		return uint64(parsedSize * 1024 * 1024 * 1024)
	default:
		return uint64(parsedSize)
	}
}

const noUpdateAvailable = "0 upgraded, 0 newly installed, 0 to remove"

func getEstimatedSizeInBytesFromAptGetUpgrade(upgradeOutput string) (bool, uint64, error) {
	log.Printf("Apt-get upgrade output: %s", upgradeOutput)
	var outputLines []string
	for line := range strings.SplitSeq(upgradeOutput, "\n") {
		if strings.Contains(line, "After this operation,") {
			outputLines = append(outputLines, line)
		} else if strings.Contains(line, noUpdateAvailable) {
			// No update available.  System is up to date
			return false, 0, nil
		}

	}
	output := strings.Join(outputLines, "\n")

	updateRegex := regexp.MustCompile(`(\d+(?:,\d+)*(\.\d+)?)(\s*(kB|B|MB|GB)).*(freed|used)`)
	matches := updateRegex.FindStringSubmatch(output)

	if matches == nil {
		return false, 0, fmt.Errorf("failed to get size of the update")
	}

	freedOrUsed := matches[5]

	if freedOrUsed == "used" {
		sizeString := strings.ReplaceAll(matches[1], ",", "")
		return true, sizeToBytes(sizeString, matches[4]), nil
	}

	log.Println("Update will free some size on disk")
	return true, 0, nil
}

func noDownload(packages []string) [][]string {
	log.Println("No download mode")
	cmds := [][]string{
		{common.DpkgCmd, "--configure", "-a", "--force-confdef", "--force-confold"},
		{common.AptGetCmd, "-o", "Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold", "-yq", "-f", "install"},
	}

	if len(packages) == 0 {
		cmds = append(cmds, []string{common.AptGetCmd, "-o",
			"Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold",
			"--with-new-pkgs", "--fix-missing", "-yq", "upgrade"})
	} else {
		cmds = append(cmds, [][]string{append([]string{common.AptGetCmd, "-o",
			"Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold",
			"--fix-missing", "-yq",
			"install"}, packages...)}...)
	}

	return cmds
}

// writeStateFileForPackageInstallation writes state file for package-only installations
func writeStateFileForPackageInstallation(fs afero.Fs, packageList []string) error {
	if len(packageList) == 0 {
		return nil
	}

	log.Printf("Writing state file for package installation: %v", packageList)

	// Check if state file already exists (from previous kernel/OS update)
	existingState, err := utils.ReadStateFile(fs, utils.StateFilePath)
	if err == nil && existingState.SnapshotNumber > 0 {
		log.Printf("State file exists from previous update. Preserving snapshot info and adding packages.")
		existingState.PackageList = strings.Join(packageList, ",")
		stateJSON, _ := json.Marshal(existingState)
		return utils.WriteToStateFile(fs, utils.StateFilePath, string(stateJSON))
	}

	// No existing state - create new one for package installation
	state := utils.INBDState{
		RestartReason: "package_installation",
		PackageList:   strings.Join(packageList, ","),
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := utils.WriteToStateFile(fs, utils.StateFilePath, string(stateJSON)); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	log.Printf("State file written successfully: %s", string(stateJSON))

	// Verify the file was written
	if data, err := afero.ReadFile(fs, utils.StateFilePath); err != nil {
		log.Printf("WARNING: State file verification failed: %v", err)
	} else {
		log.Printf("State file verification successful. Content: %s", string(data))
	}

	return nil
}

func downloadOnly(packages []string) [][]string {
	log.Println("Download only mode")

	cmds := [][]string{
		{common.DpkgCmd, "--configure", "-a", "--force-confdef", "--force-confold"},
		{common.AptGetCmd, "update"},
	}

	if len(packages) == 0 {
		cmds = append(cmds, []string{common.AptGetCmd, "-o",
			"Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold",
			"--with-new-pkgs", "--download-only",
			"--fix-missing", "-yq", "upgrade"})
	} else {
		cmds = append(cmds, [][]string{append([]string{common.AptGetCmd, "-o",
			"Dpkg::Options::=--force-confdef", "-o",
			"Dpkg::Options::=--force-confold", "--download-only",
			"--fix-missing", "-yq", "install"}, packages...)}...)
	}

	return cmds
}

func fullInstall(packages []string) [][]string {
	log.Println("Download and install mode")

	cmds := [][]string{
		{common.AptGetCmd, "update"},
		{common.AptGetCmd, "-yq", "-f", "install"}, // Fix broken dependencies
		{common.DpkgCmd, "--configure", "-a", "--force-confdef", "--force-confold"},
	}

	if len(packages) == 0 {
		cmds = append(cmds, []string{common.AptGetCmd, "-yq", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", "--with-new-pkgs", "upgrade"})
	} else {
		cmds = append(cmds, []string{common.AptGetCmd, "-yq", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", "install"})
		cmds = append(cmds, packages)
	}

	return cmds
}
