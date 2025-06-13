/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package appsource provides functionality to add an application source.
package appsource

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// Remover is a struct that implements the AppSourceManager interface.
type Remover struct {
	removeFileFunc        func(afero.Fs, string) error
	fs                    afero.Fs
	isExistGpgKeyFileFunc func(afero.Fs, string) bool
	isExistSourceFileFunc func(afero.Fs, string) bool
	removeGpgKeyFunc      func(afero.Fs, string) error
	removeSourceFileFunc  func(afero.Fs, string) error
}

// NewRemover creates a new Remover.
func NewRemover() *Remover {
	return &Remover{
		removeFileFunc:        utils.RemoveFile,
		fs:                    afero.NewOsFs(),
		isExistGpgKeyFileFunc: utils.IsFileExist,
		isExistSourceFileFunc: utils.IsFileExist,
		removeGpgKeyFunc: func(fs afero.Fs, fileName string) error {
			return utils.RemoveFile(fs, fileName)
		},
		removeSourceFileFunc: func(fs afero.Fs, fileName string) error {
			return utils.RemoveFile(fs, fileName)
		},
	}
}

// Remove removes a source file and optional GPG key used during Ubuntu application updates.
func (a *Remover) Remove(req *pb.RemoveApplicationSourceRequest) error {
	if req.GpgKeyName != "" {
		gpgKeyPath := filepath.Join(linuxGPGKeyPath, req.GpgKeyName)
		isExist := a.isExistGpgKeyFileFunc(a.fs, gpgKeyPath)
		if isExist {
			err := a.removeGpgKeyFunc(a.fs, gpgKeyPath)
			if err != nil {
				return fmt.Errorf("error removing GPG key: %v", err)
			}
			log.Printf("GPG key removed: %s", gpgKeyPath)
		} else {
			log.Printf("[WARNING] GPG key does not exist: %s", gpgKeyPath)
		}
	}

	// Remove the source file
	sourceFilePath := filepath.Join(ubuntuAptSourcesListDir, req.Filename)
	isExist := a.isExistSourceFileFunc(a.fs, sourceFilePath)
	if isExist {
		if err := a.removeSourceFileFunc(a.fs, sourceFilePath); err != nil {
			return fmt.Errorf("error removing application source file: %v", err)
		}
		log.Printf("Source file removed: %s", sourceFilePath)
	} else {
		return fmt.Errorf("source file does not exist: %s", sourceFilePath)
	}

	return nil
}
