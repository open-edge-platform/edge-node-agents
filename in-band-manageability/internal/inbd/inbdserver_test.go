/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package inbd

import (
	"context"
	"strings"
	"testing"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL",
			rawURL:  "https://example.com/firmware.bin",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with port",
			rawURL:  "https://example.com:8443/firmware.bin",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with path and query",
			rawURL:  "https://example.com/path/to/firmware.bin?version=1.0",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with IP address",
			rawURL:  "https://192.168.1.100:8443/firmware.bin",
			wantErr: false,
		},
		{
			name:    "empty URL",
			rawURL:  "",
			wantErr: true,
			errMsg:  "URL is empty",
		},
		{
			name:    "HTTP URL (not HTTPS)",
			rawURL:  "http://example.com/firmware.bin",
			wantErr: true,
			errMsg:  "URL must use https scheme",
		},
		{
			name:    "FTP URL",
			rawURL:  "ftp://example.com/firmware.bin",
			wantErr: true,
			errMsg:  "URL must use https scheme",
		},
		{
			name:    "malformed URL",
			rawURL:  "not-a-url",
			wantErr: true,
			errMsg:  "URL is not valid:",
		},
		{
			name:    "URL with space",
			rawURL:  "https://example .com/firmware.bin",
			wantErr: true,
			errMsg:  "URL is not valid:",
		},
		{
			name:    "URL without scheme",
			rawURL:  "example.com/firmware.bin",
			wantErr: true,
			errMsg:  "URL is not valid:",
		},
		{
			name:    "HTTPS URL without host",
			rawURL:  "https:///firmware.bin",
			wantErr: true,
			errMsg:  "URL must have a host",
		},
		{
			name:    "URL with encoded spaces (valid)",
			rawURL:  "https://example.com/firmware%20bin.tar",
			wantErr: false,
		},
		{
			name:    "URL with unicode domain",
			rawURL:  "https://例え.テスト/firmware.bin",
			wantErr: false,
		},
		{
			name:    "URL with subdomain",
			rawURL:  "https://api.example.com/v1/firmware.bin",
			wantErr: false,
		},
		{
			name:    "URL with fragment",
			rawURL:  "https://example.com/firmware.bin#section1",
			wantErr: false,
		},
		{
			name:    "URL with authentication info",
			rawURL:  "https://user:pass@example.com/firmware.bin",
			wantErr: false,
		},
		{
			name:    "localhost URL",
			rawURL:  "https://localhost:8443/firmware.bin",
			wantErr: false,
		},
		{
			name:    "URL with unusual but valid path",
			rawURL:  "https://example.com/path%20with%20spaces/firmware.bin",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.rawURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateURL() error = nil, wantErr %v", tt.wantErr)
					return
				}

				// Check if error message contains expected content
				if tt.errMsg != "" {
					errStr := err.Error()
					found := false
					if tt.errMsg == errStr {
						found = true
					} else {
						// For "URL is not valid:" messages, check if it starts with the expected prefix
						if tt.errMsg == "URL is not valid:" && len(errStr) > len(tt.errMsg) && errStr[:len(tt.errMsg)] == tt.errMsg {
							found = true
						}
					}
					if !found {
						t.Errorf("validateURL() error = %v, want error containing %v", err, tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("validateURL() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
		desc    string
	}{
		{
			name:    "extremely long URL with valid characters",
			rawURL:  "https://example.com/" + strings.Repeat("a", 2000),
			wantErr: false,
			desc:    "Should handle very long URLs with valid characters",
		},
		{
			name:    "URL with only scheme",
			rawURL:  "https://",
			wantErr: true,
			desc:    "Should reject URL with only scheme",
		},
		{
			name:    "URL with default HTTPS port",
			rawURL:  "https://example.com:443/firmware.bin",
			wantErr: false,
			desc:    "Should accept explicit default HTTPS port",
		},
		{
			name:    "URL with custom port",
			rawURL:  "https://example.com:9443/firmware.bin",
			wantErr: false,
			desc:    "Should accept custom ports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.rawURL)

			if tt.wantErr && err == nil {
				t.Errorf("validateURL() error = nil, wantErr %v. %s", tt.wantErr, tt.desc)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateURL() error = %v, wantErr %v. %s", err, tt.wantErr, tt.desc)
			}
		})
	}
}

// BenchmarkValidateURL benchmarks the validateURL function
func BenchmarkValidateURL(b *testing.B) {
	testURLs := []string{
		"https://example.com/firmware.bin",
		"https://192.168.1.100:8443/path/to/firmware.bin",
		"https://api.example.com/v1/firmware.bin?version=1.0",
		"http://example.com/firmware.bin", // This will error
		"not-a-url",                       // This will error
	}

	for _, url := range testURLs {
		b.Run("url_"+url[:min(len(url), 20)], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = validateURL(url)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestUpdateSystemSoftware_URLValidation tests the URL validation logic in UpdateSystemSoftware
// Note: This test may skip URL validation tests in CI environments where OS detection fails
func TestUpdateSystemSoftware_URLValidation(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty URL is valid",
			url:         "",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL",
			url:         "https://example.com/update.deb",
			expectError: false,
		},
		{
			name:        "invalid HTTP URL",
			url:         "http://example.com/update.deb",
			expectError: true,
			errorMsg:    "URL must use https scheme",
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "URL is not valid:",
		},
		{
			name:        "URL without host",
			url:         "https:///update.deb",
			expectError: true,
			errorMsg:    "URL must have a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test URL
			req := &pb.UpdateSystemSoftwareRequest{
				Url:  tt.url,
				Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL,
			}

			// Create server instance
			server := &InbdServer{}

			// Call the method
			ctx := context.Background()
			resp, err := server.UpdateSystemSoftware(ctx, req)

			// Should never return an error from the method itself
			if err != nil {
				t.Errorf("UpdateSystemSoftware() returned unexpected error: %v", err)
				return
			}

			// Check response
			if resp == nil {
				t.Fatal("UpdateSystemSoftware() returned nil response")
			}

			// In CI environments, OS detection might fail before URL validation
			// If OS detection fails (415), we can't test URL validation in this integration test
			if resp.StatusCode == 415 {
				if strings.Contains(resp.Error, "lsb_release") || strings.Contains(resp.Error, "executable file not found") {
					t.Skipf("Skipping URL validation test due to OS detection failure in CI: %v", resp.Error)
				}
				// If it's not a CI-related OS detection failure, continue with validation
			}

			if tt.expectError {
				// Should return a 400 status code for URL validation errors
				// But if OS detection failed first (415), that takes precedence
				if resp.StatusCode != 400 && resp.StatusCode != 415 {
					t.Errorf("UpdateSystemSoftware() StatusCode = %v, want 400 for URL validation error or 415 for OS detection error", resp.StatusCode)
				}

				// Only check error message if we got the expected URL validation error (400)
				if resp.StatusCode == 400 && tt.errorMsg != "" && !strings.Contains(resp.Error, tt.errorMsg) {
					t.Errorf("UpdateSystemSoftware() Error = %v, want containing %v", resp.Error, tt.errorMsg)
				}
			} else {
				// For valid URLs, we might get other errors (like OS detection failure),
				// but not URL validation errors (status 400)
				if resp.StatusCode == 400 && strings.Contains(resp.Error, "URL") {
					t.Errorf("UpdateSystemSoftware() unexpected URL validation error: %v", resp.Error)
				}
			}
		})
	}
}

// TestUpdateSystemSoftware_URLValidationDirect tests URL validation directly
// This test isolates URL validation from OS detection dependencies
func TestUpdateSystemSoftware_URLValidationDirect(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty URL is valid",
			url:         "",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL",
			url:         "https://example.com/update.deb",
			expectError: false,
		},
		{
			name:        "invalid HTTP URL",
			url:         "http://example.com/update.deb",
			expectError: true,
			errorMsg:    "URL must use https scheme",
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "URL is not valid:",
		},
		{
			name:        "URL without host",
			url:         "https:///update.deb",
			expectError: true,
			errorMsg:    "URL must have a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validateURL function directly (if URL is not empty)
			if tt.url != "" {
				err := validateURL(tt.url)

				if tt.expectError {
					if err == nil {
						t.Errorf("validateURL() error = nil, wantErr %v", tt.expectError)
						return
					}

					if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("validateURL() Error = %v, want containing %v", err.Error(), tt.errorMsg)
					}
				} else {
					if err != nil {
						t.Errorf("validateURL() error = %v, wantErr %v", err, tt.expectError)
					}
				}
			}
		})
	}
}

