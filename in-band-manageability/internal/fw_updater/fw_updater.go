/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater updates the firmware.
package fwupdater

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	common "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/common"
	telemetry "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry"
	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// HardwareInfoProvider defines an interface for getting hardware information
type HardwareInfoProvider interface {
	GetHardwareInfo() (*pb.HardwareInfo, error)
	GetFirmwareInfo() (*pb.FirmwareInfo, error)
}

// DefaultHardwareInfoProvider implements HardwareInfoProvider using real telemetry
type DefaultHardwareInfoProvider struct{}

func (d *DefaultHardwareInfoProvider) GetHardwareInfo() (*pb.HardwareInfo, error) {
	return telemetry.GetHardwareInfo()
}

func (d *DefaultHardwareInfoProvider) GetFirmwareInfo() (*pb.FirmwareInfo, error) {
	return telemetry.GetFirmwareInfo()
}

// FWUpdater is the main struct that contains the methods to update the firmware.
type FWUpdater struct {
	req        *pb.UpdateFirmwareRequest
	fs         afero.Fs
	hwProvider HardwareInfoProvider
}

// NewFWUpdater creates a new FWUpdater instance.
func NewFWUpdater(req *pb.UpdateFirmwareRequest) *FWUpdater {
	return &FWUpdater{
		req:        req,
		fs:         afero.NewOsFs(),                // Use real filesystem by default
		hwProvider: &DefaultHardwareInfoProvider{}, // Use real hardware info by default
	}
}

// NewFWUpdaterWithFS creates a new FWUpdater instance with a custom filesystem.
// This is primarily used for testing with mocked filesystems.
func NewFWUpdaterWithFS(req *pb.UpdateFirmwareRequest, fs afero.Fs) *FWUpdater {
	return &FWUpdater{
		req:        req,
		fs:         fs,
		hwProvider: &DefaultHardwareInfoProvider{}, // Use real hardware info by default
	}
}

// NewFWUpdaterWithMocks creates a new FWUpdater instance with custom filesystem and hardware provider.
// This is primarily used for testing with mocked dependencies.
func NewFWUpdaterWithMocks(req *pb.UpdateFirmwareRequest, fs afero.Fs, hwProvider HardwareInfoProvider) *FWUpdater {
	return &FWUpdater{
		req:        req,
		fs:         fs,
		hwProvider: hwProvider,
	}
}

