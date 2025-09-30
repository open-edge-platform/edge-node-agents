/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// Downloader is the concrete implementation of the IDownloader interface
// for the EMT OS.
type Downloader struct {
	request                 *pb.UpdateSystemSoftwareRequest
	readJWTTokenFunc        func(afero.Fs, string, func(string) (bool, error)) (string, error)
	isTokenExpiredFunc      func(string) (bool, error)
	writeUpdateStatus       func(afero.Fs, string, string, string)
	writeGranularLog        func(afero.Fs, string, string)
	statfs                  func(string, *unix.Statfs_t) error
	httpClient              *http.Client
	requestCreator          func(string, string, io.Reader) (*http.Request, error)
	fs                      afero.Fs
	getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error)
	getFileSizeInBytesFunc  func(afero.Fs, string, string) (int64, error)
}

// NewDownloader creates a new Downloader.
func NewDownloader(request *pb.UpdateSystemSoftwareRequest) *Downloader {
	return &Downloader{
		request:                 request,
		readJWTTokenFunc:        utils.ReadJWTToken,
		isTokenExpiredFunc:      utils.IsTokenExpired,
		writeUpdateStatus:       writeUpdateStatus,
		writeGranularLog:        writeGranularLog,
		statfs:                  unix.Statfs,
		httpClient:              &http.Client{},
		requestCreator:          http.NewRequest,
		fs:                      afero.NewOsFs(),
		getFreeDiskSpaceInBytes: utils.GetFreeDiskSpaceInBytes,
		getFileSizeInBytesFunc:  utils.GetFileSizeInBytes,
	}
}

// Download implements IDownloader.
func (t *Downloader) Download() error {
	config, err := utils.LoadConfig(t.fs, utils.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Get the request details
	jsonString, err := protojson.Marshal(t.request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	// Perform source verification
	if !utils.IsTrustedRepository(t.request.Url, config) {
		errMsg := fmt.Sprintf("URL '%s' is not in the list of trusted repositories.", t.request.Url)
		t.writeUpdateStatus(t.fs, FAIL, string(jsonString), errMsg)
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_RS_AUTHENTICATION)
		return errors.New(errMsg)
	}

	log.Println("Downloading update from", t.request.Url)

	// Check available space on disk
	isDiskEnough, err := t.isDiskSpaceAvailable()
	if err != nil {
		return fmt.Errorf("error checking disk space: %w", err)
	}

	if !isDiskEnough {
		t.writeUpdateStatus(t.fs, FAIL, string(jsonString), "Insufficient disk space")
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_INSUFFICIENT_STORAGE)
		return fmt.Errorf("insufficient disk space")
	}

	log.Println("Sufficient disk space available. Proceeding to download the artifact.")

	// Download file
	err = t.downloadFile()
	if err != nil {
		t.writeUpdateStatus(t.fs, FAIL, string(jsonString), err.Error())
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_DOWNLOAD)
		return fmt.Errorf("error downloading the file: %w", err)
	}

	log.Println("Download complete.")

	return nil
}