// TestUpdateSystemSoftware_EmptyRequest tests behavior with minimal request
func TestUpdateSystemSoftware_EmptyRequest(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	req := &pb.UpdateSystemSoftwareRequest{
		Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL,
	}

	resp, err := server.UpdateSystemSoftware(ctx, req)

	// Should not return an error from the method itself
	if err != nil {
		t.Errorf("UpdateSystemSoftware() returned unexpected error: %v", err)
	}

	// Should return a response
	if resp == nil {
		t.Fatal("UpdateSystemSoftware() returned nil response")
	}

	// Response should have a status code
	if resp.StatusCode == 0 {
		t.Error("UpdateSystemSoftware() returned response with zero status code")
	}
}

// TestUpdateSystemSoftware_InputValidation tests various input validation scenarios
func TestUpdateSystemSoftware_InputValidation(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	tests := []struct {
		name     string
		request  *pb.UpdateSystemSoftwareRequest
		wantCode int32
	}{
		{
			name: "request with valid mode",
			request: &pb.UpdateSystemSoftwareRequest{
				Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL,
			},
			wantCode: 0, // Should get some response code, not necessarily success
		},
		{
			name: "request with download only mode",
			request: &pb.UpdateSystemSoftwareRequest{
				Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_DOWNLOAD_ONLY,
			},
			wantCode: 0, // Should get some response code
		},
		{
			name: "request with no download mode",
			request: &pb.UpdateSystemSoftwareRequest{
				Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD,
			},
			wantCode: 0, // Should get some response code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.UpdateSystemSoftware(ctx, tt.request)

			// Should not return an error from the method itself
			if err != nil {
				t.Errorf("UpdateSystemSoftware() returned unexpected error: %v", err)
				return
			}

			// Should return a response
			if resp == nil {
				t.Fatal("UpdateSystemSoftware() returned nil response")
			}

			// Should have some status code (success or error)
			if resp.StatusCode == 0 {
				t.Error("UpdateSystemSoftware() returned response with zero status code")
			}
		})
	}
}

