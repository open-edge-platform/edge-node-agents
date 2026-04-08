// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewHTTPClientWithTLS_ValidCert(t *testing.T) {
	// Create a test HTTPS server
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	// Write server cert to temp file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server-cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}

	// Create HTTP client with TLS
	client, err := NewHTTPClientWithTLS(certPath, 10*time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClientWithTLS failed: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}

	if client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.Timeout)
	}

	// Verify TLS config exists
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}
}

func TestNewHTTPClientWithTLS_EmptyPath(t *testing.T) {
	client, err := NewHTTPClientWithTLS("", 5*time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClientWithTLS with empty path failed: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
}

func TestNewHTTPClientWithTLS_InvalidCert(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "invalid.pem")
	os.WriteFile(certPath, []byte("not a certificate"), 0644)

	_, err := NewHTTPClientWithTLS(certPath, 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for invalid certificate")
	}
}

func TestNewGRPCTransportCredentials_ValidCert(t *testing.T) {
	// Create a test HTTPS server
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	// Write server cert to temp file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server-cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}

	// Create gRPC transport credentials
	creds, err := NewGRPCTransportCredentials(certPath)
	if err != nil {
		t.Fatalf("NewGRPCTransportCredentials failed: %v", err)
	}

	if creds == nil {
		t.Fatal("Expected non-nil credentials")
	}
}

func TestNewGRPCTransportCredentials_EmptyPath(t *testing.T) {
	creds, err := NewGRPCTransportCredentials("")
	if err != nil {
		t.Fatalf("NewGRPCTransportCredentials with empty path failed: %v", err)
	}

	if creds == nil {
		t.Fatal("Expected non-nil credentials")
	}
}

func TestNewGRPCTransportCredentials_InvalidCert(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "invalid.pem")
	os.WriteFile(certPath, []byte("not a certificate"), 0644)

	_, err := NewGRPCTransportCredentials(certPath)
	if err == nil {
		t.Fatal("Expected error for invalid certificate")
	}
}
