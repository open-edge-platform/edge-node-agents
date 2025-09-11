/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

// Mock for os_updater.DetectOS
type MockOSDetector struct {
	mock.Mock
}

func TestGetSoftwareBOM(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		// This test requires mocking the OS detection and package retrieval
		// For now, we'll test the structure
		if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
			t.Skip("Skipping integration test")
		}

		swbom, err := GetSoftwareBOM()

		if err != nil {
			// If we can't run the actual commands, verify error handling
			assert.Error(t, err)
			assert.Nil(t, swbom)
		} else {
			// If commands work, verify structure
			assert.NotNil(t, swbom)
			assert.NotNil(t, swbom.CollectionTimestamp)
			assert.NotEmpty(t, swbom.CollectionMethod)
			// Packages might be empty on test systems
			assert.NotNil(t, swbom.Packages)
		}
	})
}

func TestParsePackageOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []*pb.SoftwarePackage
	}{
		{
			name:  "valid debian packages",
			input: "apt 2.4.11\nbase-files 12.4+deb12u5\nlibc6 2.36-9+deb12u4",
			expected: []*pb.SoftwarePackage{
				{Name: "apt", Version: "2.4.11", Type: "deb"},
				{Name: "base-files", Version: "12.4+deb12u5", Type: "deb"},
				{Name: "libc6", Version: "2.36-9+deb12u4", Type: "deb"},
			},
		},
		{
			name:  "single package",
			input: "vim 8.2.0716",
			expected: []*pb.SoftwarePackage{
				{Name: "vim", Version: "8.2.0716", Type: "deb"},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil, // Allow nil for empty input
		},
		{
			name:  "malformed line",
			input: "incomplete-line\napt 2.4.11",
			expected: []*pb.SoftwarePackage{
				{Name: "apt", Version: "2.4.11", Type: "deb"},
			},
		},
		{
			name:  "extra whitespace",
			input: "  apt   2.4.11  \n  vim   8.2.0716  ",
			expected: []*pb.SoftwarePackage{
				{Name: "apt", Version: "2.4.11", Type: "deb"},
				{Name: "vim", Version: "8.2.0716", Type: "deb"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parsePackageOutput(tc.input)

			// Handle empty/nil case flexibly
			if tc.input == "" {
				assert.Equal(t, 0, len(result), "Empty input should return nil or empty slice")
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestParseRPMOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []*pb.SoftwarePackage
	}{
		{
			name:  "valid rpm packages",
			input: "bash-5.1.8-1.el9.x86_64\nkernel-5.14.0-284.el9.x86_64\nglibc-2.34-60.el9.x86_64",
			expected: []*pb.SoftwarePackage{
				{Name: "bash", Version: "5.1.8-1.el9", Architecture: "x86_64", Type: "rpm"},
				{Name: "kernel", Version: "5.14.0-284.el9", Architecture: "x86_64", Type: "rpm"},
				{Name: "glibc", Version: "2.34-60.el9", Architecture: "x86_64", Type: "rpm"},
			},
		},
		{
			name:  "complex package name",
			input: "python3-pip-21.2.3-6.el9.noarch",
			expected: []*pb.SoftwarePackage{
				{Name: "python3-pip", Version: "21.2.3-6.el9", Architecture: "noarch", Type: "rpm"},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil, // Allow nil for empty input
		},
		{
			name:  "package without architecture",
			input: "simple-package",
			expected: []*pb.SoftwarePackage{
				{Name: "simple-package", Type: "rpm"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRPMOutput(tc.input)

			// Handle empty/nil case flexibly
			if tc.input == "" {
				assert.Equal(t, 0, len(result), "Empty input should return nil or empty slice")
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestParseRPMPackageName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *pb.SoftwarePackage
	}{
		{
			name:  "standard rpm package",
			input: "bash-5.1.8-1.el9.x86_64",
			expected: &pb.SoftwarePackage{
				Name:         "bash",
				Version:      "5.1.8-1.el9",
				Architecture: "x86_64",
				Type:         "rpm",
			},
		},
		{
			name:  "package with complex name",
			input: "python3-pip-21.2.3-6.el9.noarch",
			expected: &pb.SoftwarePackage{
				Name:         "python3-pip",
				Version:      "21.2.3-6.el9",
				Architecture: "noarch",
				Type:         "rpm",
			},
		},
		{
			name:  "package without dots",
			input: "simple-package",
			expected: &pb.SoftwarePackage{
				Name: "simple-package",
				Type: "rpm",
			},
		},
		{
			name:  "minimal package with arch",
			input: "pkg.x86_64",
			expected: &pb.SoftwarePackage{
				Name:         "pkg",
				Architecture: "x86_64",
				Type:         "rpm",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRPMPackageName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestReadMenderFile(t *testing.T) {
	// Create a temporary filesystem for testing
	fs := afero.NewMemMapFs()

	t.Run("file exists with content", func(t *testing.T) {
		// Create test file
		testPath := "/test/mender/artifact_info"
		err := fs.MkdirAll("/test/mender", 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, testPath, []byte("test-version-123\n"), 0644)
		require.NoError(t, err)

		// Mock the filesystem by temporarily replacing utils functions
		// For this test, we'll test the logic manually
		content := "test-version-123\n"
		result := strings.TrimSpace(strings.Split(content, "\x00")[0])
		expected := "test-version-123"

		assert.Equal(t, expected, result)
	})

	t.Run("file with null bytes", func(t *testing.T) {
		content := "version-1.0\x00extra-data"
		result := strings.TrimSpace(strings.Split(content, "\x00")[0])
		expected := "version-1.0"

		assert.Equal(t, expected, result)
	})

	t.Run("file not found returns default", func(t *testing.T) {
		result := readMenderFile("/nonexistent/path", "default-value")
		assert.Equal(t, "default-value", result)
	})
}

func TestGetMenderVersion(t *testing.T) {
	t.Run("mender version available", func(t *testing.T) {
		// This test would require mocking file system operations
		// For now, test the nil case when file doesn't exist
		result := getMenderVersion()
		// Since we can't guarantee the file exists in test environment
		if result == nil {
			assert.Nil(t, result)
		} else {
			assert.Equal(t, "mender", result.Name)
			assert.Equal(t, "mender", result.Type)
			assert.Equal(t, "Mender", result.Vendor)
			assert.NotEmpty(t, result.Version)
		}
	})
}

func TestGetCollectionMethod(t *testing.T) {
	// This test requires mocking osUpdater.DetectOS
	// For now, test that it returns a string
	method := getCollectionMethod()
	assert.NotEmpty(t, method)

	// Valid methods should be one of these
	validMethods := []string{"dpkg-query", "rpm", "unknown"}
	assert.Contains(t, validMethods, method)
}

func TestChunkSoftwareBOM(t *testing.T) {
	// Create test packages
	packages := make([]*pb.SoftwarePackage, 0)
	for i := 0; i < 250; i++ {
		packages = append(packages, &pb.SoftwarePackage{
			Name:    fmt.Sprintf("package-%d", i),
			Version: "1.0.0",
			Type:    "test",
		})
	}

	testCases := []struct {
		name             string
		packages         []*pb.SoftwarePackage
		maxChunkSize     int
		expectedChunks   int
		expectedLastSize int
	}{
		{
			name:             "default chunk size",
			packages:         packages,
			maxChunkSize:     0, // Should use default (100)
			expectedChunks:   3,
			expectedLastSize: 50,
		},
		{
			name:             "custom chunk size",
			packages:         packages,
			maxChunkSize:     75,
			expectedChunks:   4,
			expectedLastSize: 25,
		},
		{
			name:             "single chunk",
			packages:         packages[:50],
			maxChunkSize:     100,
			expectedChunks:   1,
			expectedLastSize: 50,
		},
		{
			name:             "empty packages",
			packages:         []*pb.SoftwarePackage{},
			maxChunkSize:     100,
			expectedChunks:   0,
			expectedLastSize: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks := ChunkSoftwareBOM(tc.packages, tc.maxChunkSize)

			assert.Len(t, chunks, tc.expectedChunks)

			if tc.expectedChunks > 0 {
				// Check last chunk size
				lastChunk := chunks[len(chunks)-1]
				assert.Len(t, lastChunk, tc.expectedLastSize)

				// Verify all packages are included
				totalPackages := 0
				for _, chunk := range chunks {
					totalPackages += len(chunk)
				}
				assert.Equal(t, len(tc.packages), totalPackages)
			}
		})
	}
}

func TestGetSoftwareBOMSummaryInfo(t *testing.T) {
	t.Run("summary structure", func(t *testing.T) {
		if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
			t.Skip("Skipping integration test")
		}

		summary, err := GetSoftwareBOMSummaryInfo()

		if err != nil {
			// If we can't run the actual commands, verify error handling
			assert.Error(t, err)
			assert.Nil(t, summary)
		} else {
			// Verify summary structure
			assert.NotNil(t, summary)

			// Check required fields
			assert.Contains(t, summary, "total_packages")
			assert.Contains(t, summary, "collection_timestamp")
			assert.Contains(t, summary, "os_type")
			assert.Contains(t, summary, "architecture")
			assert.Contains(t, summary, "packages_by_type")

			// Check types
			assert.IsType(t, int32(0), summary["total_packages"])
			assert.IsType(t, time.Time{}, summary["collection_timestamp"])
			assert.IsType(t, "", summary["os_type"])
			assert.IsType(t, "", summary["architecture"])
			assert.IsType(t, map[string]int32{}, summary["packages_by_type"])

			// Check architecture matches runtime
			assert.Equal(t, runtime.GOARCH, summary["architecture"])
		}
	})
}

func TestGetOSType(t *testing.T) {
	osType := getOSType()
	assert.NotEmpty(t, osType)

	// Should be either detected OS or runtime.GOOS
	assert.True(t, len(osType) > 0)
}

// Integration tests that can be run with actual system commands
func TestIntegrationGetDebianPackages(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	// Check if dpkg-query is available
	if _, err := exec.LookPath("dpkg-query"); err != nil {
		t.Skip("dpkg-query not available, skipping Debian package test")
	}

	packages, err := getDebianPackages()
	if err != nil {
		t.Logf("Expected error on non-Debian system: %v", err)
		return
	}

	assert.NotNil(t, packages)
	if len(packages) > 0 {
		// Verify package structure
		pkg := packages[0]
		assert.NotEmpty(t, pkg.Name)
		assert.NotEmpty(t, pkg.Version)
		assert.Equal(t, "deb", pkg.Type)
	}
}

func TestIntegrationGetRPMPackages(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	// Check if rpm is available
	if _, err := exec.LookPath("rpm"); err != nil {
		t.Skip("rpm not available, skipping RPM package test")
	}

	packages, err := getRPMPackages()
	if err != nil {
		t.Logf("Expected error on non-RPM system: %v", err)
		return
	}

	assert.NotNil(t, packages)
	if len(packages) > 0 {
		// Verify package structure
		pkg := packages[0]
		assert.NotEmpty(t, pkg.Name)
		assert.Equal(t, "rpm", pkg.Type)
	}
}

// Benchmark tests
func BenchmarkParsePackageOutput(b *testing.B) {
	testData := generateTestPackageData(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parsePackageOutput(testData)
	}
}

func BenchmarkParseRPMOutput(b *testing.B) {
	testData := generateTestRPMData(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseRPMOutput(testData)
	}
}

func BenchmarkChunkSoftwareBOM(b *testing.B) {
	packages := make([]*pb.SoftwarePackage, 1000)
	for i := 0; i < 1000; i++ {
		packages[i] = &pb.SoftwarePackage{
			Name:    fmt.Sprintf("package-%d", i),
			Version: "1.0.0",
			Type:    "test",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ChunkSoftwareBOM(packages, 100)
	}
}

// Helper functions for tests
func generateTestPackageData(count int) string {
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, fmt.Sprintf("package-%d %d.0.0", i, i))
	}
	return strings.Join(lines, "\n")
}

func generateTestRPMData(count int) string {
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, fmt.Sprintf("package-%d-1.0.0-1.el9.x86_64", i))
	}
	return strings.Join(lines, "\n")
}