// TestUpdateSystemSoftware_ContextHandling tests that the function properly handles context
func TestUpdateSystemSoftware_ContextHandling(t *testing.T) {
	server := &InbdServer{}

	req := &pb.UpdateSystemSoftwareRequest{
		Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL,
	}

	// Test with background context
	t.Run("background context", func(t *testing.T) {
		ctx := context.Background()
		resp, err := server.UpdateSystemSoftware(ctx, req)

		if err != nil {
			t.Errorf("UpdateSystemSoftware() with background context failed: %v", err)
		}

		if resp == nil {
			t.Error("UpdateSystemSoftware() returned nil response with background context")
		}
	})

	// Test with context with timeout (but not expired)
	t.Run("context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		resp, err := server.UpdateSystemSoftware(ctx, req)

		if err != nil {
			t.Errorf("UpdateSystemSoftware() with timeout context failed: %v", err)
		}

		if resp == nil {
			t.Error("UpdateSystemSoftware() returned nil response with timeout context")
		}
	})
}

// TestUpdateSystemSoftware_OSDetectionFailure tests behavior when OS detection fails
// This simulates the CI environment where lsb_release might be missing
func TestUpdateSystemSoftware_OSDetectionFailure(t *testing.T) {
	if !testing.Short() {
		t.Skip("Skipping in non-short mode to avoid simulating CI environment issues")
	}

	server := &InbdServer{}
	ctx := context.Background()

	// Test with invalid URL to see if URL validation would be reached
	req := &pb.UpdateSystemSoftwareRequest{
		Url:  "http://example.com/update.deb", // Invalid HTTP URL
		Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL,
	}

	resp, err := server.UpdateSystemSoftware(ctx, req)

	// Should not return an error from the method itself
	if err != nil {
		t.Errorf("UpdateSystemSoftware() returned unexpected error: %v", err)
		return
	}

	// Should return a response
	if resp == nil {
		t.Fatal("UpdateSystemSoftware() returned nil response")
	}

	// The response should have some error (either URL validation or OS detection)
	// In CI without lsb_release, OS detection fails first with 415
	// With lsb_release, URL validation should fail with 400
	if resp.StatusCode != 400 && resp.StatusCode != 415 {
		t.Errorf("UpdateSystemSoftware() StatusCode = %v, want 400 (URL validation) or 415 (OS detection failure)", resp.StatusCode)
	}

	// Log what we got for debugging
	t.Logf("Got StatusCode: %d, Error: %s", resp.StatusCode, resp.Error)

	// If we got 415, it should be an OS detection error
	if resp.StatusCode == 415 {
		// This is expected in CI environments without lsb_release
		if !strings.Contains(resp.Error, "lsb_release") && !strings.Contains(resp.Error, "executable file not found") {
			t.Logf("OS detection failed with different error than expected: %v", resp.Error)
		}
	}

	// If we got 400, it should be URL validation error
	if resp.StatusCode == 400 {
		if !strings.Contains(resp.Error, "URL must use https scheme") {
			t.Errorf("Expected URL validation error, got: %v", resp.Error)
		}
	}
}

