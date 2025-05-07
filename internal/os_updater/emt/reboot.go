/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// Rebooter is the concrete implementation of the IUpdater interface
// for the EMT OS.
type Rebooter struct {
	commandExecutor   utils.Executor
	request           *pb.UpdateSystemSoftwareRequest
	writeUpdateStatus func(afero.Fs, string, string, string)
	writeGranularLog  func(string, string)
	fs                afero.Fs
}

// NewRebooter creates a new EMTRebooter.
func NewRebooter(commandExecutor utils.Executor, request *pb.UpdateSystemSoftwareRequest) *Rebooter {
	return &Rebooter{
		commandExecutor:   commandExecutor,
		request:           request,
		writeUpdateStatus: writeUpdateStatus,
		writeGranularLog:  writeGranularLog,
		fs:                afero.NewOsFs(),
	}
}

// Reboot method for EMT
func (t *Rebooter) Reboot() error {
	log.Println("Rebooting the system...")
	// Get the request details
	jsonString, err := protojson.Marshal(t.request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	rebootCommand := []string{
		"/usr/sbin/reboot",
	}

	if _, _, err := t.commandExecutor.Execute(rebootCommand); err != nil {
		t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
		t.writeGranularLog(FAIL, FAILURE_REASON_UNSPECIFIED)
		return fmt.Errorf("failed to execute shell command(%v)- %v", rebootCommand, err)
	}
	return nil
}
