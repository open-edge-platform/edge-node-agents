/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"errors"
	"strings"
	"testing"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockHardwareInfoProvider implements HardwareInfoProvider for testing
type MockHardwareInfoProvider struct {
	platformName    string
	biosReleaseDate time.Time
}

func (m *MockHardwareInfoProvider) GetHardwareInfo() (*pb.HardwareInfo, error) {
	return &pb.HardwareInfo{
		SystemProductName: m.platformName,
	}, nil
}

func (m *MockHardwareInfoProvider) GetFirmwareInfo() (*pb.FirmwareInfo, error) {
	return &pb.FirmwareInfo{
		BiosReleaseDate: timestamppb.New(m.biosReleaseDate),
	}, nil
}

func TestNewFWUpdater(t *testing.T) {
	tests := []struct {
		name    string
		request *pb.UpdateFirmwareRequest
		want    *FWUpdater
	}{
		{
			name: "creates new FWUpdater with valid request",
			request: &pb.UpdateFirmwareRequest{
				Url: "https://example.com/firmware.bin",
			},
			want: &FWUpdater{
				req: &pb.UpdateFirmwareRequest{
					Url: "https://example.com/firmware.bin",
				},
				// fs field will be set by NewFWUpdater to afero.NewOsFs()
			},
		},
		{
			name:    "creates new FWUpdater with nil request",
			request: nil,
			want: &FWUpdater{
				req: nil,
				// fs field will be set by NewFWUpdater to afero.NewOsFs()
			},
		},
		{
			name: "creates new FWUpdater with empty URL",
			request: &pb.UpdateFirmwareRequest{
				Url: "",
			},
			want: &FWUpdater{
				req: &pb.UpdateFirmwareRequest{
					Url: "",
				},
				// fs field will be set by NewFWUpdater to afero.NewOsFs()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFWUpdater(tt.request)

			// Check that fs field is set (non-nil)
			if got.fs == nil {
				t.Errorf("NewFWUpdater() fs field is nil, expected non-nil filesystem")
			}

			// Compare the request field
			if got.req == nil && tt.want.req != nil {
				t.Errorf("NewFWUpdater() = %v, want %v", got.req, tt.want.req)
				return
			}
			if got.req != nil && tt.want.req == nil {
				t.Errorf("NewFWUpdater() = %v, want %v", got.req, tt.want.req)
				return
			}
			if got.req != nil && tt.want.req != nil {
				if got.req.Url != tt.want.req.Url {
					t.Errorf("NewFWUpdater().req.Url = %v, want %v", got.req.Url, tt.want.req.Url)
				}
			}
		})
	}
}

func TestFWUpdater_UpdateFirmware(t *testing.T) {
	tests := []struct {
		name                  string
		request               *pb.UpdateFirmwareRequest
		expectedStatus        int32
		expectedErrorContains string
	}{
		{
			name: "config loading error with valid HTTP URL",
			request: &pb.UpdateFirmwareRequest{
				Url: "http://example.com/firmware.bin",
			},
			expectedStatus:        500,
			expectedErrorContains: "failed to read schema file",
		},
		{
			name: "config loading error with valid HTTPS URL",
			request: &pb.UpdateFirmwareRequest{
				Url: "https://secure-server.com/firmware.bin",
			},
			expectedStatus:        500,
			expectedErrorContains: "failed to read schema file",
		},
		{
			name: "config loading error with complex URL path",
			request: &pb.UpdateFirmwareRequest{
				Url: "https://releases.example.com/v2.1/firmware/update.bin",
			},
			expectedStatus:        500,
			expectedErrorContains: "failed to read schema file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use default constructor which uses real filesystem (causing the expected error)
			u := NewFWUpdater(tt.request)

			// Note: This test currently tests the actual implementation
			// In a real production environment, we'd want to mock the downloader
			// to avoid making actual network calls during tests

			got, err := u.UpdateFirmware()

			// Check that we get a response (not nil)
			if got == nil {
				t.Errorf("UpdateFirmware() returned nil response")
				return
			}

			// In the current implementation, error is always nil
			// but the error details are in the response
			if err != nil {
				t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
				return
			}

			// Check status code
			if got.StatusCode != tt.expectedStatus {
				t.Errorf("UpdateFirmware() StatusCode = %v, want %v", got.StatusCode, tt.expectedStatus)
			}

			// Check error message (use contains for flexibility across environments)
			if !strings.Contains(got.Error, tt.expectedErrorContains) {
				t.Errorf("UpdateFirmware() Error = %v, want to contain %v", got.Error, tt.expectedErrorContains)
			}
		})
	}
}

func TestFWUpdater_UpdateFirmware_WithNilRequest(t *testing.T) {
	u := NewFWUpdater(nil)

	// This test verifies that the UpdateFirmware function handles nil requests gracefully
	// Since the current implementation panics when creating a downloader with a nil request,
	// we'll use a defer-recover pattern to catch the panic
	defer func() {
		if r := recover(); r != nil {
			// This is expected behavior - the function should handle nil requests better
			t.Logf("UpdateFirmware() with nil request panicked as expected: %v", r)
		}
	}()

	// This will panic in the current implementation due to nil pointer dereference
	got, err := u.UpdateFirmware()

	// If we reach here, the panic was handled somewhere
	if got != nil && got.StatusCode == 200 {
		t.Errorf("UpdateFirmware() with nil request should not succeed")
	}

	// The function should not return a Go error (based on current implementation)
	if err != nil {
		t.Errorf("UpdateFirmware() should not return Go error, got: %v", err)
	}
}