// UpdateFirmware updates the firmware based on the request.
func (u *FWUpdater) UpdateFirmware() (*pb.UpdateResponse, error) {
	log.Println("Starting firmware update process.")

	// Validate hash algorithm, default to sha384 if not provided
	finalHashAlgorithm := "sha384"
	if u.req.HashAlgorithm != "" {
		switch strings.ToLower(u.req.HashAlgorithm) {
		case "sha256", "sha384", "sha512":
			finalHashAlgorithm = strings.ToLower(u.req.HashAlgorithm)
		default:
			return &pb.UpdateResponse{
				StatusCode: 400,
				Error:      "invalid hash algorithm: must be 'sha256', 'sha384', or 'sha512'",
			}, nil
		}
	}

	hwInfo, err := u.hwProvider.GetHardwareInfo()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	log.Printf("platform name: %+v", hwInfo.GetSystemProductName())
	// Get the firmware update tool info.
	firmwareToolInfo, err := GetFirmwareUpdateToolInfo(u.fs, hwInfo.GetSystemProductName())
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}
	log.Printf("Firmware update tool info: %+v", firmwareToolInfo)

	// Get the firmware information for the release date check.
	fwInfo, err := u.hwProvider.GetFirmwareInfo()
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	// Check if the firmware update is required.
	// Compare timestamps properly: if current BIOS date is newer than or equal to requested date, skip update
	currentBiosTime := fwInfo.GetBiosReleaseDate().AsTime()
	requestedTime := u.req.ReleaseDate.AsTime()

	if currentBiosTime.After(requestedTime) || currentBiosTime.Equal(requestedTime) {
		return &pb.UpdateResponse{
			StatusCode: 400,
			Error: fmt.Sprintf("Firmware update is not required. Current firmware (%s) is up to date or newer than requested (%s).",
				currentBiosTime.Format(time.RFC3339), requestedTime.Format(time.RFC3339)),
		}, nil
	}

	// Download the firmware update file.
	// TODO: Download needs to support signature checking
	// and username and password for private repositories.
	log.Printf("Downloading firmware update from URL: %s", u.req.Url)
	downloader := NewDownloader(u.req)
	if err := downloader.download(); err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	// Get the downloaded firmware file path
	firmwareFilePath := filepath.Join(utils.IntelManageabilityCachePathPrefix, filepath.Base(u.req.Url))

	// Verify signature if provided
	if u.req.Signature != "" {
		log.Printf("Verifying signature for downloaded firmware package: %s", firmwareFilePath)
		if err := utils.VerifySignature(
			u.req.Signature,
			firmwareFilePath,
			utils.ParseHashAlgorithm(finalHashAlgorithm),
		); err != nil {
			// Clean up downloaded file on signature verification failure
			if removeErr := u.fs.Remove(firmwareFilePath); removeErr != nil {
				log.Printf("Warning: failed to remove invalid firmware file %s: %v", firmwareFilePath, removeErr)
			}
			return &pb.UpdateResponse{StatusCode: 400, Error: fmt.Sprintf("Signature verification failed: %v", err)}, nil //nolint:nilerr // gRPC response pattern
		}
		log.Printf("Signature verification passed for firmware package.")
	} else {
		log.Printf("No signature provided, proceeding without signature verification.")
	}

	// Extract firmware file info and unpack if needed
	fwFile, certFile, err := u.extractFileInfo(firmwareFilePath, utils.IntelManageabilityCachePathPrefix)
	if err != nil {
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	// Perform the firmware update using the extracted firmware file and the firmware update tool info
	actualFirmwarePath := filepath.Join(utils.IntelManageabilityCachePathPrefix, fwFile)
	if err := u.applyFirmware(actualFirmwarePath, firmwareToolInfo); err != nil {
		// Clean up files before returning error
		u.deleteFiles(filepath.Base(u.req.Url), fwFile, certFile)
		return &pb.UpdateResponse{StatusCode: 500, Error: err.Error()}, nil //nolint:nilerr // gRPC response pattern
	}

	log.Println("Update completed successfully.")

	// Remove the artifacts after update success
	u.deleteFiles(filepath.Base(u.req.Url), fwFile, certFile)

	// Check if reboot is requested
	if !u.req.DoNotReboot {
		log.Println("Firmware update completed successfully. Rebooting system...")
		executor := common.NewExecutor(exec.Command, common.ExecuteAndReadOutput)
		if err := utils.RebootSystem(executor); err != nil {
			log.Printf("Warning: Failed to reboot system: %v", err)
			// Don't return error here as firmware update was successful
		}
	} else {
		log.Println("Firmware update completed successfully. Reboot skipped as requested.")
	}

	return &pb.UpdateResponse{StatusCode: 200, Error: "Success"}, nil
}

