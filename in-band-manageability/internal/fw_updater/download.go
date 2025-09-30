/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package fwupdater provides the implementation for updating the firmware.
package fwupdater

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// Downloader is the concrete implementation of the IDownloader interface
// for the EMT OS.
type Downloader struct {
	request              *pb.UpdateFirmwareRequest
	isDiskSpaceAvailable func(string,
		func(afero.Fs, string, func(string) (bool, error)) (string, error),
		func(string, func(string, *unix.Statfs_t) error) (uint64, error),
		func(string, string) (int64, error),
		func(string) (bool, error),
		afero.Fs) (bool, error)
	statfs           func(string, *unix.Statfs_t) error
	httpClient       *http.Client
	requestCreator   func(string, string, io.Reader) (*http.Request, error)
	fs               afero.Fs
	downloadFileFunc func(afero.Fs, string, string, *http.Client,
		func(string, string, io.Reader) (*http.Request, error),
		func(afero.Fs, string, func(string) (bool, error)) (string, error),
		func(string) (bool, error)) error
}

// NewDownloader creates a new Downloader.
func NewDownloader(request *pb.UpdateFirmwareRequest) *Downloader {
	// Create HTTP client with secure TLS configuration
	httpClient, err := utils.CreateSecureHTTPClient(afero.NewOsFs(), request.Url)
	if err != nil {
		// If URL parsing fails, create a default secure client
		log.Printf("Failed to parse URL for secure client creation: %v. Using default secure client.", err)
		httpClient = &http.Client{
			// Default to secure configuration - let the DoSecureHTTPRequest handle fallbacks
		}
	}

	return &Downloader{
		request:              request,
		isDiskSpaceAvailable: utils.IsDiskSpaceAvailable,
		statfs:               unix.Statfs,
		httpClient:           httpClient,
		requestCreator:       http.NewRequest,
		fs:                   afero.NewOsFs(),
		downloadFileFunc:     utils.DownloadFile,
	}
}

// download downloads the firmware update based on the request.
func (t *Downloader) download() error {
	config, err := utils.LoadConfig(t.fs, utils.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Perform source verification
	if !utils.IsTrustedRepository(t.request.Url, config) {
		errMsg := fmt.Sprintf("URL '%s' is not in the list of trusted repositories.", t.request.Url)
		return errors.New(errMsg)
	}

	log.Println("Downloading update from", t.request.Url)

	// Check available space on disk
	isDiskEnough, err := t.isDiskSpaceAvailable(t.request.Url,
		utils.ReadJWTToken,
		utils.GetFreeDiskSpaceInBytes,
		func(url string, token string) (int64, error) {
			return utils.GetFileSizeInBytes(t.fs, url, token)
		},
		utils.IsTokenExpired,
		t.fs)
	if err != nil {
		return fmt.Errorf("error checking disk space: %w", err)
	}

	if !isDiskEnough {
		return fmt.Errorf("insufficient disk space")
	}

	log.Println("Sufficient disk space available. Proceeding to download the artifact.")

	// Download file
	err = t.downloadFileFunc(t.fs, t.request.Url,
		utils.IntelManageabilityCachePathPrefix,
		t.httpClient,
		t.requestCreator,
		utils.ReadJWTToken,
		utils.IsTokenExpired)
	if err != nil {
		return fmt.Errorf("error downloading the file: %w", err)
	}

	log.Println("Download complete.")

	return nil
}