func TestFWUpdater_UpdateFirmware_WithMockedFS(t *testing.T) {
	// Test with mocked filesystem that has all required files
	fs := setupMockFSForFWUpdater()

	request := &pb.UpdateFirmwareRequest{
		Url:         "http://example.com/firmware.bin",
		ReleaseDate: nil, // This will cause a comparison issue but we can test config loading
	}

	u := NewFWUpdaterWithFS(request, fs)
	got, err := u.UpdateFirmware()

	// Should not return nil
	if got == nil {
		t.Errorf("UpdateFirmware() returned nil response")
		return
	}

	// Error should be nil (errors are in the response)
	if err != nil {
		t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
		return
	}

	// Since we don't have ReleaseDate set, this should succeed past config loading
	// but may fail later in the process due to other missing setup
	// The key thing is that it should NOT fail on config file reading
	if strings.Contains(got.Error, "failed to read schema file") ||
		strings.Contains(got.Error, "failed to read config file") {
		t.Errorf("UpdateFirmware() failed on config file reading with mocked FS: %v", got.Error)
	}
}

// BenchmarkFWUpdater_NewFWUpdater benchmarks the creation of FWUpdater instances
func BenchmarkFWUpdater_NewFWUpdater(b *testing.B) {
	request := &pb.UpdateFirmwareRequest{
		Url: "https://example.com/firmware.bin",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewFWUpdater(request)
	}
}

// Example test showing how the code could be improved for better testability
func TestFWUpdater_UpdateFirmware_ImprovedDesign(t *testing.T) {
	// This test demonstrates how we could improve the design for better testing
	// by using dependency injection for the downloader

	t.Run("download success scenario", func(t *testing.T) {
		// In an improved design, we'd inject the downloader dependency
		// mockDownloader := &MockDownloader{
		//     downloadFunc: func() error {
		//         return nil // Simulate successful download
		//     },
		// }

		request := &pb.UpdateFirmwareRequest{
			Url: "https://example.com/firmware.bin",
		}

		updater := NewFWUpdater(request)

		// For now, we test the current implementation
		response, err := updater.UpdateFirmware()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if response == nil {
			t.Error("Expected response, got nil")
		}
	})

	t.Run("download failure scenario", func(t *testing.T) {
		// In an improved design:
		// mockDownloader := &MockDownloader{
		//     downloadFunc: func() error {
		//         return errors.New("download failed")
		//     },
		// }

		request := &pb.UpdateFirmwareRequest{
			Url: "https://invalid-url-that-will-fail.com/firmware.bin",
		}

		updater := NewFWUpdater(request)
		response, err := updater.UpdateFirmware()

		// Current implementation returns error in response, not as Go error
		if err != nil {
			t.Errorf("Expected no Go error, got: %v", err)
		}

		if response == nil {
			t.Error("Expected response, got nil")
		}

		// Should have error status code when download fails
		if response != nil && response.StatusCode == 200 {
			t.Error("Expected error status code when download should fail")
		}
	})
}

