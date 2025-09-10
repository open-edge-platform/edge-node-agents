/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFirmwareInfo(t *testing.T) {
	t.Run("success on Linux", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Test only runs on Linux")
		}

		fw, err := GetFirmwareInfo()

		assert.NoError(t, err)
		assert.NotNil(t, fw)

		assert.NotNil(t, fw.BiosVendor)
		assert.NotNil(t, fw.BiosVersion)
	})

	t.Run("non-Linux system", func(t *testing.T) {
		if runtime.GOOS == "linux" {
			t.Skip("Test only runs on non-Linux systems")
		}

		fw, err := GetFirmwareInfo()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "firmware information only supported on Linux")
		assert.NotNil(t, fw)
	})
}

func TestGetBootFirmwareInfo(t *testing.T) {
	t.Run("should return same as GetFirmwareInfo", func(t *testing.T) {
		fw1, err1 := GetFirmwareInfo()
		fw2, err2 := GetBootFirmwareInfo()

		if err1 != nil {
			assert.Error(t, err2)
		} else {
			assert.NoError(t, err2)
		}

		assert.Equal(t, fw1.BiosVendor, fw2.BiosVendor)
		assert.Equal(t, fw1.BiosVersion, fw2.BiosVersion)
	})
}

func TestIsDeviceTreeAvailable(t *testing.T) {
	t.Run("device tree availability", func(t *testing.T) {
		available := isDeviceTreeAvailable()

		assert.IsType(t, false, available)

		if available {
			_, err := os.Stat(DEVICE_TREE_FW_PATH)
			assert.NoError(t, err)
		}
	})
}

func TestReadFileContent(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		content, err := readFileContent("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
		assert.Empty(t, content)
	})

	t.Run("non-existent file", func(t *testing.T) {
		content, err := readFileContent("/non/existent/file")
		assert.Error(t, err)
		assert.Empty(t, content)
	})

	t.Run("valid file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test_firmware_*.txt")
		require.NoError(t, err)
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("Failed to remove temp file: %v", err)
			}
		}()

		testContent := "test content\x00with null bytes\n  "
		_, err = tmpFile.Write([]byte(testContent))
		require.NoError(t, err)
		tmpFile.Close()

		content, err := readFileContent(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, "test contentwith null bytes", content)
	})

	t.Run("file with only whitespace", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test_firmware_*.txt")
		require.NoError(t, err)
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("Failed to remove temp file: %v", err)
			}
		}()

		_, err = tmpFile.Write([]byte("   \n\t  "))
		require.NoError(t, err)
		tmpFile.Close()

		content, err := readFileContent(tmpFile.Name())
		assert.NoError(t, err)
		assert.Empty(t, content)
	})
}

func TestParseBiosDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		shouldError bool
		expectYear  int
	}{
		{
			name:        "empty string",
			dateStr:     "",
			shouldError: true,
		},
		{
			name:        "too long string",
			dateStr:     "this is a very long string that should be rejected because it's too long for a date",
			shouldError: true,
		},
		{
			name:        "MM/DD/YYYY format",
			dateStr:     "01/15/2023",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "MM/DD/YY format",
			dateStr:     "01/15/23",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "YYYY-MM-DD format",
			dateStr:     "2023-01-15",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "DD/MM/YYYY format",
			dateStr:     "15/01/2023",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "YYYY/MM/DD format",
			dateStr:     "2023/01/15",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "Mon DD, YYYY format",
			dateStr:     "Jan 15, 2023",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "Month DD, YYYY format",
			dateStr:     "January 15, 2023",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "DD Mon YYYY format",
			dateStr:     "15 Jan 2023",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "YYYY.MM.DD format",
			dateStr:     "2023.01.15",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "year only extraction",
			dateStr:     "some text 2023 more text",
			shouldError: false,
			expectYear:  2023,
		},
		{
			name:        "invalid format",
			dateStr:     "invalid date format",
			shouldError: true,
		},
		{
			name:        "future year boundary",
			dateStr:     "2050-01-01",
			shouldError: false,
			expectYear:  2050,
		},
		{
			name:        "past year boundary",
			dateStr:     "1980-01-01",
			shouldError: false,
			expectYear:  1980,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBiosDate(tt.dateStr)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if tt.expectYear > 0 {
					parsedTime := result.AsTime()
					assert.Equal(t, tt.expectYear, parsedTime.Year())
				}
			}
		})
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected string
	}{
		{
			name:     "valid year in slash format",
			dateStr:  "01/15/2023",
			expected: "2023",
		},
		{
			name:     "valid year in dash format",
			dateStr:  "2023-01-15",
			expected: "2023",
		},
		{
			name:     "valid year in dot format",
			dateStr:  "15.01.2023",
			expected: "2023",
		},
		{
			name:     "valid year in space format",
			dateStr:  "15 Jan 2023",
			expected: "2023",
		},
		{
			name:     "year at boundary (1980)",
			dateStr:  "1980-01-01",
			expected: "1980",
		},
		{
			name:     "year at boundary (2050)",
			dateStr:  "2050-01-01",
			expected: "2050",
		},
		{
			name:     "year too old",
			dateStr:  "1979-01-01",
			expected: "",
		},
		{
			name:     "year too new",
			dateStr:  "2051-01-01",
			expected: "",
		},
		{
			name:     "no year",
			dateStr:  "Jan 15",
			expected: "",
		},
		{
			name:     "invalid year format",
			dateStr:  "abcd-01-01",
			expected: "",
		},
		{
			name:     "three digit year",
			dateStr:  "123-01-01",
			expected: "",
		},
		{
			name:     "five digit year",
			dateStr:  "12345-01-01",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractYear(tt.dateStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "all digits",
			input:    "12345",
			expected: true,
		},
		{
			name:     "single digit",
			input:    "5",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "contains letter",
			input:    "123a",
			expected: false,
		},
		{
			name:     "contains space",
			input:    "123 ",
			expected: false,
		},
		{
			name:     "contains dash",
			input:    "123-",
			expected: false,
		},
		{
			name:     "contains dot",
			input:    "123.45",
			expected: false,
		},
		{
			name:     "all letters",
			input:    "abcd",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFirmwareFromDMI(t *testing.T) {
	t.Run("with empty fw struct", func(t *testing.T) {
		fw := &pb.FirmwareInfo{}
		err := getFirmwareFromDMI(fw)

		assert.NoError(t, err)
		assert.NotNil(t, fw)
	})
}

func TestGetFirmwareFromDeviceTree(t *testing.T) {
	t.Run("with empty fw struct", func(t *testing.T) {
		fw := &pb.FirmwareInfo{}
		err := getFirmwareFromDeviceTree(fw)

		assert.NoError(t, err)
		assert.NotNil(t, fw)
	})
}

// Integration tests with temporary files
func TestFirmwareInfoIntegration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Integration tests only run on Linux")
	}

	t.Run("DMI integration", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dmi_test_*")
		require.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Logf("Failed to remove temp directory: %v", err)
			}
		}()

		vendorPath := filepath.Join(tmpDir, "bios_vendor")
		versionPath := filepath.Join(tmpDir, "bios_version")
		datePath := filepath.Join(tmpDir, "bios_date")

		err = os.WriteFile(vendorPath, []byte("Test Vendor\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(versionPath, []byte("1.0.0\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(datePath, []byte("01/15/2023\n"), 0644)
		require.NoError(t, err)

		// Test reading files
		vendor, err := readFileContent(vendorPath)
		assert.NoError(t, err)
		assert.Equal(t, "Test Vendor", vendor)

		version, err := readFileContent(versionPath)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0", version)

		date, err := readFileContent(datePath)
		assert.NoError(t, err)
		assert.Equal(t, "01/15/2023", date)
	})
}

// Benchmark tests
func BenchmarkGetFirmwareInfo(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("Benchmark only runs on Linux")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetFirmwareInfo()
		if err != nil {
			b.Logf("GetFirmwareInfo failed: %v", err)
		}
	}
}

func BenchmarkParseBiosDate(b *testing.B) {
	testDates := []string{
		"01/15/2023",
		"2023-01-15",
		"January 15, 2023",
		"15 Jan 2023",
		"invalid date",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, date := range testDates {
			_, err := parseBiosDate(date)
			if err != nil {
				b.Logf("parseBiosDate failed for %s: %v", date, err)
			}
		}
	}
}

func BenchmarkReadFileContent(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "benchmark_*.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.Write([]byte("test content for benchmarking"))
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readFileContent(tmpFile.Name())
		if err != nil {
			b.Logf("readFileContent failed: %v", err)
		}
	}
}
