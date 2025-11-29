/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package ubuntu updates the Ubuntu OS.
package ubuntu

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
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
	GetEstimatedSize        func(cmdExec common.Executor) (bool, uint64, error)
	GetFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error)
}

// Update method for Ubuntu
func (u *Updater) Update() (bool, error) {
	fs := afero.NewOsFs()

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

	isUpdateAvail, updateSize, err := u.GetEstimatedSize(u.CommandExecutor)
	if err != nil {
		emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), err.Error())
		emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_DOWNLOAD, "ubuntu")
		return false, fmt.Errorf("SOTA Aborted: Update Failed: %s", err)
	}
	if !isUpdateAvail {
		log.Println("No update available.  System is up to date.")
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
	if u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL ||
		u.Request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
		log.Println("Save snapshot before applying the update.")
		if err := NewSnapshotter(u.CommandExecutor, fs).Snapshot(); err != nil {
			errMsg := fmt.Sprintf("Error taking snapshot: %v", err)
			emt.WriteUpdateStatus(fs, emt.FAIL, string(jsonString), errMsg)
			emt.WriteGranularLogWithOSType(fs, emt.FAIL, emt.FAILURE_REASON_INBM, "ubuntu")
			return false, fmt.Errorf("failed to take snapshot before applying the update: %v", err)
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

	// Success - write success status (will be verified after reboot)
	emt.WriteUpdateStatus(fs, emt.SUCCESS, string(jsonString), "")
	emt.WriteGranularLogWithOSType(fs, emt.SUCCESS, "", "ubuntu")

	return true, nil
}

// GetEstimatedSize returns the estimated size of the update
// and whether an update is available.
func GetEstimatedSize(cmdExec common.Executor) (bool, uint64, error) {
	cmd := []string{common.AptGetCmd, "-o", "Dpkg::Options::='--force-confdef'", "-o",
		"Dpkg::Options::='--force-confold'", "--with-new-pkgs", "-u", "upgrade", "--assume-no"}

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
