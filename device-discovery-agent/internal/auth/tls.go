// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc/credentials"
)

// LoadCACertPool loads CA certificate from file or returns system pool.
// If caCertPath is empty, it returns the system certificate pool.
// If caCertPath is provided, it loads the certificate from the file and creates a new pool.
func LoadCACertPool(caCertPath string) (*x509.CertPool, error) {
	if caCertPath == "" {
		// Use system certificate pool
		pool, err := x509.SystemCertPool()
		if err != nil {
			// Fallback to empty pool on systems without system certs
			return x509.NewCertPool(), nil
		}
		return pool, nil
	}

	// Load custom CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caCertPath, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caCertPath)
	}

	return pool, nil
}

// NewHTTPClientWithTLS creates an HTTP client with TLS configuration.
// If caCertPath is empty, system CAs are used. Otherwise, the specified CA is loaded.
// The timeout parameter sets the HTTP client timeout (use 0 for no timeout).
func NewHTTPClientWithTLS(caCertPath string, timeout time.Duration) (*http.Client, error) {
	certPool, err := LoadCACertPool(caCertPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: timeout,
	}

	return client, nil
}

// NewGRPCTransportCredentials creates gRPC transport credentials with TLS.
// If caCertPath is empty, system CAs are used. Otherwise, the specified CA is loaded.
func NewGRPCTransportCredentials(caCertPath string) (credentials.TransportCredentials, error) {
	if caCertPath == "" {
		// Use system default CA certificates
		return credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}), nil
	}

	// Load custom CA certificate
	certPool, err := LoadCACertPool(caCertPath)
	if err != nil {
		return nil, err
	}

	return credentials.NewClientTLSFromCert(certPool, ""), nil
}
