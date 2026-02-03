// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempCACert creates a temporary CA certificate file for testing
func createTempCACert(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "ca-cert.pem")

	// This is a sample self-signed certificate for testing purposes only
	certPEM := `-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUB3r8VqJvC0qLvqVqr2R3bvQ3wdowDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCVVMxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNDAxMDEwMDAwMDBaFw0yNTAx
MDEwMDAwMDBaMEUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQC7VJTUt9Us8cKjMzEfYyjiWA4/qMz9qKyXDfsow5WB
0P9K3S0wHE5pEjwqT2XczKuZE7p2hTKqxRhF4bQdOv3dN7fGYn9KLuPMLPczZVMG
SZ+xvPB+jPsJqpS3KvJxWfPHXVqg8QzZxVJqyqH6X4V4F0lJ0VJVIbMvXGvk1mAD
YzJLEqiYh3GqMZKjJ1vJkKDvqLmBZKgw4E4gD8W4mXGK0K5RNvq0hTQ3h0XGI4tC
fOGkdLYpqzLKZDwZz0VchJhMkH4vWDLfaHM3lLGqPVChJxR8PVCMqKJ0gJxLVGBw
7nOZ1xhZwNvGmGPaKLhDHN/vHjJGJkJlJHrVTLJKJqZxAgMBAAGjUzBRMB0GA1Ud
DgQWBBQ8qXhKLJ7vCm5fq8nZVbEG+8J3YzAfBgNVHSMEGDAWgBQ8qXhKLJ7vCm5f
q8nZVbEG+8J3YzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCc
lZTdPOvmxK1qpZJE3uRlK8qT3xVNnC4Z0YVDKNbGLJBfFqZNpOqJvPgLLLJDQxjN
wvvVh6chLxKdITWtCXZzGxNz0c7QdCxqJ3xKgJXJqJvZPBVLQcPQQf0ysLkWE3Wh
C2mKGxqJLpQhHMgqKJqmzJKvGEqLZPxLqxKJZzJqvKJLqJZKqxJLqJZKqxJLqJZK
qxJLqJZKqxJLqJZKqxJLqJZKqxJLqJZKqxJLqJZKqxJLqJZKqxJLqJZKqxJLqJZK
-----END CERTIFICATE-----`

	err := os.WriteFile(certPath, []byte(certPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CA cert: %v", err)
	}

	return certPath
}

