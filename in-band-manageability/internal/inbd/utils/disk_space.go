/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// Compiled regex pattern for TLS certificate error detection (case-insensitive)
var tlsCertErrorPattern = regexp.MustCompile(`(?i)(tls|certificate|handshake|ssl|x509|crypto|verify|signature|algorithm)`)

// GetFreeDiskSpaceInBytes returns the amount of free disk space in bytes for the given path.
// It uses the unix.Statfs function to retrieve filesystem statistics.
func GetFreeDiskSpaceInBytes(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
	var stat unix.Statfs_t

	// Get filesystem statistics for the given path
	err := statfsFunc(path, &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// Calculate free space in bytes
	freeSpace := stat.Bavail * uint64(stat.Bsize)
	return freeSpace, nil
}

// getSafeTokenPrefix returns the first few characters of a token for safe logging
func getSafeTokenPrefix(token string) string {
	if len(token) < 10 {
		return "***"
	}
	return token[:10]
}

// GetFileSizeInBytes retrieves the size of a file at the given URL.
func GetFileSizeInBytes(fs afero.Fs, url string) (int64, error) {
	// Try HEAD request first
	size, err := getFileSizeWithHEAD(fs, url)
	if err != nil {
		// If HEAD request fails, try GET with Range header as fallback
		size, err := getFileSizeWithRange(fs, url)
		if err != nil {
			return 0, err
		}
		return size, nil
	}
	return size, nil
}

// getFileSizeWithHEAD attempts to get file size using HEAD request
func getFileSizeWithHEAD(fs afero.Fs, url string) (int64, error) {
	// Create a new HTTP HEAD request
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating HEAD request: %w", err)
	}

	// Perform the HEAD request using secure TLS client
	client, err := CreateSecureHTTPClient(fs, url, "")
	if err != nil {
		return 0, fmt.Errorf("error creating secure HTTP client: %w", err)
	}

	resp, err := DoSecureHTTPRequest(client, req, url)
	if err != nil {
		return 0, fmt.Errorf("error performing HEAD request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the status code is 200/Success
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HEAD request failed with status code: %d", resp.StatusCode)
	}

	// Get the Content-Length header
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return 0, fmt.Errorf("Content-Length header is missing in HEAD response")
	}

	// Parse the Content-Length to an integer
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing Content-Length: %w", err)
	}

	return size, nil
}

// getFileSizeWithRange attempts to get file size using GET request with Range header
func getFileSizeWithRange(fs afero.Fs, url string) (int64, error) {
	// Create a new HTTP GET request with Range header to get only first byte
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating Range GET request: %w", err)
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error performing Range GET request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the status code is 206 (Partial Content) or 200 (OK)
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("range GET request failed with status code: %d", resp.StatusCode)
	}

	// Try to get the total file size from Content-Range header first
	contentRange := resp.Header.Get("Content-Range")
	if contentRange != "" {
		// Content-Range format: "bytes 0-0/12345" where 12345 is the total size
		parts := strings.Split(contentRange, "/")
		if len(parts) == 2 {
			size, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				return size, nil
			}
		}
	}

	// Fall back to Content-Length header
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return 0, fmt.Errorf("neither Content-Range nor Content-Length header available in Range GET response")
	}

	// Parse the Content-Length to an integer
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing Content-Length: %w", err)
	}

	return size, nil
}

// IsDiskSpaceAvailable checks if there is enough disk space to download the artifacts.
func IsDiskSpaceAvailable(url string,
	readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
	getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error),
	getFileSizeInBytesFunc func(string, string) (int64, error),
	isTokenExpiredFunc func(string) (bool, error),
	fs afero.Fs) (bool, error) {

	availableSpace, err := getFreeDiskSpaceInBytes("/var/cache/manageability/repository-tool/sota", unix.Statfs)
	if err != nil {
		log.Printf("Error getting disk space: %v\n", err)
		return false, err
	}

	// Read JWT token
	token, err := readJWTTokenFunc(fs, JWTTokenPath, isTokenExpiredFunc)
	if err != nil {
		return false, fmt.Errorf("error reading JWT token: %w", err)
	}
	log.Println("JWT token read successfully.")

	requiredSpace, err := getFileSizeInBytesFunc(url, token)
	if err != nil {
		return false, fmt.Errorf("error getting file size: %w", err)
	}

	// Calculate required space with buffer
	// 20% buffer for safety margin plus minimum 100MB buffer
	const bufferMultiplier = 1.2
	const minBufferBytes = 100 * 1024 * 1024 // 100MB

	requiredSpaceWithBuffer := uint64(float64(requiredSpace) * bufferMultiplier)

	// Ensure minimum buffer is applied
	if requiredSpaceWithBuffer < uint64(requiredSpace)+minBufferBytes {
		requiredSpaceWithBuffer = uint64(requiredSpace) + minBufferBytes
	}

	// Check if there is enough space including buffer
	if availableSpace < requiredSpaceWithBuffer {
		log.Printf("Insufficient disk space. Available: %d bytes, Required: %d bytes (including buffer)\n", availableSpace, requiredSpaceWithBuffer)
		return false, nil
	}

	log.Printf("Sufficient disk space. Available: %d bytes, Required: %d bytes (including buffer)\n", availableSpace, requiredSpaceWithBuffer)
	return true, nil
}

