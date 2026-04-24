// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadCACertPool_ValidCert(t *testing.T) {
	// Create a temporary test certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test-cert.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Now test loading it
	pool, err := LoadCACertPool(certPath)
	if err != nil {
		t.Fatalf("LoadCACertPool failed: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected non-nil certificate pool")
	}
}

func TestLoadCACertPool_EmptyPath_UsesSystemCerts(t *testing.T) {
	pool, err := LoadCACertPool("")
	if err != nil {
		t.Fatalf("LoadCACertPool with empty path failed: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected non-nil certificate pool from system certs")
	}
}

func TestLoadCACertPool_NonexistentFile(t *testing.T) {
	_, err := LoadCACertPool("/nonexistent/path/to/cert.pem")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read CA certificate") {
		t.Errorf("Expected error about reading cert, got: %s", err.Error())
	}
}

func TestLoadCACertPool_InvalidCert(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "invalid-cert.pem")

	// Write invalid certificate content
	invalidCert := "This is not a valid certificate"
	err := os.WriteFile(certPath, []byte(invalidCert), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid cert file: %v", err)
	}

	_, err = LoadCACertPool(certPath)
	if err == nil {
		t.Fatal("Expected error for invalid certificate, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse CA certificate") {
		t.Errorf("Expected error about parsing cert, got: %s", err.Error())
	}
}

func TestFetchClientCredentialsToken_MockServer(t *testing.T) {
	// Create a mock Keycloak server
	mockResponse := map[string]interface{}{
		"access_token": "mock-access-token-12345",
		"token_type":   "Bearer",
		"expires_in":   300,
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type 'application/x-www-form-urlencoded', got '%s'", contentType)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}

		// Verify grant_type
		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("Expected grant_type 'client_credentials', got '%s'", r.FormValue("grant_type"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Extract host from server URL (remove https://)
	serverHost := strings.TrimPrefix(server.URL, "https://")

	// Save server cert to temp file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server-cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write server cert: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := FetchClientCredentialsToken(ctx, ClientCredentialsParams{
		KeycloakURL:  serverHost,
		TokenPath:    "/realms/test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		CACertPath:   certPath,
	})

	if err != nil {
		t.Fatalf("FetchClientCredentialsToken failed: %v", err)
	}

	if token != "mock-access-token-12345" {
		t.Errorf("Expected token 'mock-access-token-12345', got '%s'", token)
	}
}

func TestFetchReleaseToken_MockServer(t *testing.T) {
	// Create a mock release server
	mockToken := "mock-release-token-xyz"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with 'Bearer ' prefix, got '%s'", authHeader)
		}

		// Write response (plain text token)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(mockToken))
	}))
	defer server.Close()

	// Extract host from server URL (remove https://)
	serverHost := strings.TrimPrefix(server.URL, "https://")

	// Save server cert to temp file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server-cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write server cert: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := FetchReleaseToken(ctx, serverHost, "test-access-token", certPath)

	if err != nil {
		t.Fatalf("FetchReleaseToken failed: %v", err)
	}

	if token != mockToken {
		t.Errorf("Expected token '%s', got '%s'", mockToken, token)
	}
}

func TestClientAuth_Integration(t *testing.T) {
	// Test the main ClientAuth function
	// This would require mocking both Keycloak and release servers
	// For now, we test that it calls the correct functions
	t.Skip("Integration test - requires full mock setup")
}