// TestInbdServer_InputValidation tests basic input validation at the server layer
func TestInbdServer_InputValidation(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	t.Run("Query nil request", func(t *testing.T) {
		resp, err := server.Query(ctx, nil)
		if err != nil {
			t.Errorf("Query() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("Query() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("Query() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "request is required") {
			t.Errorf("Query() Error = %v, want containing 'request is required'", resp.Error)
		}
	})

	t.Run("Query unspecified option", func(t *testing.T) {
		req := &pb.QueryRequest{Option: pb.QueryOption_QUERY_OPTION_UNSPECIFIED}
		resp, err := server.Query(ctx, req)
		if err != nil {
			t.Errorf("Query() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("Query() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("Query() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "invalid query option") {
			t.Errorf("Query() Error = %v, want containing 'invalid query option'", resp.Error)
		}
	})

	t.Run("UpdateFirmware empty URL", func(t *testing.T) {
		req := &pb.UpdateFirmwareRequest{Url: ""}
		resp, err := server.UpdateFirmware(ctx, req)
		if err != nil {
			t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("UpdateFirmware() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("UpdateFirmware() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "URL is required") {
			t.Errorf("UpdateFirmware() Error = %v, want containing 'URL is required'", resp.Error)
		}
	})

	t.Run("Config operations require validation", func(t *testing.T) {
		// Test GetConfig with empty path
		req := &pb.GetConfigRequest{Path: ""}
		resp, err := server.GetConfig(ctx, req)
		if err != nil {
			t.Errorf("GetConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("GetConfig() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("GetConfig() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "path is required") {
			t.Errorf("GetConfig() Error = %v, want containing 'path is required'", resp.Error)
		}
	})

	t.Run("LoadConfig empty URI", func(t *testing.T) {
		req := &pb.LoadConfigRequest{Uri: ""}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("LoadConfig() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "uri is required") {
			t.Errorf("LoadConfig() Error = %v, want containing 'uri is required'", resp.Error)
		}
	})
}

// TestInbdServer_ResponseFormat tests that all methods return proper response format
func TestInbdServer_ResponseFormat(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	t.Run("All methods return non-nil responses", func(t *testing.T) {
		// Test that methods don't panic and return responses
		methods := []func() (interface{}, error){
			func() (interface{}, error) { return server.Query(ctx, nil) },
			func() (interface{}, error) { return server.UpdateFirmware(ctx, &pb.UpdateFirmwareRequest{}) },
			func() (interface{}, error) { return server.LoadConfig(ctx, &pb.LoadConfigRequest{}) },
			func() (interface{}, error) { return server.GetConfig(ctx, &pb.GetConfigRequest{}) },
			func() (interface{}, error) { return server.SetConfig(ctx, &pb.SetConfigRequest{}) },
			func() (interface{}, error) { return server.AppendConfig(ctx, &pb.AppendConfigRequest{}) },
			func() (interface{}, error) { return server.RemoveConfig(ctx, &pb.RemoveConfigRequest{}) },
		}

		for i, method := range methods {
			resp, err := method()
			if err != nil {
				t.Errorf("Method %d returned unexpected error: %v", i, err)
			}
			if resp == nil {
				t.Errorf("Method %d returned nil response", i)
			}
		}
	})
}

// TestConvertQueryOptionToString tests the convertQueryOptionToString function
func TestConvertQueryOptionToString(t *testing.T) {
	tests := []struct {
		option   pb.QueryOption
		expected string
	}{
		{pb.QueryOption_QUERY_OPTION_HARDWARE, "hw"},
		{pb.QueryOption_QUERY_OPTION_FIRMWARE, "fw"},
		{pb.QueryOption_QUERY_OPTION_OS, "os"},
		{pb.QueryOption_QUERY_OPTION_SWBOM, "swbom"},
		{pb.QueryOption_QUERY_OPTION_VERSION, "version"},
		{pb.QueryOption_QUERY_OPTION_ALL, "all"},
		{pb.QueryOption_QUERY_OPTION_UNSPECIFIED, "all"},
	}

	for _, tt := range tests {
		result := convertQueryOptionToString(tt.option)
		if result != tt.expected {
			t.Errorf("convertQueryOptionToString(%v) = %v, want %v", tt.option, result, tt.expected)
		}
	}
}

func TestInbdServer_LoadConfig(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	t.Run("empty URI", func(t *testing.T) {
		req := &pb.LoadConfigRequest{Uri: ""}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("LoadConfig() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "uri is required") {
			t.Errorf("LoadConfig() Error = %v, want containing 'uri is required'", resp.Error)
		}
	})

	t.Run("invalid hash algorithm", func(t *testing.T) {
		req := &pb.LoadConfigRequest{
			Uri:           "file:///tmp/intel_manageability.conf",
			HashAlgorithm: "invalidalgo",
		}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		if resp.StatusCode != 400 {
			t.Errorf("LoadConfig() StatusCode = %v, want 400", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "invalid hash algorithm") {
			t.Errorf("LoadConfig() Error = %v, want containing 'invalid hash algorithm'", resp.Error)
		}
	})

	t.Run("valid request defaults to sha384", func(t *testing.T) {
		req := &pb.LoadConfigRequest{
			Uri: "file:///tmp/intel_manageability.conf",
			// No hash algorithm provided
		}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		// StatusCode may be 200 or 500 depending on file existence, but should not be 400
		if resp.StatusCode == 400 {
			t.Errorf("LoadConfig() StatusCode = %v, did not expect 400 for valid request", resp.StatusCode)
		}
	})

	t.Run("valid request with sha256", func(t *testing.T) {
		req := &pb.LoadConfigRequest{
			Uri:           "file:///tmp/intel_manageability.conf",
			HashAlgorithm: "sha256",
		}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		if resp.StatusCode == 400 {
			t.Errorf("LoadConfig() StatusCode = %v, did not expect 400 for valid request", resp.StatusCode)
		}
	})

	t.Run("valid request with sha512", func(t *testing.T) {
		req := &pb.LoadConfigRequest{
			Uri:           "file:///tmp/intel_manageability.conf",
			HashAlgorithm: "sha512",
		}
		resp, err := server.LoadConfig(ctx, req)
		if err != nil {
			t.Errorf("LoadConfig() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("LoadConfig() returned nil response")
		}
		if resp.StatusCode == 400 {
			t.Errorf("LoadConfig() StatusCode = %v, did not expect 400 for valid request", resp.StatusCode)
		}
	})
}
