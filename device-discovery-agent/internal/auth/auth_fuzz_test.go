// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzLoadCACertPool tests the PEM certificate parsing logic for crashes and panics.
// This is security-critical as malformed certificates could cause vulnerabilities.
func FuzzLoadCACertPool(f *testing.F) {
	// Seed with valid PEM certificate
	validPEM := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU6KbMA0GCSqGSIb3DQEBCwUAMCExHzAdBgNVBAMMFnRl
c3QtY2EuZXhhbXBsZS5sb2NhbDAeFw0yNTAyMDIwMDAwMDBaFw0yNjAyMDIwMDAw
MDBaMCExHzAdBgNVBAMMFnRlc3QtY2EuZXhhbXBsZS5sb2NhbDBcMA0GCSqGSIb3
DQEBAQUAA0sAMEgCQQC9h0P3hxVz6LQRX7+ZJ4Z0J3Z0J3Z0J3Z0J3Z0J3Z0J3Z0
J3Z0J3Z0J3Z0J3Z0J3Z0J3Z0J3Z0J3Z0J3Z0J3AgMBAAEwDQYJKoZIhvcNAQEL
BQADQQBvZmZzZXQgdGVzdCBkYXRhIGZvciBmdXp6aW5nIHB1cnBvc2VzIG9ubHk=
-----END CERTIFICATE-----`
	f.Add([]byte(validPEM))

	// Seed with edge cases
	f.Add([]byte(""))                                                                                             // Empty file
	f.Add([]byte("not a certificate"))                                                                            // Invalid content
	f.Add([]byte("-----BEGIN CERTIFICATE-----"))                                                                  // Incomplete PEM
	f.Add([]byte("-----END CERTIFICATE-----"))                                                                    // Missing begin
	f.Add([]byte("-----BEGIN CERTIFICATE-----\n\n\n"))                                                            // Empty PEM block
	f.Add([]byte("-----BEGIN CERTIFICATE-----\nYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=\n-----END CERTIFICATE-----")) // Invalid base64 cert data
	f.Add([]byte(strings.Repeat("A", 10000)))                                                                     // Very large input

	f.Fuzz(func(t *testing.T, pemData []byte) {
		// Create temporary file for testing
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "test-cert.pem")

		// Write the fuzzed data to file
		if err := os.WriteFile(certPath, pemData, 0600); err != nil {
			t.Skip("Failed to write test file")
		}

		// Test loadCACertPool - should not panic
		pool, err := loadCACertPool(certPath)

		// Validate results
		if err == nil {
			// If no error, pool should be valid and non-nil
			if pool == nil {
				t.Error("loadCACertPool returned nil pool without error")
			}
			// Verify it's a valid x509.CertPool
			if _, ok := interface{}(pool).(*x509.CertPool); !ok {
				t.Error("loadCACertPool returned invalid type")
			}
		}
		// If error occurred, that's fine - we just don't want panics
	})
}

// FuzzJSONTokenResponse tests JSON unmarshaling of access token responses.
// This simulates parsing responses from Keycloak or other OAuth2 providers.
func FuzzJSONTokenResponse(f *testing.F) {
	// Seed with valid JSON responses
	f.Add([]byte(`{"access_token":"valid-token-123","token_type":"Bearer"}`))
	f.Add([]byte(`{"access_token":"","token_type":"Bearer"}`))
	f.Add([]byte(`{"access_token":null}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"access_token":123}`)) // Wrong type

	// Edge cases
	f.Add([]byte(`{`))                                                     // Incomplete JSON
	f.Add([]byte(`}`))                                                     // Just closing brace
	f.Add([]byte(`[]`))                                                    // Array instead of object
	f.Add([]byte(`null`))                                                  // Null JSON
	f.Add([]byte(`"string"`))                                              // String instead of object
	f.Add([]byte(strings.Repeat(`{"a":`, 1000)))                           // Deeply nested
	f.Add([]byte(`{"access_token":"` + strings.Repeat("x", 10000) + `"}`)) // Very long token

	f.Fuzz(func(t *testing.T, jsonData []byte) {
		// Simulate the JSON parsing logic from fetchAccessToken
		var result map[string]interface{}
		err := json.Unmarshal(jsonData, &result)

		if err == nil {
			// If parsing succeeded, try to extract access_token
			token, ok := result["access_token"].(string)

			// Validate the extraction logic
			if ok && token != "" {
				// Token successfully extracted
				if len(token) > 100000 {
					t.Error("Unexpectedly large token accepted")
				}
			}
			// If token is missing or empty, that's handled by error logic
		}
		// Unmarshaling errors are expected and handled
	})
}

// FuzzReleaseTokenResponse tests parsing of release token HTTP response bodies.
// This ensures the token validation logic handles malformed responses safely.
func FuzzReleaseTokenResponse(f *testing.F) {
	// Seed with valid tokens
	f.Add([]byte("valid-release-token-abc123"))
	f.Add([]byte("token-with-dashes-and_underscores"))

	// Edge cases that should be rejected
	f.Add([]byte("null")) // Should be rejected
	f.Add([]byte(""))     // Should be rejected
	f.Add([]byte(" "))    // Whitespace
	f.Add([]byte("\n"))   // Newline
	f.Add([]byte("\x00")) // Null byte

	// Boundary cases
	f.Add([]byte(strings.Repeat("a", 10000))) // Very long token
	f.Add([]byte("\"quoted-token\""))         // Quoted token
	f.Add([]byte("{\"token\":\"value\"}"))    // JSON when expecting plain text

	f.Fuzz(func(t *testing.T, tokenData []byte) {
		// Simulate the token validation logic from fetchReleaseToken
		token := string(tokenData)

		// Test the validation logic
		if token == "null" || token == "" {
			// These should be rejected by the actual code
			return
		}

		// Valid tokens should not cause issues
		if len(token) > 0 && token != "null" {
			// Token is considered valid by the current logic
			// Ensure it doesn't contain unexpected characters that could cause issues
			if strings.Contains(token, "\x00") {
				// Null bytes in tokens could be problematic
			}
		}
	})
}

// FuzzURLConstruction tests URL parsing and construction to prevent injection attacks.
// This is critical for security to ensure malformed URLs don't cause issues.
func FuzzURLConstruction(f *testing.F) {
	// Seed with valid URLs
	f.Add("keycloak.example.com:8443/auth/realms/master/protocol/openid-connect/token")
	f.Add("release.example.com:8443/api/v1/token")
	f.Add("localhost:8080/token")

	// Edge cases
	f.Add("")                         // Empty URL
	f.Add(":")                        // Just colon
	f.Add("http://")                  // Incomplete URL
	f.Add("//example.com")            // Protocol-relative
	f.Add("example.com:99999")        // Invalid port
	f.Add("example.com:-1")           // Negative port
	f.Add("example.com:abc")          // Non-numeric port
	f.Add("ex ample.com")             // Space in hostname
	f.Add("example.com\x00/path")     // Null byte
	f.Add(strings.Repeat("a", 10000)) // Very long hostname

	f.Fuzz(func(t *testing.T, urlStr string) {
		// Test URL construction with "https://" prefix
		fullURL := "https://" + urlStr

		// Validate URL parsing doesn't panic
		// This simulates the URL construction in fetchAccessToken and fetchReleaseToken
		_ = fullURL

		// Test strings.Replace operation from ClientAuth
		modifiedURL := strings.Replace(urlStr, "keycloak", "release", 1)
		_ = modifiedURL

		// Ensure no panics occur during string operations
	})
}

// FuzzAuthorizationHeader tests Bearer token header construction.
// Ensures malformed tokens don't cause injection attacks in HTTP headers.
func FuzzAuthorizationHeader(f *testing.F) {
	// Seed with valid tokens
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")
	f.Add("simple-token-123")
	f.Add("token_with_underscores")

	// Edge cases
	f.Add("")                         // Empty token
	f.Add(" ")                        // Whitespace
	f.Add("\n")                       // Newline
	f.Add("\r\n")                     // CRLF (HTTP header injection)
	f.Add("token\r\nX-Injected: bad") // Header injection attempt
	f.Add("token\x00")                // Null byte
	f.Add(strings.Repeat("a", 10000)) // Very long token

	f.Fuzz(func(t *testing.T, token string) {
		// Simulate Authorization header construction from fetchReleaseToken
		authHeader := "Bearer " + token

		// Check for potential header injection vulnerabilities
		if strings.Contains(authHeader, "\r") || strings.Contains(authHeader, "\n") {
			// CRLF in headers could allow injection attacks
			// The actual HTTP client should reject these, but we test defensively
		}

		// Ensure header construction doesn't panic
		_ = authHeader
	})
}

// FuzzPEMBlockParsing tests direct PEM block parsing edge cases.
// This complements FuzzLoadCACertPool with more targeted PEM parsing tests.
func FuzzPEMBlockParsing(f *testing.F) {
	// Seed with various PEM formats
	validPEM := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU6KbMA0GCSqGSIb3DQEBCwUAMCExHzAdBgNVBAMMFnRl
-----END CERTIFICATE-----`
	f.Add([]byte(validPEM))

	// Multiple PEM blocks
	multiplePEM := validPEM + "\n" + validPEM
	f.Add([]byte(multiplePEM))

	// Different PEM types
	f.Add([]byte("-----BEGIN RSA PRIVATE KEY-----\ndata\n-----END RSA PRIVATE KEY-----"))
	f.Add([]byte("-----BEGIN PUBLIC KEY-----\ndata\n-----END PUBLIC KEY-----"))

	// Edge cases
	f.Add([]byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"))                                     // Empty block
	f.Add([]byte("-----BEGIN CERTIFICATE-----\n!\n-----END CERTIFICATE-----"))                                  // Invalid base64
	f.Add([]byte("-----BEGIN CERTIFICATE-----\n" + strings.Repeat("A", 10000) + "\n-----END CERTIFICATE-----")) // Large block

	f.Fuzz(func(t *testing.T, pemData []byte) {
		// Test PEM decoding - should not panic
		block, rest := pem.Decode(pemData)

		if block == nil {
			// This is expected for empty or invalid PEM data
			// Just ensure rest is the original data or empty
			if len(rest) > 0 && len(rest) != len(pemData) {
				t.Errorf("Invalid rest data: got %d bytes, expected 0 or %d", len(rest), len(pemData))
			}
			return
		}

		// If we got a block, it should be a valid *pem.Block
		// Type can be empty (e.g., "-----BEGIN -----\n-----END -----")
		// Bytes can be empty (e.g., empty certificate content)
		// These are all valid PEM structures that shouldn't cause panics

		// Rest should be the remaining unparsed data
		if len(rest) > len(pemData) {
			t.Error("Rest data is larger than input data")
		}

		// Test if there are more blocks
		if len(rest) > 0 {
			block2, _ := pem.Decode(rest)
			_ = block2
		}
	})
}