// VerifyHash verifies the hash of the downloaded file.
func (t *Updater) VerifyHash() error {
	log.Println("Verify file SHA.")

	// Extract the file name from the URL
	urlParts := strings.Split(t.request.Url, "/")
	fileName := urlParts[len(urlParts)-1]
	filePath := utils.SOTADownloadDir + "/" + fileName

	file, err := utils.Open(t.fs, filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create a new SHA256 hash
	hash := sha256.New()
	// Copy the file content into the hash
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Compute the hash
	computedHash := hash.Sum(nil)

	// Convert the computed hash to a hex string
	computedChecksum := hex.EncodeToString(computedHash)

	if computedChecksum != t.request.Signature {
		return fmt.Errorf("checksum mismatch: Expected: %s, got: %s", t.request.Signature, computedChecksum)
	}
	log.Println("SHA verification complete.")

	return nil
}

// downloadFile downloads the file from the url.
func (t *Downloader) downloadFile() error {
	// Read JWT token
	token, err := t.readJWTTokenFunc(t.fs, utils.JWTTokenPath, t.isTokenExpiredFunc)
	if err != nil {
		return fmt.Errorf("error reading JWT token: %w", err)
	}

	if token == "" {
		log.Println("JWT token is empty. Proceeding without Authorization.")
	}

	// Try different authentication methods
	authMethods := t.getAuthMethods(token)

	var lastErr error
	for _, method := range authMethods {
		if err := t.tryAuthMethod(method); err != nil {
			lastErr = err
			continue
		}
		// Success - file was downloaded
		return nil
	}

	// If we reach here, all methods failed
	if lastErr == nil {
		lastErr = fmt.Errorf("all authentication methods failed")
	}
	return lastErr
}

// authMethod represents an authentication method configuration
type authMethod struct {
	name      string
	setupAuth func(*http.Request)
}

// getAuthMethods returns a slice of authentication methods to try
func (t *Downloader) getAuthMethods(token string) []authMethod {
	return []authMethod{
		{
			name:      "Bearer Token",
			setupAuth: t.setupBearerTokenAuth(token),
		},
		{
			name:      "No Authentication",
			setupAuth: t.setupNoAuth(),
		},
	}
}

// setupBearerTokenAuth returns a function that sets up Bearer token authentication
func (t *Downloader) setupBearerTokenAuth(token string) func(*http.Request) {
	return func(req *http.Request) {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

// setupNoAuth returns a function that sets up no authentication
func (t *Downloader) setupNoAuth() func(*http.Request) {
	return func(req *http.Request) {
		// No authentication headers added
	}
}

// tryAuthMethod attempts to download the file using a specific authentication method
func (t *Downloader) tryAuthMethod(method authMethod) error {
	// Create a new HTTP request
	req, err := t.requestCreator("GET", t.request.Url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Setup authentication for this method
	method.setupAuth(req)

	// Perform the request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request with %s: %w", method.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return t.downloadFileFromResponse(resp, method.name)
	}

	return t.handleAuthError(resp, method.name)
}

// downloadFileFromResponse handles the actual file download from a successful HTTP response
func (t *Downloader) downloadFileFromResponse(resp *http.Response, methodName string) error {
	// Extract the file name from the URL
	urlParts := strings.Split(t.request.Url, "/")
	fileName := urlParts[len(urlParts)-1]

	// Create the file
	file, err := t.fs.Create(utils.SOTADownloadDir + "/" + fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Copy the response body to the file
	bytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}

	log.Printf("Successfully downloaded %d bytes using %s", bytesWritten, methodName)
	return nil
}

// handleAuthError processes authentication-related HTTP errors
func (t *Downloader) handleAuthError(resp *http.Response, methodName string) error {
	// Read response body for error details
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed with %s: status %d", methodName, resp.StatusCode)
	}

	// For other errors, fail immediately as they're likely not auth-related
	return fmt.Errorf("HTTP error with %s: status %d, response: %s", methodName, resp.StatusCode, string(bodyBytes)[:min(len(bodyBytes), 200)])
}

// isDiskSpaceAvailable checks if there is enough disk space to download the artifacts.
func (t *Downloader) isDiskSpaceAvailable() (bool, error) {
	availableSpace, err := t.getFreeDiskSpaceInBytes("/var/cache/manageability/repository-tool/sota", unix.Statfs)
	if err != nil {
		log.Printf("Error getting disk space: %v\n", err)
		return false, err
	}

	// Get the request details
	jsonString, err := protojson.Marshal(t.request)
	if err != nil {
		log.Printf("Error converting request to string: %v\n", err)
		jsonString = []byte("{}")
	}

	// Get required space for the file
	requiredSpace, err := t.getRequiredFileSize(string(jsonString))
	if err != nil {
		return false, err
	}

	// Check if there is enough space
	if availableSpace < uint64(requiredSpace) {
		log.Printf("Insufficient disk space. Available: %d bytes, Required: %d bytes\n", availableSpace, requiredSpace)
		return false, nil
	}

	return true, nil
}

// getRequiredFileSize gets the file size needed for the download with proper error handling
func (t *Downloader) getRequiredFileSize(jsonString string) (int64, error) {
	// Read JWT token
	token, err := t.readJWTTokenFunc(t.fs, utils.JWTTokenPath, t.isTokenExpiredFunc)
	if err != nil {
		t.writeUpdateStatus(t.fs, FAIL, jsonString, err.Error())
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_INBM)
		return 0, fmt.Errorf("error reading JWT token: %w", err)
	}

	requiredSpace, err := t.getFileSizeInBytesFunc(t.fs, t.request.Url, token)
	if err != nil {
		// If we get a 401 error, run diagnostics to help debug the issue
		if strings.Contains(err.Error(), "401") {
			log.Printf("Authentication failed (401), running JWT token diagnostics")
			utils.DiagnoseJWTToken(t.fs, utils.JWTTokenPath)
		}
		t.writeUpdateStatus(t.fs, FAIL, jsonString, err.Error())
		t.writeGranularLog(t.fs, FAIL, FAILURE_REASON_DOWNLOAD)
		return 0, fmt.Errorf("error getting file size: %w", err)
	}

	return requiredSpace, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