// applyFirmware applies the firmware update using the firmware tool
func (u *FWUpdater) applyFirmware(firmwareFilePath string, toolInfo FirmwareToolInfo) error {
	log.Printf("Applying firmware using tool: %s", toolInfo.FirmwareTool)

	// Check if firmware tool exists
	if strings.Contains(toolInfo.FirmwareTool, "/") {
		if !utils.IsFileExist(u.fs, toolInfo.FirmwareTool) {
			return fmt.Errorf("firmware update aborted: firmware tool does not exist at %s", toolInfo.FirmwareTool)
		}
	}

	// Check firmware tool if check args are provided
	if toolInfo.FirmwareToolCheckArgs != "" {
		checkCmd := exec.Command(toolInfo.FirmwareTool, strings.Fields(toolInfo.FirmwareToolCheckArgs)...)
		if output, err := checkCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("firmware update aborted: firmware tool check failed: %s", string(output))
		}
	}

	// Build the command : fw_tool + fw_tool_args + guid + fw_file + tool_options
	var cmdArgs []string

	// Add tool arguments if present
	if toolInfo.FirmwareToolArgs != "" {
		cmdArgs = append(cmdArgs, strings.Fields(toolInfo.FirmwareToolArgs)...)
	}

	// check GUID
	if toolInfo.GUID {
		guid, err := u.getGuidFromSystem(toolInfo.FirmwareTool, "")
		if err != nil {
			return fmt.Errorf("failed to get GUID from system: %w", err)
		}
		cmdArgs = append(cmdArgs, guid)
	}

	// Add firmware file path
	cmdArgs = append(cmdArgs, firmwareFilePath)

	// Add tool options if supported by the tool
	if toolInfo.ToolOptions {
		// In this implementation, we don't have specific tool options from the request
		// This could be extended in the future if needed
		log.Println("Tool supports options, but none provided in request")
	}

	cmd := exec.Command(toolInfo.FirmwareTool, cmdArgs...)

	log.Printf("Executing firmware update command: %s %v", toolInfo.FirmwareTool, cmdArgs)

	// Special handling for afulnx_64 tool
	if strings.Contains(toolInfo.FirmwareTool, "afulnx") {
		log.Println("Device will be rebooting upon successful firmware install.")
	}

	// Execute the command
	output, err := cmd.CombinedOutput()
	log.Printf("Firmware tool output: %s", string(output))

	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = "Firmware command failed"
		}
		return fmt.Errorf("firmware update failed: %s", errMsg)
	}

	log.Println("Apply firmware command successful.")
	return nil
}

// getGuidFromSystem extracts the GUID from the system using the firmware tool
func (u *FWUpdater) getGuidFromSystem(firmwareTool, manifestGuid string) (string, error) {
	// Extract GUIDs from the system using firmware tool
	extractedGuids, err := u.extractGuids(firmwareTool, []string{"System Firmware type", "system-firmware type"})
	if err != nil {
		return "", err
	}

	if manifestGuid != "" {
		// Check if manifest GUID matches any system firmware GUID
		for _, guid := range extractedGuids {
			if guid == manifestGuid {
				return manifestGuid, nil
			}
		}
		return "", fmt.Errorf("GUID in manifest does not match any system firmware GUID on the system")
	} else {
		// Return the first extracted GUID
		if len(extractedGuids) > 0 {
			return extractedGuids[0], nil
		}
		return "", fmt.Errorf("no GUIDs found")
	}
}

// extractGuids extracts firmware GUIDs from the system using the firmware tool
func (u *FWUpdater) extractGuids(firmwareTool string, types []string) ([]string, error) {
	// Run firmware tool with -l flag to list GUIDs
	cmd := exec.Command(firmwareTool, "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("firmware update aborted: failed to list GUIDs: %s", string(output))
	}

	// Parse GUIDs from output
	guids := u.parseGuids(string(output), types)
	log.Printf("Found GUIDs: %v", guids)

	if len(guids) == 0 {
		return nil, fmt.Errorf("firmware update aborted: no GUIDs found matching types: %v", types)
	}

	return guids, nil
}

// parseGuids parses the shell command output to retrieve GUID values
func (u *FWUpdater) parseGuids(output string, types []string) []string {
	var guids []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Check if any of the specified types are in the line
		for _, typeStr := range types {
			if strings.Contains(line, typeStr) {
				// Split by comma and extract the GUID (second part)
				parts := strings.Split(line, ",")
				if len(parts) >= 2 {
					// Extract GUID, remove curly braces and whitespace
					guidPart := strings.Fields(parts[1])
					if len(guidPart) > 0 {
						guidStr := strings.Trim(guidPart[0], "{}")
						// Validate that this looks like a GUID (8-4-4-4-12 hex format)
						if len(guidStr) == 36 && strings.Count(guidStr, "-") == 4 {
							guids = append(guids, guidStr)
						}
					}
				}
				break
			}
		}
	}

	return guids
}