// CreateSecureHTTPClient creates an HTTP client with appropriate TLS configuration
// based on whether the URL uses an IP address or hostname.
// When token is empty (anonymous access), uses InsecureSkipVerify for TLS.
func CreateSecureHTTPClient(fs afero.Fs, url string, token string) (*http.Client, error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	// Check if the hostname is an IP address
	hostname := parsedURL.Hostname()
	isIP := net.ParseIP(hostname) != nil

	// For anonymous access (no token), use insecure TLS to skip certificate verification
	insecureTLS := (token == "")

	// Use strict TLS configuration with optional custom CA support
	tlsConfig := &tls.Config{
		ServerName:         hostname,
		MinVersion:         tls.VersionTLS12, // Enforce minimum TLS 1.2
		MaxVersion:         tls.VersionTLS13, // Prefer TLS 1.3 when available
		InsecureSkipVerify: insecureTLS,      // Skip verification for anonymous access
	}

	// Check for custom CA certificate file for development scenarios
	customCAFile := os.Getenv("INBM_CUSTOM_CA_FILE")
	log.Printf("Custom CA environment variable INBM_CUSTOM_CA_FILE: '%s'", customCAFile)

	if customCAFile != "" {
		log.Printf("Loading custom CA certificate from: %s", customCAFile)

		// Load system CA pool
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			log.Printf("Failed to load system CA pool, creating new one: %v", err)
			caCertPool = x509.NewCertPool()
		}

		// Read the custom CA certificate
		caCert, err := ReadFile(fs, customCAFile)
		if err != nil {
			log.Printf("Failed to read custom CA file %s: %v", customCAFile, err)
		} else {
			// Add the custom CA to the pool
			if caCertPool.AppendCertsFromPEM(caCert) {
				log.Printf("Successfully added custom CA certificate")
				tlsConfig.RootCAs = caCertPool
			} else {
				log.Printf("Failed to parse custom CA certificate from %s", customCAFile)
			}
		}
	} else {
		log.Printf("No custom CA certificate specified via INBM_CUSTOM_CA_FILE environment variable")
	}

	if insecureTLS {
		log.Printf("Creating HTTP client for %s - INSECURE MODE (anonymous access, skipping certificate verification)", hostname)
	} else if isIP {
		log.Printf("Creating HTTP client for IP address %s - full certificate verification", hostname)
		// For IP addresses, we might need to be more explicit
		tlsConfig.ServerName = hostname
	} else {
		log.Printf("Creating HTTP client for hostname %s - full certificate verification", hostname)
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // 30 second timeout for HEAD/size check requests
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client, nil
}

// DoSecureHTTPRequest performs an HTTP request with fallback TLS verification.
// It tries secure verification first, and falls back to insecure verification if needed.
func DoSecureHTTPRequest(client *http.Client, req *http.Request, url string) (*http.Response, error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	isIP := net.ParseIP(hostname) != nil

	// First attempt with the provided client
	resp, err := client.Do(req)

	// If TLS certificate verification fails, log the error but don't retry with insecure settings
	if err != nil && !isIP && isTLSCertificateError(err) {
		log.Printf("Certificate verification failed for hostname %s: %v", hostname, err)
		log.Printf("TLS certificate verification is required - will not retry with insecure settings")
		log.Printf("Please ensure the certificate is valid for the hostname or use a trusted certificate authority")
	}

	return resp, err
}

// isTLSCertificateError checks if the error is specifically a TLS certificate verification error
func isTLSCertificateError(err error) bool {
	return tlsCertErrorPattern.MatchString(err.Error())
}

// DiagnoseJWTToken provides diagnostic information about a JWT token without exposing sensitive data
func DiagnoseJWTToken(fs afero.Fs, path string) {
	log.Printf("=== JWT Token Diagnostics ===")
	log.Printf("Token path: %s", path)

	// Check if file exists
	exists, err := afero.Exists(fs, path)
	if err != nil {
		log.Printf("Error checking if token file exists: %v", err)
		return
	}
	if !exists {
		log.Printf("Token file does not exist")
		return
	}

	// Get file info
	info, err := fs.Stat(path)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return
	}
	log.Printf("Token file size: %d bytes", info.Size())
	log.Printf("Token file modified: %v", info.ModTime())

	// Read token content
	token, err := afero.ReadFile(fs, path)
	if err != nil {
		log.Printf("Error reading token file: %v", err)
		return
	}

	if len(token) == 0 {
		log.Printf("Token file is empty")
		return
	}

	tokenStr := strings.TrimSpace(string(token))
	log.Printf("Token length: %d characters", len(tokenStr))
	log.Printf("Token prefix: %s...", getSafeTokenPrefix(tokenStr))

	// Check if it looks like a JWT (has two dots)
	parts := strings.Split(tokenStr, ".")
	log.Printf("Token parts count: %d (JWT should have 3)", len(parts))

	// Try to parse and check expiration
	expired, err := IsTokenExpired(tokenStr)
	if err != nil {
		log.Printf("Error checking token expiration: %v", err)
	} else {
		log.Printf("Token expired: %v", expired)
	}

	log.Printf("=== End JWT Token Diagnostics ===")
}
