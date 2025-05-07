/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package appsource provides functionality to add an application source.
package appsource

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

// Manager is an interface that defines the methods to add an application source.
type Manager interface {
	Add(sourceListFileName string, sources []string, gpgKeyURI string, gpgKeyName string) error
}

// Adder is a struct that implements the AppSourceManager interface.
type Adder struct {
	httpClient        *http.Client
	requestCreator    func(string, string, io.Reader) (*http.Request, error)
	CommandExecutor   utils.Executor
	openFileFunc      func(afero.Fs, string, int, os.FileMode) (afero.File, error)
	loadConfigFunc    func(afero.Fs, string) (*utils.Configurations, error)
	isTrustedRepoFunc func(string, *utils.Configurations) bool
	addGpgKeyFunc     func(string, string, func(string, string, io.Reader) (*http.Request, error), *http.Client, utils.Executor) error
	fs                afero.Fs
}

// NewAdder creates a new Adder.
func NewAdder() *Adder {
	return &Adder{
		httpClient:        &http.Client{},
		requestCreator:    http.NewRequest,
		CommandExecutor:   utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput),
		openFileFunc:      utils.OpenFile,
		loadConfigFunc:    utils.LoadConfig,
		isTrustedRepoFunc: utils.IsTrustedRepository,
		addGpgKeyFunc:     addGpgKey,
		fs:                afero.NewOsFs(),
	}
}

// Add adds a source file and optional GPG key to be used during Ubuntu application updates.
func (a *Adder) Add(req *pb.AddApplicationSourceRequest) error {
	// Verify GPG key URI is from trusted repo list
	if req.GpgKeyUri != "" && req.GpgKeyName != "" {
		config, err := a.loadConfigFunc(a.fs, utils.ConfigFilePath)
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		// Perform source verification
		if !a.isTrustedRepoFunc(req.GpgKeyUri, config) {
			return fmt.Errorf("GPG key URI verification failed.  URI is not in the list of trusted repositories")
		}

		// Add GPG key
		err = a.addGpgKeyFunc(req.GpgKeyUri, req.GpgKeyName, a.requestCreator, a.httpClient, a.CommandExecutor)
		if err != nil {
			return fmt.Errorf("error adding GPG key: %w", err)
		}
	}

	newSourcePath := filepath.Join(ubuntuAptSourcesListDir, req.Filename)
	sourceFile, err := a.fs.Create(newSourcePath)
	if err != nil {
		log.Printf("[Warning] Error creating application source list file: %v", err)
	}
	defer sourceFile.Close()

	// Add source list file
	file, err := a.openFileFunc(a.fs, filepath.Join(ubuntuAptSourcesListDir, req.Filename),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening source list file: %w", err)
	}
	defer file.Close()

	// Write sources to file
	for _, source := range req.Source {
		if _, err := file.WriteString(source + "\n"); err != nil {
			return fmt.Errorf("error writing to source list file: %w", err)
		}
	}

	return nil
}

func addGpgKey(gpgKeyURI string,
	gpgKeyName string,
	requestCreator func(string, string, io.Reader) (*http.Request, error),
	client *http.Client,
	cmdExec utils.Executor) error {

	// Create a new HTTP request
	req, err := requestCreator("GET", gpgKeyURI, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the status code is 200/Success. If not, return the error.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error getting GPG key.  Status code: %d. Expected 200/Success", resp.StatusCode)
	}

	// Save the GPG key to a temporary file
	tempFile, err := os.CreateTemp("", "gpgkey-*.asc")
	if err != nil {
		return fmt.Errorf("error creating temporary file for GPG key: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the temporary file
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving GPG key to temporary file: %w", err)
	}

	//  Use GPG to dearmor the key and save it to the keyrings directory
	gpgKeyPath := filepath.Join(linuxGPGKeyPath, gpgKeyName)

	dearmorGpgKeyCommand := []string{
		"/usr/bin/gpg", "--dearmor", "--output", gpgKeyPath, tempFile.Name(),
	}
	_, _, err = cmdExec.Execute(dearmorGpgKeyCommand)
	if err != nil {
		return fmt.Errorf("error dearmoring GPG key: %w", err)
	}

	fmt.Printf("GPG key added to %s\n", gpgKeyPath)
	return nil
}