// extractFileInfo checks the file extension and extracts files if needed
func (u *FWUpdater) extractFileInfo(pkgFilePath, repoPath string) (string, string, error) {
	pkgFilename := filepath.Base(pkgFilePath)

	ext := u.extractFileExt(pkgFilename)
	log.Printf("File extension detected: %s for file: %s", ext, pkgFilename)

	if ext == "package" || ext == "bios" {
		// File doesn't need extraction
		return pkgFilename, "", nil
	} else {
		// File needs to be unpacked
		return u.unpackFile(repoPath, pkgFilename)
	}
}

// extractFileExt determines the file type based on extension
func (u *FWUpdater) extractFileExt(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) == 0 {
		return ""
	}

	ext := strings.ToLower(parts[len(parts)-1])

	switch ext {
	case "fv", "cap", "bio":
		return "package"
	case "cert", "pem", "crt":
		return "cert"
	case "bin":
		return "bios"
	default:
		return ""
	}
}

// unpackFile extracts tar files and returns firmware and cert filenames
func (u *FWUpdater) unpackFile(repoPath, pkgFilename string) (string, string, error) {
	log.Printf("Unpacking file: %s in directory: %s", pkgFilename, repoPath)

	pkgPath := filepath.Join(repoPath, pkgFilename)
	cmd := exec.Command("tar", "-xvf", pkgPath, "--no-same-owner", "-C", repoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("firmware update aborted: invalid file sent. error: %s", string(output))
	}

	fwFile, certFile := u.getFilesFromTarOutput(string(output))
	log.Printf("Extracted files - firmware: %s, cert: %s", fwFile, certFile)

	return fwFile, certFile, nil
}

// getFilesFromTarOutput extracts firmware and cert filenames from tar output
func (u *FWUpdater) getFilesFromTarOutput(output string) (string, string) {
	var fwFile, certFile string

	lines := strings.Split(output, "\n")
	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	uniqueLines := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			seen[line] = true
			uniqueLines = append(uniqueLines, line)
		}
	}

	for _, line := range uniqueLines {
		ext := u.extractFileExt(line)
		switch ext {
		case "package", "bios":
			fwFile = line
		case "cert":
			certFile = line
		}
	}

	return fwFile, certFile
}

// deleteFiles removes the downloaded and extracted files
func (u *FWUpdater) deleteFiles(pkgFilename, fwFilename, certFilename string) {
	log.Printf("Cleaning up files: pkg=%s, fw=%s, cert=%s", pkgFilename, fwFilename, certFilename)

	// Delete package file
	if pkgFilename != "" {
		pkgPath := filepath.Join(utils.IntelManageabilityCachePathPrefix, pkgFilename)
		if err := u.fs.Remove(pkgPath); err != nil {
			log.Printf("Warning: failed to delete package file %s: %v", pkgPath, err)
		} else {
			log.Printf("Deleted package file: %s", pkgPath)
		}
	}

	// Delete firmware file
	if fwFilename != "" {
		fwPath := filepath.Join(utils.IntelManageabilityCachePathPrefix, fwFilename)
		if err := u.fs.Remove(fwPath); err != nil {
			log.Printf("Warning: failed to delete firmware file %s: %v", fwPath, err)
		} else {
			log.Printf("Deleted firmware file: %s", fwPath)
		}
	}

	// Delete certificate file
	if certFilename != "" {
		certPath := filepath.Join(utils.IntelManageabilityCachePathPrefix, certFilename)
		if err := u.fs.Remove(certPath); err != nil {
			log.Printf("Warning: failed to delete certificate file %s: %v", certPath, err)
		} else {
			log.Printf("Deleted certificate file: %s", certPath)
		}
	}
}