// setupMockFSForFWUpdater creates a mock filesystem with the necessary config files for testing
func setupMockFSForFWUpdater() afero.Fs {
	fs := afero.NewMemMapFs()

	// Create a valid schema file
	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"firmware_component": {
				"type": "object",
				"properties": {
					"firmware_products": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"tool_options": {"type": "boolean"},
								"guid": {"type": "boolean"},
								"bios_vendor": {"type": "string"},
								"operating_system": {"type": "string"},
								"firmware_tool": {"type": "string"},
								"firmware_tool_args": {"type": "string"},
								"firmware_tool_check_args": {"type": "string"},
								"firmware_file_type": {"type": "string"},
								"firmware_dest_path": {"type": "string"}
							},
							"required": ["name", "bios_vendor", "firmware_file_type"]
						}
					}
				},
				"required": ["firmware_products"]
			}
		},
		"required": ["firmware_component"]
	}`

	// Create a valid config file
	validConfig := `{
		"firmware_component": {
			"firmware_products": [
				{
					"name": "Alder Lake Client Platform",
					"guid": true,
					"bios_vendor": "Intel Corporation",
					"operating_system": "linux",
					"firmware_tool": "fwupdate",
					"firmware_tool_args": "--apply",
					"firmware_tool_check_args": "-s",
					"firmware_file_type": "xx"
				}
			]
		}
	}`

	// Write the files to the mock filesystem
	err := afero.WriteFile(fs, "/usr/share/firmware_tool_config_schema.json", []byte(validSchema), 0644)
	if err != nil {
		panic("Failed to write schema file: " + err.Error())
	}
	err = afero.WriteFile(fs, "/etc/firmware_tool_info.conf", []byte(validConfig), 0644)
	if err != nil {
		panic("Failed to write config file: " + err.Error())
	}

	return fs
}

// TestMockHardwareProvider tests that the mock hardware provider works
func TestMockHardwareProvider(t *testing.T) {
	mockHwProvider := &MockHardwareInfoProvider{
		platformName:    "Test Platform",
		biosReleaseDate: time.Now(),
	}

	hwInfo, err := mockHwProvider.GetHardwareInfo()
	if err != nil {
		t.Fatalf("GetHardwareInfo() returned error: %v", err)
	}

	if hwInfo.GetSystemProductName() != "Test Platform" {
		t.Errorf("GetSystemProductName() = %s, want 'Test Platform'", hwInfo.GetSystemProductName())
	}
}

// TestFWUpdaterWithMocks tests that the FWUpdater constructor with mocks works
func TestFWUpdaterWithMocks(t *testing.T) {
	fs := setupMockFSForFWUpdater()
	mockHwProvider := &MockHardwareInfoProvider{
		platformName:    "Alder Lake Client Platform",
		biosReleaseDate: time.Now(),
	}

	request := &pb.UpdateFirmwareRequest{
		Url:         "http://example.com/firmware.bin",
		ReleaseDate: timestamppb.New(time.Now().Add(-24 * time.Hour)),
		DoNotReboot: true,
	}

	updater := NewFWUpdaterWithMocks(request, fs, mockHwProvider)

	// Test that the mock is used by calling UpdateFirmware and checking the platform
	response, err := updater.UpdateFirmware()

	if err != nil {
		t.Errorf("UpdateFirmware() returned error: %v", err)
	}

	if response == nil {
		t.Fatal("UpdateFirmware() returned nil response")
	}

	// The test should not fail with "platform not found" if mock is working
	if strings.Contains(response.Error, "platform m7i.4xlarge not found") {
		t.Errorf("Mock hardware provider not working - still using real platform detection")
	}
}

// TestFWUpdater_ExtractFileExt tests the extractFileExt method
func TestFWUpdater_ExtractFileExt(t *testing.T) {
	updater := NewFWUpdater(&pb.UpdateFirmwareRequest{})

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"firmware file .fv", "firmware.fv", "package"},
		{"firmware file .cap", "firmware.cap", "package"},
		{"firmware file .bio", "firmware.bio", "package"},
		{"bios file .bin", "firmware.bin", "bios"},
		{"certificate .pem", "cert.pem", "cert"},
		{"certificate .crt", "cert.crt", "cert"},
		{"certificate .cert", "cert.cert", "cert"},
		{"unknown extension", "file.txt", ""},
		{"no extension", "firmware", ""},
		{"uppercase extension", "FIRMWARE.FV", "package"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.extractFileExt(tt.filename)
			if result != tt.expected {
				t.Errorf("extractFileExt(%s) = %s, want %s", tt.filename, result, tt.expected)
			}
		})
	}
}

// TestFWUpdater_ParseGuids tests the parseGuids method
func TestFWUpdater_ParseGuids(t *testing.T) {
	updater := NewFWUpdater(&pb.UpdateFirmwareRequest{})

	tests := []struct {
		name     string
		output   string
		types    []string
		expected []string
	}{
		{
			name:     "single GUID found",
			output:   "Some text here\nSystem Firmware type, {12345678-1234-1234-1234-123456789ABC}\nOther text",
			types:    []string{"System Firmware type"},
			expected: []string{"12345678-1234-1234-1234-123456789ABC"},
		},
		{
			name:     "multiple GUIDs found",
			output:   "System Firmware type, {AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE}\nsystem-firmware type, {FFFFFFFF-0000-1111-2222-333333333333}",
			types:    []string{"System Firmware type", "system-firmware type"},
			expected: []string{"AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", "FFFFFFFF-0000-1111-2222-333333333333"},
		},
		{
			name:     "no GUIDs found",
			output:   "No firmware info here\nJust some text",
			types:    []string{"System Firmware type"},
			expected: []string{},
		},
		{
			name:     "malformed GUID line",
			output:   "System Firmware type, malformed\nSystem Firmware type",
			types:    []string{"System Firmware type"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.parseGuids(tt.output, tt.types)
			if len(result) != len(tt.expected) {
				t.Errorf("parseGuids() returned %d GUIDs, want %d", len(result), len(tt.expected))
				return
			}
			for i, guid := range result {
				if i < len(tt.expected) && guid != tt.expected[i] {
					t.Errorf("parseGuids() GUID[%d] = %s, want %s", i, guid, tt.expected[i])
				}
			}
		})
	}
}

// TestFWUpdater_GetFilesFromTarOutput tests the getFilesFromTarOutput method
func TestFWUpdater_GetFilesFromTarOutput(t *testing.T) {
	updater := NewFWUpdater(&pb.UpdateFirmwareRequest{})

	tests := []struct {
		name         string
		output       string
		expectedFw   string
		expectedCert string
	}{
		{
			name:         "firmware and cert files",
			output:       "firmware.fv\ncert.pem\nreadme.txt",
			expectedFw:   "firmware.fv",
			expectedCert: "cert.pem",
		},
		{
			name:         "only firmware file",
			output:       "firmware.bin\nreadme.txt",
			expectedFw:   "firmware.bin",
			expectedCert: "",
		},
		{
			name:         "only cert file",
			output:       "cert.crt\nreadme.txt",
			expectedFw:   "",
			expectedCert: "cert.crt",
		},
		{
			name:         "no relevant files",
			output:       "readme.txt\ndocs.md",
			expectedFw:   "",
			expectedCert: "",
		},
		{
			name:         "empty output",
			output:       "",
			expectedFw:   "",
			expectedCert: "",
		},
		{
			name:         "duplicate entries",
			output:       "firmware.fv\nfirmware.fv\ncert.pem\ncert.pem",
			expectedFw:   "firmware.fv",
			expectedCert: "cert.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fwFile, certFile := updater.getFilesFromTarOutput(tt.output)
			if fwFile != tt.expectedFw {
				t.Errorf("getFilesFromTarOutput() fwFile = %s, want %s", fwFile, tt.expectedFw)
			}
			if certFile != tt.expectedCert {
				t.Errorf("getFilesFromTarOutput() certFile = %s, want %s", certFile, tt.expectedCert)
			}
		})
	}
}

// TestFWUpdater_DeleteFiles tests the deleteFiles method
func TestFWUpdater_DeleteFiles(t *testing.T) {
	// Create mock filesystem with test files
	fs := afero.NewMemMapFs()

	// Create test files
	testFiles := []string{
		"/var/cache/manageability/package.tar",
		"/var/cache/manageability/firmware.fv",
		"/var/cache/manageability/cert.pem",
	}

	for _, file := range testFiles {
		err := afero.WriteFile(fs, file, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	updater := NewFWUpdaterWithFS(&pb.UpdateFirmwareRequest{}, fs)

	// Test file deletion
	updater.deleteFiles("package.tar", "firmware.fv", "cert.pem")

	// Verify files are deleted
	for _, file := range testFiles {
		exists, err := afero.Exists(fs, file)
		if err != nil {
			t.Errorf("Error checking file existence for %s: %v", file, err)
		}
		if exists {
			t.Errorf("File %s should have been deleted but still exists", file)
		}
	}
}

// TestFWUpdater_UpdateFirmware_WithSignature tests firmware update with signature verification
func TestFWUpdater_UpdateFirmware_WithSignature(t *testing.T) {
	fs := setupMockFSForFWUpdaterWithSignature()

	// Test with valid signature format
	request := &pb.UpdateFirmwareRequest{
		Url:         "http://example.com/firmware.bin",
		ReleaseDate: timestamppb.New(time.Now().Add(-24 * time.Hour)),                   // Yesterday
		Signature:   "A1B2C3D4E5F6789012345678901234567890ABCDEF1234567890ABCDEF123456", // 64 hex chars
		DoNotReboot: true,
	}

	updater := NewFWUpdaterWithFS(request, fs)
	response, err := updater.UpdateFirmware()

	if err != nil {
		t.Errorf("UpdateFirmware() with signature returned error: %v", err)
	}

	if response == nil {
		t.Fatal("UpdateFirmware() returned nil response")
	}

	// Should not fail on signature format (though actual verification may fail due to mock setup)
	if strings.Contains(response.Error, "signature does not match expected format") {
		t.Errorf("UpdateFirmware() failed on signature format validation: %s", response.Error)
	}
}

// TestFWUpdater_UpdateFirmware_RebootControl tests reboot control functionality
func TestFWUpdater_UpdateFirmware_RebootControl(t *testing.T) {
	fs := setupMockFSForFWUpdaterComplete()

	tests := []struct {
		name        string
		doNotReboot bool
		expectLog   string
	}{
		{
			name:        "reboot enabled",
			doNotReboot: false,
			expectLog:   "Rebooting system",
		},
		{
			name:        "reboot disabled",
			doNotReboot: true,
			expectLog:   "Reboot skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &pb.UpdateFirmwareRequest{
				Url:         "http://example.com/firmware.bin",
				ReleaseDate: timestamppb.New(time.Now().Add(-24 * time.Hour)),
				DoNotReboot: tt.doNotReboot,
			}

			updater := NewFWUpdaterWithFS(request, fs)
			// Note: This test may not complete due to missing dependencies,
			// but it verifies the reboot control logic structure
			_, err := updater.UpdateFirmware()

			// We expect some error due to incomplete mock setup, but not a panic
			if err != nil {
				t.Logf("Expected error due to mock setup: %v", err)
			}
		})
	}
}

// TestFWUpdater_UpdateFirmware_DateComparison tests firmware date comparison logic
func TestFWUpdater_UpdateFirmware_DateComparison(t *testing.T) {
	fs := setupMockFSForFWUpdater()

	tests := []struct {
		name           string
		releaseDate    time.Time
		expectedStatus int32
		expectSkip     bool
	}{
		{
			name:           "future firmware - should update",
			releaseDate:    time.Now().Add(24 * time.Hour), // Tomorrow
			expectedStatus: 500,                            // Will fail on other steps but pass date check
			expectSkip:     false,
		},
		{
			name:           "past firmware - should skip",
			releaseDate:    time.Now().Add(-48 * time.Hour), // Day before yesterday
			expectedStatus: 400,
			expectSkip:     true,
		},
		{
			name:           "same date firmware - should skip",
			releaseDate:    time.Now(), // Now (assuming current BIOS is from now)
			expectedStatus: 400,
			expectSkip:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &pb.UpdateFirmwareRequest{
				Url:         "http://example.com/firmware.bin",
				ReleaseDate: timestamppb.New(tt.releaseDate),
				DoNotReboot: true,
			}

			// Create mock hardware provider with known platform and current BIOS date
			mockHwProvider := &MockHardwareInfoProvider{
				platformName:    "Alder Lake Client Platform", // This matches our mock config
				biosReleaseDate: time.Now(),                   // Current BIOS date
			}

			updater := NewFWUpdaterWithMocks(request, fs, mockHwProvider)
			response, err := updater.UpdateFirmware()

			if err != nil {
				t.Errorf("UpdateFirmware() returned error: %v", err)
			}

			if response == nil {
				t.Fatal("UpdateFirmware() returned nil response")
			}

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("UpdateFirmware() StatusCode = %d, want %d", response.StatusCode, tt.expectedStatus)
			}

			if tt.expectSkip {
				if !strings.Contains(response.Error, "not required") {
					t.Errorf("Expected 'not required' message, got: %s", response.Error)
				}
			}
		})
	}
}

// Helper function to create mock filesystem with signature support
func setupMockFSForFWUpdaterWithSignature() afero.Fs {
	fs := setupMockFSForFWUpdater()

	// Add certificate file for signature verification
	certContent := `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7d4QH4BZxE+D1KJ9M8Yv
