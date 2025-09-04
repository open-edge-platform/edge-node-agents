/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"log"
	"os"
	"path/filepath"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	"github.com/spf13/afero"
)

// CleanerInterface defines the interface for the Cleaner
type CleanerInterface interface {
	DeleteAll(path string) error
}

// Cleaner is the concrete implementation of the CleanerInterface
type Cleaner struct {
	commandExecutor utils.Executor
	path            string
	fs              afero.Fs
}

// NewCleaner creates a new Cleaner instance
func NewCleaner(commandExecutor utils.Executor, path string) *Cleaner {
	return &Cleaner{
		commandExecutor: commandExecutor,
		path:            path,
		fs:              afero.NewOsFs(),
	}
}

// Clean removes all files in the specified path
func (c *Cleaner) Clean() error {
	log.Println("Removes file after update")
	// Walk through the directory and remove all files
	err := filepath.Walk(c.path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		// Remove the file
		err = utils.RemoveFile(c.fs, p)
		if err != nil {
			log.Printf("Failed to delete file: %s, error: %v\n", p, err)
		} else {
			log.Printf("Deleted file: %s\n", p)
		}
		return err
	})
	if err != nil {
		log.Printf("Failed to delete files in path %s: %v\n", c.path, err)
		return err
	}
	return nil
}
