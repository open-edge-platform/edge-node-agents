// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
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

func TestFetchPasswordGrantToken_Success(t *testing.T) {
	mockToken := "test-password-grant-token-abc123"

	// Create mock Keycloak server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}

		// Verify form values
		if r.FormValue("grant_type") != "password" {
			t.Errorf("Expected grant_type=password, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("username") != "testuser" {
			t.Errorf("Expected username=testuser, got %s", r.FormValue("username"))
		}
		if r.FormValue("password") != "testpass" {
			t.Errorf("Expected password=testpass, got %s", r.FormValue("password"))
		}
		if r.FormValue("client_id") != "test-client" {
			t.Errorf("Expected client_id=test-client, got %s", r.FormValue("client_id"))
		}

		// Return token
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": mockToken,
			"token_type":   "Bearer",
			"expires_in":   300,
		})
	}))
	defer server.Close()

	// Write server cert
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	os.WriteFile(certPath, certPEM, 0644)

	// Test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverHost := strings.TrimPrefix(server.URL, "https://")
	token, err := FetchPasswordGrantToken(ctx, PasswordGrantParams{
		KeycloakURL: serverHost,
		TokenPath:   "/token",
		Username:    "testuser",
		Password:    "testpass",
		ClientID:    "test-client",
		Scope:       "openid",
		CACertPath:  certPath,
	})

	if err != nil {
		t.Fatalf("FetchPasswordGrantToken failed: %v", err)
	}

	if token != mockToken {
		t.Errorf("Expected token %s, got %s", mockToken, token)
	}
}

func TestFetchPasswordGrantToken_InvalidCredentials(t *testing.T) {
	// Mock server that returns 401
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	os.WriteFile(certPath, certPEM, 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverHost := strings.TrimPrefix(server.URL, "https://")
	_, err := FetchPasswordGrantToken(ctx, PasswordGrantParams{
		KeycloakURL: serverHost,
		TokenPath:   "/token",
		Username:    "baduser",
		Password:    "badpass",
		ClientID:    "test-client",
		CACertPath:  certPath,
	})

	if err == nil {
		t.Fatal("Expected error for invalid credentials")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 error, got: %v", err)
	}
}

func TestFetchPasswordGrantToken_Timeout(t *testing.T) {
	// Mock server that delays response
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	os.WriteFile(certPath, certPEM, 0644)

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	serverHost := strings.TrimPrefix(server.URL, "https://")
	_, err := FetchPasswordGrantToken(ctx, PasswordGrantParams{
		KeycloakURL: serverHost,
		TokenPath:   "/token",
		Username:    "testuser",
		Password:    "testpass",
		ClientID:    "test-client",
		CACertPath:  certPath,
	})

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestFetchPasswordGrantToken_MalformedJSON(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": malformed json`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	os.WriteFile(certPath, certPEM, 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverHost := strings.TrimPrefix(server.URL, "https://")
	_, err := FetchPasswordGrantToken(ctx, PasswordGrantParams{
		KeycloakURL: serverHost,
		TokenPath:   "/token",
		Username:    "testuser",
		Password:    "testpass",
		ClientID:    "test-client",
		CACertPath:  certPath,
	})

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}
}
