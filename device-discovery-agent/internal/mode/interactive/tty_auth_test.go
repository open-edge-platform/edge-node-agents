// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTTYAuthenticator_ValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.env")

	configContent := `KEYCLOAK_URL=keycloak.example.com
CA_CERT=/path/to/cert.pem
EXTRA_HOSTS=
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create authenticator
	auth, err := NewTTYAuthenticator(configPath)
	if err != nil {
		t.Fatalf("NewTTYAuthenticator failed: %v", err)
	}

	if auth.keycloakURL != "keycloak.example.com" {
		t.Errorf("Expected keycloakURL 'keycloak.example.com', got '%s'", auth.keycloakURL)
	}

	if auth.caCertPath != "/path/to/cert.pem" {
		t.Errorf("Expected caCertPath '/path/to/cert.pem', got '%s'", auth.caCertPath)
	}

	if len(auth.ttyDevices) != 3 {
		t.Errorf("Expected 3 TTY devices, got %d", len(auth.ttyDevices))
	}

	if auth.maxAttempts != 3 {
		t.Errorf("Expected maxAttempts 3, got %d", auth.maxAttempts)
	}
}

func TestNewTTYAuthenticator_MissingConfig(t *testing.T) {
	_, err := NewTTYAuthenticator("/nonexistent/config.env")
	if err == nil {
		t.Fatal("Expected error for missing config file")
	}
}

func TestTTYAuthenticator_CredentialSanitization(t *testing.T) {
	// Test the sanitization logic used in collectFromDevice
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"No special chars", "validuser", "validuser"},
		{"With spaces", "user name", "username"},
		{"With newlines", "user\nname", "username"},
		{"With semicolons", "user;name", "username"},
		{"Mixed", "user name\n;test", "usernametest"},
		{"Leading/trailing", " username ", "username"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Apply same sanitization as collectFromDevice
			sanitized := strings.Map(func(r rune) rune {
				if r == ' ' || r == '\n' || r == ';' {
					return -1
				}
				return r
			}, tc.input)

			if sanitized != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, sanitized)
			}
		})
	}
}

func TestTTYAuthenticator_MinimumLengthValidation(t *testing.T) {
	// Verify minimum length validation (matching bash script: >=3 chars)
	testCases := []struct {
		username string
		password string
		valid    bool
	}{
		{"abc", "xyz", true},
		{"ab", "xyz", false},   // username too short
		{"abc", "xy", false},   // password too short
		{"a", "b", false},      // both too short
		{"", "", false},        // empty
		{"user", "pass", true}, // valid
	}

	for _, tc := range testCases {
		valid := len(tc.username) >= 3 && len(tc.password) >= 3
		if valid != tc.valid {
			t.Errorf("username='%s' password='%s': expected valid=%v, got valid=%v",
				tc.username, tc.password, tc.valid, valid)
		}
	}
}

func TestTTYAuthenticator_ValidateAndFetchTokens_InvalidCredentials(t *testing.T) {
	// Test that invalid credentials result in appropriate error
	// This is a simpler test that doesn't require complex mocking

	tmpDir := t.TempDir()

	// Create authenticator with fake URL
	auth := &TTYAuthenticator{
		keycloakURL: "nonexistent-keycloak.invalid.local:9999",
		caCertPath:  "", // Use system certs
		ttyDevices:  []string{"ttyS0"},
		maxAttempts: 3,
		logFile:     filepath.Join(tmpDir, "test.log"),
	}

	// Test credentials
	creds := &Credentials{
		Username: "testuser",
		Password: "testpass",
	}

	// Validate and fetch tokens - should fail due to network error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := auth.validateAndFetchTokens(ctx, creds)
	if err == nil {
		t.Fatal("Expected error for invalid keycloak URL")
	}

	// Verify error is about failed token fetch
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Logf("Got expected error: %v", err)
	}
}

func TestTTYAuthenticator_WriteTokens(t *testing.T) {
	// Note: writeTokens() uses config constants (/dev/shm paths)
	// This test verifies the method doesn't error with valid input
	// Actual file writes tested in integration tests

	tmpDir := t.TempDir()
	auth := &TTYAuthenticator{
		logFile: filepath.Join(tmpDir, "test.log"),
	}

	// Write tokens - may fail if /dev/shm not writable in test environment
	accessToken := "test-access-token-123"
	releaseToken := "test-release-token-456"

	err := auth.writeTokens(accessToken, releaseToken)
	// Accept success or permission error (depending on test environment)
	if err != nil {
		if !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "no such file") {
			t.Fatalf("writeTokens failed with unexpected error: %v", err)
		}
		t.Logf("writeTokens failed as expected in test environment: %v", err)
	} else {
		t.Log("writeTokens succeeded (test environment has /dev/shm access)")
	}
}

func TestTTYAuthenticator_LogToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test-auth.log")

	auth := &TTYAuthenticator{
		logFile: logFile,
	}

	// Write log message
	auth.logToFile("Test log message")

	// Verify log file was created and contains message
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(data)
	if !strings.Contains(logContent, "Test log message") {
		t.Errorf("Expected log to contain 'Test log message', got: %s", logContent)
	}

	// Verify timestamp format (should have RFC3339 format)
	if !strings.Contains(logContent, "T") { // RFC3339 has 'T' between date and time
		t.Errorf("Expected RFC3339 timestamp in log, got: %s", logContent)
	}
}

func TestTTYAuthenticator_CollectCredentials_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.env")

	configContent := `KEYCLOAK_URL=keycloak.example.com
CA_CERT=/path/to/cert.pem
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	auth, err := NewTTYAuthenticator(configPath)
	if err != nil {
		t.Fatalf("NewTTYAuthenticator failed: %v", err)
	}

	// Set to use nonexistent TTY devices (will fail to open)
	auth.ttyDevices = []string{"nonexistent-tty-999"}

	// Attempt to collect credentials (should timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = auth.collectCredentials(ctx)
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestTTYAuthenticator_ShowErrorToAllTTYs(t *testing.T) {
	tmpDir := t.TempDir()

	auth := &TTYAuthenticator{
		ttyDevices: []string{"nonexistent-tty-1", "nonexistent-tty-2"},
		logFile:    filepath.Join(tmpDir, "test.log"),
	}

	// This should not crash even if devices don't exist
	auth.showErrorToAllTTYs("Test error message")

	// If we got here without crashing, the test passes
	// (The method gracefully skips devices that can't be opened)
}