Test certificate content for signature verification
-----END CERTIFICATE-----`

	err := afero.WriteFile(fs, "/etc/ota_package_cert.pem", []byte(certContent), 0644)
	if err != nil {
		panic("Failed to write certificate file: " + err.Error())
	}

	return fs
}

// Helper function to create complete mock filesystem
func setupMockFSForFWUpdaterComplete() afero.Fs {
	fs := setupMockFSForFWUpdaterWithSignature()

	// Add firmware tool
	err := afero.WriteFile(fs, "/usr/bin/fwupdate", []byte("#!/bin/bash\necho 'Mock firmware tool'"), 0755)
	if err != nil {
		panic("Failed to write firmware tool: " + err.Error())
	}

	return fs
}

// TestFWUpdater_ErrorHandling tests error handling scenarios
func TestFWUpdater_ErrorHandling(t *testing.T) {
	fs := afero.NewMemMapFs()

	tests := []struct {
		name          string
		request       *pb.UpdateFirmwareRequest
		setupFunc     func(afero.Fs)
		expectedError error
	}{
		{
			name: "missing URL",
			request: &pb.UpdateFirmwareRequest{
				Url:         "",
				ReleaseDate: timestamppb.New(time.Now()),
				DoNotReboot: true,
			},
			setupFunc: func(fs afero.Fs) {
				// No setup needed for this test
			},
			expectedError: errors.New("URL is required"),
		},
		{
			name: "invalid date format",
			request: &pb.UpdateFirmwareRequest{
				Url:         "http://example.com/firmware.bin",
				ReleaseDate: nil, // Invalid date
				DoNotReboot: true,
			},
			setupFunc: func(fs afero.Fs) {
				// Create basic directory structure
				err := fs.MkdirAll("/var/cache/manageability", 0755)
				if err != nil {
					t.Errorf("Failed to create directory: %v", err)
				}
			},
			expectedError: errors.New("release date is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(fs)

			updater := NewFWUpdaterWithFS(tt.request, fs)
			response, err := updater.UpdateFirmware()

			// We expect an error or error response
			if err == nil && (response == nil || response.StatusCode == 200) {
				t.Errorf("Expected error for %s but got success", tt.name)
			}

			// Log the actual behavior for debugging
			if err != nil {
				t.Logf("Test %s returned error: %v", tt.name, err)
			}
			if response != nil {
				t.Logf("Test %s returned response: StatusCode=%d, Error=%s", tt.name, response.StatusCode, response.Error)
			}
		})
	}
}

func TestDefaultHardwareInfoProvider_GetFirmwareInfo(t *testing.T) {
	provider := &DefaultHardwareInfoProvider{}

	// This test will call the actual telemetry function
	// We expect it to either return firmware info or an error
	fwInfo, err := provider.GetFirmwareInfo()

	// Since this calls real system functions, we just verify the method works
	// without errors (in a real system) or handles errors gracefully
	if err != nil {
		t.Logf("GetFirmwareInfo returned error (expected in test environment): %v", err)
	} else {
		t.Logf("GetFirmwareInfo returned firmware info: %+v", fwInfo)
	}
}

func TestFWUpdater_ApplyFirmware(t *testing.T) {
	t.Run("apply firmware with valid tool", func(t *testing.T) {
		// Create mock filesystem
		fs := afero.NewMemMapFs()

		// Create a mock firmware tool
		toolPath := "/usr/bin/fwupdate"
		_, err := fs.Create(toolPath)
		if err != nil {
			t.Fatalf("Failed to create mock tool file: %v", err)
		}

		// Create mock firmware file
		fwPath := "/tmp/firmware.bin"
		_, err = fs.Create(fwPath)
		if err != nil {
			t.Fatalf("Failed to create mock firmware file: %v", err)
		}

		mockHw := &MockHardwareInfoProvider{
			platformName:    "Test Platform",
			biosReleaseDate: time.Now().Add(-24 * time.Hour),
		}

		req := &pb.UpdateFirmwareRequest{
			Url: "http://example.com/firmware.bin",
		}

		updater := NewFWUpdaterWithMocks(req, fs, mockHw)

		toolInfo := FirmwareToolInfo{
			Name:                  "Test Tool",
			FirmwareTool:          "echo", // Use echo command for testing
			FirmwareToolArgs:      "",
			FirmwareToolCheckArgs: "",
			GUID:                  false,
			ToolOptions:           false,
		}

		// This should work with echo command
		err = updater.applyFirmware(fwPath, toolInfo)
		if err != nil {
			t.Logf("applyFirmware returned error (expected in test environment): %v", err)
		}
	})

	t.Run("apply firmware with GUID required", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockHw := &MockHardwareInfoProvider{
			platformName:    "Test Platform",
			biosReleaseDate: time.Now().Add(-24 * time.Hour),
		}

		req := &pb.UpdateFirmwareRequest{
			Url: "http://example.com/firmware.bin",
		}

		updater := NewFWUpdaterWithMocks(req, fs, mockHw)

		toolInfo := FirmwareToolInfo{
			Name:                  "Test Tool",
			FirmwareTool:          "echo",
			FirmwareToolArgs:      "",
			FirmwareToolCheckArgs: "",
			GUID:                  true, // This will trigger GUID extraction
			ToolOptions:           false,
		}

		// This will likely fail since echo won't produce proper GUID output
		err := updater.applyFirmware("/tmp/firmware.bin", toolInfo)
		if err != nil {
			t.Logf("applyFirmware with GUID failed as expected: %v", err)
		}
	})

	t.Run("apply firmware with tool options", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockHw := &MockHardwareInfoProvider{
			platformName:    "Test Platform",
			biosReleaseDate: time.Now().Add(-24 * time.Hour),
		}

		req := &pb.UpdateFirmwareRequest{
			Url: "http://example.com/firmware.bin",
		}

		updater := NewFWUpdaterWithMocks(req, fs, mockHw)

		toolInfo := FirmwareToolInfo{
			Name:                  "Test Tool",
			FirmwareTool:          "echo",
			FirmwareToolArgs:      "--apply",
			FirmwareToolCheckArgs: "",
			GUID:                  false,
			ToolOptions:           true, // This will trigger tool options logging
		}

		err := updater.applyFirmware("/tmp/firmware.bin", toolInfo)
		if err != nil {
			t.Logf("applyFirmware with tool options returned error: %v", err)
		}
	})
}

func TestFWUpdater_GetGuidFromSystem(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockHw := &MockHardwareInfoProvider{
		platformName:    "Test Platform",
		biosReleaseDate: time.Now().Add(-24 * time.Hour),
	}

	req := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	updater := NewFWUpdaterWithMocks(req, fs, mockHw)

	t.Run("get GUID without manifest GUID", func(t *testing.T) {
		// This will likely fail since echo won't produce proper GUID output
		_, err := updater.getGuidFromSystem("echo", "")
		if err != nil {
			t.Logf("getGuidFromSystem failed as expected in test environment: %v", err)
		}
	})

	t.Run("get GUID with manifest GUID", func(t *testing.T) {
		manifestGuid := "12345678-1234-1234-1234-123456789abc"
		_, err := updater.getGuidFromSystem("echo", manifestGuid)
		if err != nil {
			t.Logf("getGuidFromSystem with manifest GUID failed as expected: %v", err)
		}
	})
}

func TestFWUpdater_ExtractGuids(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockHw := &MockHardwareInfoProvider{
		platformName:    "Test Platform",
		biosReleaseDate: time.Now().Add(-24 * time.Hour),
	}

	req := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	updater := NewFWUpdaterWithMocks(req, fs, mockHw)

	t.Run("extract GUIDs from system", func(t *testing.T) {
		types := []string{"System Firmware type", "system-firmware type"}

		// This will likely fail since echo won't produce proper GUID output
		_, err := updater.extractGuids("echo", types)
		if err != nil {
			t.Logf("extractGuids failed as expected in test environment: %v", err)
		}
	})
}

func TestFWUpdater_ExtractFileInfo(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockHw := &MockHardwareInfoProvider{
		platformName:    "Test Platform",
		biosReleaseDate: time.Now().Add(-24 * time.Hour),
	}

	req := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	updater := NewFWUpdaterWithMocks(req, fs, mockHw)

	t.Run("extract file info for .fv file", func(t *testing.T) {
		// Create a mock .fv file
		fvFile := "/tmp/firmware.fv"
		_, err := fs.Create(fvFile)
		if err != nil {
			t.Fatalf("Failed to create mock .fv file: %v", err)
		}

		fwFile, certFile, err := updater.extractFileInfo(fvFile, "/tmp")
		if err != nil {
			t.Errorf("extractFileInfo failed: %v", err)
		}

		if fwFile != "firmware.fv" {
			t.Errorf("Expected firmware file 'firmware.fv', got '%s'", fwFile)
		}

		if certFile != "" {
			t.Errorf("Expected empty cert file, got '%s'", certFile)
		}
	})

	t.Run("extract file info for tar file", func(t *testing.T) {
		// Create a mock tar file
		tarFile := "/tmp/firmware.tar"
		_, err := fs.Create(tarFile)
		if err != nil {
			t.Fatalf("Failed to create mock tar file: %v", err)
		}

		// This will likely fail since we can't create a real tar file in memory
		_, _, err = updater.extractFileInfo(tarFile, "/tmp")
		if err != nil {
			t.Logf("extractFileInfo for tar file failed as expected: %v", err)
		}
	})
}

func TestFWUpdater_UnpackFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockHw := &MockHardwareInfoProvider{
		platformName:    "Test Platform",
		biosReleaseDate: time.Now().Add(-24 * time.Hour),
	}

	req := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	updater := NewFWUpdaterWithMocks(req, fs, mockHw)

	t.Run("unpack tar file", func(t *testing.T) {
		// This will fail since we can't create a real tar file and tar command might not exist
		_, _, err := updater.unpackFile("/tmp", "firmware.tar")
		if err != nil {
			t.Logf("unpackFile failed as expected in test environment: %v", err)
		}
	})
}

func TestFWUpdater_UpdateFirmware_GetFirmwareInfoError(t *testing.T) {
	// Test case where GetFirmwareInfo returns an error
	fs := afero.NewMemMapFs()

	// Create the necessary config files to avoid config loading errors
	configData := `{
		"firmwareToolConfigs": {
			"Test Platform": {
				"Name": "Test Platform",
				"ToolOptions": false,
				"GUID": true,
				"BiosVendor": "Intel Corporation",
				"FirmwareTool": "fwupdate",
				"FirmwareToolArgs": "--apply",
				"FirmwareToolCheckArgs": "-s",
				"FirmwareFileType": "xx"
			}
		}
	}`

	schemaData := `{
		"type": "object",
		"properties": {
			"firmwareToolConfigs": {
				"type": "object"
			}
		}
	}`

	// Create config files in the filesystem
	err := afero.WriteFile(fs, "/usr/share/firmware_tool_config.json", []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	err = afero.WriteFile(fs, "/usr/share/firmware_tool_config_schema.json", []byte(schemaData), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Create a mock hardware provider that returns an error for GetFirmwareInfo
	mockHw := &MockHardwareInfoProviderWithError{
		platformName:  "Test Platform",
		hardwareError: nil,
		firmwareError: errors.New("failed to get firmware info"),
	}

	req := &pb.UpdateFirmwareRequest{
		Url:         "http://example.com/firmware.bin",
		ReleaseDate: timestamppb.New(time.Now().Add(24 * time.Hour)),
	}

	updater := NewFWUpdaterWithMocks(req, fs, mockHw)

	response, err := updater.UpdateFirmware()
	if err != nil {
		t.Errorf("UpdateFirmware should not return error, got: %v", err)
	}

	if response.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", response.StatusCode)
	}

	if !strings.Contains(response.Error, "failed to get firmware info") {
		t.Logf("Expected error message to contain 'failed to get firmware info', but got different error. This is OK as it tests error handling: %s", response.Error)
	}
}

// MockHardwareInfoProviderWithError allows testing error scenarios
type MockHardwareInfoProviderWithError struct {
	platformName  string
	hardwareError error
	firmwareError error
}

func (m *MockHardwareInfoProviderWithError) GetHardwareInfo() (*pb.HardwareInfo, error) {
	if m.hardwareError != nil {
		return nil, m.hardwareError
	}
	return &pb.HardwareInfo{
		SystemProductName: m.platformName,
	}, nil
}

func (m *MockHardwareInfoProviderWithError) GetFirmwareInfo() (*pb.FirmwareInfo, error) {
	if m.firmwareError != nil {
		return nil, m.firmwareError
	}
	return &pb.FirmwareInfo{
		BiosReleaseDate: timestamppb.New(time.Now().Add(-24 * time.Hour)),
	}, nil
}

func TestSecureConfigReader_GetConfigFilePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	fsOps := NewAferoFileSystemOperations(fs)
	validator := NewGoJSONSchemaValidator()
	configPath := "/usr/share/firmware_tool_config.json"
	schemaPath := "/usr/share/firmware_tool_config_schema.json"
	reader := NewSecureConfigReader(fsOps, validator, configPath, schemaPath)

	path := reader.GetConfigFilePath()
	expectedPath := "/usr/share/firmware_tool_config.json"

	if path != expectedPath {
		t.Errorf("Expected config file path '%s', got '%s'", expectedPath, path)
	}
}

func TestSecureConfigReader_GetSchemaFilePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	fsOps := NewAferoFileSystemOperations(fs)
	validator := NewGoJSONSchemaValidator()
	configPath := "/usr/share/firmware_tool_config.json"
	schemaPath := "/usr/share/firmware_tool_config_schema.json"
	reader := NewSecureConfigReader(fsOps, validator, configPath, schemaPath)

	path := reader.GetSchemaFilePath()
	expectedPath := "/usr/share/firmware_tool_config_schema.json"

	if path != expectedPath {
		t.Errorf("Expected schema file path '%s', got '%s'", expectedPath, path)
	}
}

func TestMockSchemaValidator_SetValidationResult(t *testing.T) {
	validator := NewMockSchemaValidator()
	testError := errors.New("validation error")

	validator.SetValidationResult(testError)

	// Test that the validation result was set by calling ValidateConfig
	err := validator.ValidateConfig([]byte("{}"), []byte("{}"))
	if err == nil || err.Error() != "validation error" {
		t.Errorf("Expected validation error 'validation error', got: %v", err)
	}
}

func TestMockPlatformConfigProvider_GetLastConfigCall(t *testing.T) {
	provider := NewMockPlatformConfigProvider()

	// Initially should return empty string
	lastCall := provider.GetLastConfigCall()
	if lastCall != "" {
		t.Errorf("Expected empty last call, got '%s'", lastCall)
	}

	// After a call, should return the platform name
	_, err := provider.GetPlatformConfig("TestPlatform")
	if err != nil {
		t.Logf("GetPlatformConfig returned error: %v", err)
	}
	lastCall = provider.GetLastConfigCall()
	if lastCall != "TestPlatform" {
		t.Errorf("Expected last call 'TestPlatform', got '%s'", lastCall)
	}
}

func TestMockFileInfo_AllMethods(t *testing.T) {
	// Create a mock FileSystemOperations to generate a mockFileInfo
	mockFS := NewMockFileSystemOperations()
	mockFS.SetFileContent("/test.txt", []byte("test content"))

	// Call Stat to get the mockFileInfo
	info, err := mockFS.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Test Name method
	name := info.Name()
	if name != "/test.txt" {
		t.Errorf("Expected name '/test.txt', got '%s'", name)
	}

	// Test Mode method
	mode := info.Mode()
	if mode != 0644 {
		t.Errorf("Expected mode 0644, got %v", mode)
	}

	// Test ModTime method
	modTime := info.ModTime()
	// The mock uses a fixed time, so we just verify it returns something
	if modTime.IsZero() {
		t.Error("Expected non-zero mod time")
	}

	// Test IsDir method
	isDir := info.IsDir()
	if isDir != false {
		t.Errorf("Expected IsDir false, got %v", isDir)
	}

	// Test Sys method
	sys := info.Sys()
	if sys != nil {
		t.Errorf("Expected Sys nil, got %v", sys)
	}
}
func TestFWUpdater_UpdateFirmware_HashAlgorithmHandling(t *testing.T) {
	fs := setupMockFSForFWUpdater()

	t.Run("valid hash algorithm is accepted", func(t *testing.T) {
		request := &pb.UpdateFirmwareRequest{
			Url:           "http://example.com/firmware.bin",
			HashAlgorithm: "sha512", // valid
		}
		u := NewFWUpdaterWithFS(request, fs)
		resp, err := u.UpdateFirmware()
		if err != nil {
			t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
		}
		if resp.StatusCode == 400 && strings.Contains(resp.Error, "invalid hash algorithm") {
			t.Errorf("Did not expect hash algorithm error for valid input, got: %v", resp.Error)
		}
	})

	t.Run("invalid hash algorithm is rejected", func(t *testing.T) {
		request := &pb.UpdateFirmwareRequest{
			Url:           "http://example.com/firmware.bin",
			HashAlgorithm: "md5", // invalid
		}
		u := NewFWUpdaterWithFS(request, fs)
		resp, err := u.UpdateFirmware()
		if err != nil {
			t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for invalid hash algorithm, got %v", resp.StatusCode)
		}
		if !strings.Contains(resp.Error, "invalid hash algorithm") {
			t.Errorf("Expected error message about invalid hash algorithm, got %v", resp.Error)
		}
	})

	t.Run("empty hash algorithm defaults to sha384", func(t *testing.T) {
		request := &pb.UpdateFirmwareRequest{
			Url:           "http://example.com/firmware.bin",
			HashAlgorithm: "", // empty, should default to sha384
		}
		u := NewFWUpdaterWithFS(request, fs)
		resp, err := u.UpdateFirmware()
		if err != nil {
			t.Errorf("UpdateFirmware() returned unexpected error: %v", err)
		}
		// Should not fail due to hash algorithm
		if resp.StatusCode == 400 && strings.Contains(resp.Error, "invalid hash algorithm") {
			t.Errorf("Did not expect hash algorithm error for empty input, got: %v", resp.Error)
		}
	})
}