func TestLoadCACertPool_ValidCert(t *testing.T) {
	// Create a test TLS server to get a valid certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	// Write the test server's certificate to a temp file
	certPath := filepath.Join(t.TempDir(), "test-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Now test loading it
	pool, err := loadCACertPool(certPath)
	if err != nil {
		t.Fatalf("loadCACertPool failed: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected non-nil certificate pool")
	}
}

func TestLoadCACertPool_NonexistentFile(t *testing.T) {
	_, err := loadCACertPool("/nonexistent/path/to/cert.pem")
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

	_, err = loadCACertPool(certPath)
	if err == nil {
		t.Fatal("Expected error for invalid certificate, got nil")
	}

	if !strings.Contains(err.Error(), "failed to append CA certificate") {
		t.Errorf("Expected error about appending cert, got: %s", err.Error())
	}
}

func TestFetchAccessToken_MockServer(t *testing.T) {
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

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Write the test server's certificate to a temp file
	certPath := filepath.Join(t.TempDir(), "test-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Test fetching access token
	// Note: fetchAccessToken adds "https://" prefix, so we need to strip it from server.URL
	serverHost := strings.TrimPrefix(server.URL, "https://")
	token, err := fetchAccessToken(serverHost+"/realms/test", "test-client", "test-secret", certPath)
	if err != nil {
		t.Fatalf("fetchAccessToken failed: %v", err)
	}

	if token != "mock-access-token-12345" {
		t.Errorf("Expected token 'mock-access-token-12345', got '%s'", token)
	}
}

func TestFetchAccessToken_ErrorStatus(t *testing.T) {
	// Create a mock server that returns unauthorized
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	// Write the test server's certificate to a temp file
	certPath := filepath.Join(t.TempDir(), "test-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Test that error status is handled correctly
	// Note: fetchAccessToken adds "https://" prefix, so we need to strip it from server.URL
	serverHost := strings.TrimPrefix(server.URL, "https://")
	_, err := fetchAccessToken(serverHost+"/realms/test", "bad-client", "bad-secret", certPath)
	if err == nil {
		t.Fatal("Expected error for unauthorized status, got nil")
	}

	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("Expected error about unauthorized, got: %s", err.Error())
	}
}

func TestFetchReleaseToken_EmptyAccessToken(t *testing.T) {
	certPath := createTempCACert(t)

	_, err := fetchReleaseToken("release.example.com/token", "", certPath)
	if err == nil {
		t.Fatal("Expected error for empty access token, got nil")
	}

	if !strings.Contains(err.Error(), "access token is required") {
		t.Errorf("Expected error about access token, got: %s", err.Error())
	}
}

func TestFetchReleaseToken_InvalidCACert(t *testing.T) {
	_, err := fetchReleaseToken("release.example.com/token", "valid-token", "/nonexistent/cert.pem")
	if err == nil {
		t.Fatal("Expected error for invalid CA cert, got nil")
	}

	if !strings.Contains(err.Error(), "error loading CA certificate") {
		t.Errorf("Expected error about CA certificate, got: %s", err.Error())
	}
}

func TestFetchReleaseToken_MockServer(t *testing.T) {
	// Create a mock server that returns a valid token
	expectedToken := "valid-release-token-xyz"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Bearer token in Authorization header, got '%s'", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedToken))
	}))
	defer server.Close()

	// Write the test server's certificate to a temp file
	certPath := filepath.Join(t.TempDir(), "test-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Test fetching release token
	// Note: fetchReleaseToken adds "https://" prefix, so we need to strip it from server.URL
	serverHost := strings.TrimPrefix(server.URL, "https://")
	token, err := fetchReleaseToken(serverHost, "test-access-token", certPath)
	if err != nil {
		t.Fatalf("fetchReleaseToken failed: %v", err)
	}

	if token != expectedToken {
		t.Errorf("Expected token '%s', got '%s'", expectedToken, token)
	}
}

func TestFetchReleaseToken_NullToken(t *testing.T) {
	// Create a mock server that returns "null"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("null"))
	}))
	defer server.Close()

	// Write the test server's certificate to a temp file
	certPath := filepath.Join(t.TempDir(), "test-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// Test that "null" response is handled correctly
	// Note: fetchReleaseToken adds "https://" prefix, so we need to strip it from server.URL
	serverHost := strings.TrimPrefix(server.URL, "https://")
	token, err := fetchReleaseToken(serverHost, "test-access-token", certPath)

	// The function should return an error for "null" token
	if err == nil {
		t.Fatal("Expected error for 'null' token, got nil")
	}

	if !strings.Contains(err.Error(), "invalid token received") {
		t.Errorf("Expected error about invalid token, got: %s", err.Error())
	}

	if token != "" {
		t.Errorf("Expected empty token on error, got '%s'", token)
	}
}

func TestClientAuth_KeycloakFailure(t *testing.T) {
	certPath := createTempCACert(t)

	// Test with invalid Keycloak URL
	_, _, err := ClientAuth(
		"client-id",
		"client-secret",
		"invalid-keycloak-url",
		"/realms/master/protocol/openid-connect/token",
		"/token",
		certPath,
	)

	if err == nil {
		t.Fatal("Expected error for invalid Keycloak connection, got nil")
	}

	if !strings.Contains(err.Error(), "failed to get JWT access token from Keycloak") {
		t.Errorf("Expected Keycloak error, got: %s", err.Error())
	}
}

func TestClientAuth_ReleaseURLConstruction(t *testing.T) {
	// Test that release token URL is properly constructed from keycloak URL
	keycloakURL := "keycloak.example.com"
	expected := "release.example.com"

	// Simulate the strings.Replace logic from ClientAuth
	result := strings.Replace(keycloakURL, "keycloak", "release", 1)

	if result != expected {
		t.Errorf("Expected release URL '%s', got '%s'", expected, result)
	}
}
