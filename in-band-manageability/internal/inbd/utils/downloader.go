/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/spf13/afero"
)

// DownloadFile downloads the file from the URL.
func DownloadFile(fs afero.Fs, urlStr string, destinationDir string, httpClient *http.Client,
	requestCreator func(string, string, io.Reader) (*http.Request, error),
	readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
	isTokenExpiredFunc func(string) (bool, error)) error {

	// Create a new HTTP request
	req, err := requestCreator("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Add the JWT token to the request header
	token, err := readJWTTokenFunc(fs, JWTTokenPath, isTokenExpiredFunc)
	if err != nil {
		return fmt.Errorf("error reading JWT token: %w", err)
	}

	// Check if the token exists
	if token == "" {
		log.Println("JWT token is empty. Proceeding without Authorization.")
	} else {
		// Add the JWT token to the request header
		req.Header.Add("Authorization", "Bearer "+token)
	}

	// Perform the request with secure TLS handling
	resp, err := DoSecureHTTPRequest(httpClient, req, urlStr)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the status code is 200/Success. If not, return the error.
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Status code: %d. Expected 200/Success.", resp.StatusCode)
		return errors.New(errMsg)
	}

	// Extract the file name from the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	fileName := path.Base(parsedURL.Path)
	if fileName == "" || fileName == "." || fileName == "/" {
		return fmt.Errorf("could not extract file name from URL: %s", urlStr)
	}

	// Create the file
	file, err := fs.Create(destinationDir + "/" + fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}

	return nil
}
