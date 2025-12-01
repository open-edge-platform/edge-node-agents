/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	"github.com/spf13/afero"
)

// Log file paths
const (
	UpdateStatusLogPath = "/var/log/inbm-update-status.log"
	GranularLogPath     = "/var/log/inbm-update-log.log"
)

// UpdateStatus represents the structure of the update status log.
type UpdateStatus struct {
	Status   string `json:"Status"`
	Type     string `json:"Type"`
	Time     string `json:"Time"`
	Metadata string `json:"Metadata"`
	Error    string `json:"Error"`
	Version  string `json:"Version"`
}

// WriteUpdateStatus writes the update status to the log file (exported for use by other packages)
func WriteUpdateStatus(fs afero.Fs, status, metadata, errorDetails string) {
	// Create the update status log file if it does not exist.
	if _, err := os.Stat(UpdateStatusLogPath); os.IsNotExist(err) {
		file, err := os.Create(UpdateStatusLogPath)
		if err != nil {
			log.Printf("[Warning] Error writing update status: failed to create update status log file: %v", err)
			return
		}
		file.Close()
	}

	// Open the update status log file for writing and truncate it.
	file, err := utils.OpenFile(fs, UpdateStatusLogPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[Warning] Error writing update status: failed to open update status log file: %v", err)
		return
	}
	defer file.Close()

	// Create the JSON structure.
	updateStatus := UpdateStatus{
		Status:   status,
		Type:     "sota",
		Time:     time.Now().Format("2006-01-02 15:04:05"),
		Metadata: metadata,
		Error:    errorDetails,
		Version:  "v1",
	}

	// Marshal the JSON structure to a string.
	jsonData, err := json.MarshalIndent(updateStatus, "", "  ")
	if err != nil {
		log.Printf("[Warning] Error writing update status: failed to marshal JSON: %v", err)
		return
	}

	// Write the JSON data to the file.
	_, err = file.Write(jsonData)
	if err != nil {
		log.Printf("[Warning] Error writing update status log file: %v", err)
	}
}

// WriteGranularLog writes the granular update log (exported for use by other packages)
func WriteGranularLog(fs afero.Fs, statusDetail string, failureReason string) {
	WriteGranularLogWithOSType(fs, statusDetail, failureReason, "emt")
}

// WriteGranularLogWithOSType writes the granular update log with OS type specification
func WriteGranularLogWithOSType(fs afero.Fs, statusDetail string, failureReason string, osType string) {
	// Create the granular log file if it does not exist.
	if _, err := os.Stat(GranularLogPath); os.IsNotExist(err) {
		file, err := os.Create(GranularLogPath)
		if err != nil {
			log.Printf("[Warning] Error writing granular log: failed to create granular log file: %v", err)
			return
		}
		file.Close()
	}

	// Open the granular log file for writing and truncate it.
	file, err := utils.OpenFile(fs, GranularLogPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[Warning] Error writing granular log: failed to open granular log file: %v", err)
		return
	}
	defer file.Close()

	// If update is successful, get the version and write it to the granular log.
	// If update is not successful, write the failure reason to the granular log.
	var granularLogData map[string][]map[string]string
	if statusDetail == SUCCESS {
		var version string
		var err error

		if osType == "Ubuntu" {
			// For Ubuntu, we need to import and call ubuntu
			// Since we can't have circular imports, we'll check for /etc/os-release directly here
			version, err = getVersionForOS(fs, "/etc/os-release")
		} else {
			// For EMT, use /etc/image-id
			version, err = GetImageBuildDate(fs)
		}

		if err != nil || version == "" {
			log.Printf("[Warning] Error writing granular log: failed to get version: %v", err)
		}
		granularLogData = map[string][]map[string]string{
			"UpdateLog": {
				{
					"StatusDetail.Status": statusDetail,
					"Version":             version,
				},
			},
		}
	} else {
		// Create the JSON structure.
		granularLogData = map[string][]map[string]string{
			"UpdateLog": {
				{
					"StatusDetail.Status": statusDetail,
					"FailureReason":       failureReason,
				},
			},
		}
	}

	// Marshal the JSON structure to a string.
	jsonData, err := json.MarshalIndent(granularLogData, "", "  ")
	if err != nil {
		log.Printf("[Warning] Error writing granular log: failed to marshal JSON for granular log: %v", err)
		return
	}

	// Write the JSON data to the file.
	_, err = file.Write(jsonData)
	if err != nil {
		log.Printf("[Warning] Error writing granular log: failed to write to granular log file: %v", err)
	}
}

// getVersionForOS reads version from OS-specific release file (Ubuntu's /etc/os-release)
func getVersionForOS(fs afero.Fs, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening", filePath, ":", err)
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// For Ubuntu /etc/os-release, look for VERSION_ID or VERSION
		if strings.HasPrefix(line, "VERSION_ID=") || strings.HasPrefix(line, "VERSION=") {
			version := strings.SplitN(line, "=", 2)[1]
			version = strings.Trim(version, "\"")
			log.Println("OS VERSION:", version)
			return version, nil
		}
	}

	log.Println("VERSION not found in", filePath)
	return "", nil
}

// Internal wrapper functions for backward compatibility within this package
func writeUpdateStatus(fs afero.Fs, status, metadata, errorDetails string) {
	WriteUpdateStatus(fs, status, metadata, errorDetails)
}

func writeGranularLog(fs afero.Fs, statusDetail string, failureReason string) {
	WriteGranularLog(fs, statusDetail, failureReason)
}
