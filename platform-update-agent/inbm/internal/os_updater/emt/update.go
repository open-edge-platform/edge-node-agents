/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

var (
	// OsUpdateTool will be changed in 3.1 release. Have to change the name and API call.
	// Check https://github.com/intel-sandbox/os.linux.tiberos.ab-update.go/blob/main/README.md
	osUpdateToolPath = "/usr/bin/os-update-tool.sh"
)

// Updater is the concrete implementation of the IUpdater interface
// for the EMT OS.
type Updater struct {
	commandExecutor   utils.Executor
	request           *pb.UpdateSystemSoftwareRequest
	writeUpdateStatus func(afero.Fs, string, string, string)
	writeGranularLog  func(afero.Fs, string, string)
	fs                afero.Fs
}

// NewUpdater creates a new Updater.
func NewUpdater(commandExecutor utils.Executor, request *pb.UpdateSystemSoftwareRequest) *Updater {
	return &Updater{
		commandExecutor:   commandExecutor,
		request:           request,
		writeUpdateStatus: writeUpdateStatus,
		writeGranularLog:  writeGranularLog,
		fs:                afero.NewOsFs(),
	}
}

// errReader is a helper type to simulate an error during reading
type errReader struct{}

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("error copying response body")
}

// Update method for EMT
func (t *Updater) Update() (bool, error) {
	// Print the value of tu.request.Mode
	log.Printf("Mode: %v\n", t.request.Mode)

	// Get the request details
	jsonString, err := protojson.Marshal(t.request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	if t.request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY {

		err := t.VerifyHash()
		if err != nil {
			t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
			t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_SIGNATURE_CHECK)
			return false, fmt.Errorf("hash verification failed: %w", err)
		}

		log.Println("Execute update tool write command.")

		// Extract the file name from the URL
		urlParts := strings.Split(t.request.Url, "/")
		fileName := urlParts[len(urlParts)-1]

		// Create the file
		filePath := utils.SOTADownloadDir + "/" + fileName

		updateToolWriteCommand := []string{
			osUpdateToolPath, "-w", "-u", filePath, "-s", t.request.Signature,
		}

		if _, _, err := t.commandExecutor.Execute(updateToolWriteCommand); err != nil {
			t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
			t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_UT_WRITE)
			return false, fmt.Errorf("failed to execute shell command(%v)- %v", updateToolWriteCommand, err)
		}

		jsonString, err := protojson.Marshal(t.request)
		if err != nil {
			log.Printf("Error converting request to string: %v\n", err)
			jsonString = []byte("{}")
		}
		// Write the update status to the status log file
		t.writeUpdateStatus(t.fs, SUCCESS, string(jsonString), "")
	}

	if t.request.Mode == pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD {
		log.Println("Save snapshot before applying the update.")
		if err := NewSnapshotter(t.commandExecutor, t.request).Snapshot(); err != nil {
			errMsg := fmt.Sprintf("Error taking snapshot: %v", err)
			t.writeUpdateStatus(t.fs, FAIL, string(jsonString), errMsg)
			t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_INBM)
			return false, fmt.Errorf("failed to take snapshot before applying the update: %v", err)
		}

		log.Println("Execute update tool apply command.")
		updateToolApplyCommand := []string{
			osUpdateToolPath, "-a",
		}

		if _, _, err := t.commandExecutor.Execute(updateToolApplyCommand); err != nil {
			t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
			t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_BOOT_CONFIGURATION)
			return false, fmt.Errorf("failed to execute shell command(%v)- %v", updateToolApplyCommand, err)
		}

		// Write the update status to the status log file
		writeUpdateStatus(t.fs, SUCCESS, string(jsonString), "")
		writeGranularLog(t.fs, SUCCESS, "")
	}

	return true, nil
}

func (t *Updater) commitUpdate() error {
	log.Println("Committing the update.")
	// Get the request details
	jsonString, err := protojson.Marshal(t.request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	updateToolCommitCommand := []string{
		osUpdateToolPath, "-c",
	}

	if _, _, err := t.commandExecutor.Execute(updateToolCommitCommand); err != nil {
		log.Printf("Error executing shell command(%v): %v\n", updateToolCommitCommand, err)
		t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_OS_COMMIT)
		return fmt.Errorf("failed to execute shell command(%v)- %v", updateToolCommitCommand, err)
	}
	return nil
}
