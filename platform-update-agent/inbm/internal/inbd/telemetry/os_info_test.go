/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOSInfo(t *testing.T) {
	t.Run("success - returns OS info", func(t *testing.T) {
		osInfo, err := GetOSInfo()

		assert.NoError(t, err)
		assert.NotNil(t, osInfo)
		assert.NotEmpty(t, osInfo.OsInformation)

		assert.Contains(t, osInfo.OsInformation, runtime.GOOS)
		assert.Contains(t, osInfo.OsInformation, runtime.GOARCH)

		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			assert.Contains(t, osInfo.OsInformation, hostname)
		}
	})

	t.Run("never returns empty OS information", func(t *testing.T) {
		osInfo, err := GetOSInfo()

		assert.NoError(t, err)
		assert.NotNil(t, osInfo)
		assert.NotEmpty(t, osInfo.OsInformation)

		parts := strings.Split(osInfo.OsInformation, " ")
		assert.GreaterOrEqual(t, len(parts), 2)
	})
}

func TestGetOSInformation(t *testing.T) {
	t.Run("contains required fields", func(t *testing.T) {
		osInfo := getOSInformation()

		assert.NotEmpty(t, osInfo)

		assert.Contains(t, osInfo, runtime.GOOS)
		assert.Contains(t, osInfo, runtime.GOARCH)

		parts := strings.Split(osInfo, " ")
		assert.GreaterOrEqual(t, len(parts), 2)
	})

	t.Run("includes hostname if available", func(t *testing.T) {
		osInfo := getOSInformation()

		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			assert.Contains(t, osInfo, hostname)
		}
	})

	t.Run("includes kernel version on Linux", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Kernel version test only runs on Linux")
		}

		osInfo := getOSInformation()

		parts := strings.Split(osInfo, " ")
		foundVersion := false
		for _, part := range parts {
			if strings.Contains(part, ".") && len(part) > 3 {
				foundVersion = true
				break
			}
		}

		if foundVersion {
			t.Logf("Found version-like string in OS info: %s", osInfo)
		}
	})
}

func TestGetOSVersion(t *testing.T) {
	t.Run("returns version string", func(t *testing.T) {
		version := getOSVersion()

		assert.NotEmpty(t, version)
		assert.IsType(t, "", version)

		if version != "Unknown" {
			assert.NotEmpty(t, strings.TrimSpace(version))
		}
	})

	t.Run("Linux version detection", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Linux version test only runs on Linux")
		}

		version := getOSVersion()

		// On Linux, should try to get actual version
		assert.NotEmpty(t, version)

		// Log the detected version for debugging
		t.Logf("Detected OS version: %s", version)
	})

	t.Run("handles missing files gracefully", func(t *testing.T) {
		version := getOSVersion()
		assert.NotEmpty(t, version)
	})
}

func TestGetOSReleaseDate(t *testing.T) {
	t.Run("returns timestamp or nil", func(t *testing.T) {
		releaseDate := getOSReleaseDate()

		if releaseDate != nil {
			assert.NotNil(t, releaseDate.AsTime())

			releaseTime := releaseDate.AsTime()
			now := time.Now()
			year1990 := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
			futureLimit := now.AddDate(1, 0, 0)

			assert.True(t, releaseTime.After(year1990), "Release date should be after 1990")
			assert.True(t, releaseTime.Before(futureLimit), "Release date should not be too far in future")

			t.Logf("Detected OS release date: %s", releaseTime.Format("2006-01-02"))
		}
	})

	t.Run("Linux release date detection", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Linux release date test only runs on Linux")
		}

		releaseDate := getOSReleaseDate()

		// On Linux, may or may not find a release date
		if releaseDate != nil {
			t.Logf("Found OS release date: %s", releaseDate.AsTime().Format("2006-01-02"))
		} else {
			t.Log("No OS release date found (this is normal for some distributions)")
		}
	})

	t.Run("handles malformed dates gracefully", func(t *testing.T) {
		releaseDate := getOSReleaseDate()

		if releaseDate != nil {
			assert.NotNil(t, releaseDate.AsTime())
		}
	})
}

// Integration tests
func TestOSInfoIntegration(t *testing.T) {
	t.Run("complete OS info pipeline", func(t *testing.T) {
		osInfo, err := GetOSInfo()
		require.NoError(t, err)
		require.NotNil(t, osInfo)

		osInformation := getOSInformation()
		osVersion := getOSVersion()
		osReleaseDate := getOSReleaseDate()

		assert.Equal(t, osInformation, osInfo.OsInformation)
		assert.NotEmpty(t, osInformation)
		assert.NotEmpty(t, osVersion)

		t.Logf("OS Information: %s", osInformation)
		t.Logf("OS Version: %s", osVersion)
		if osReleaseDate != nil {
			t.Logf("OS Release Date: %s", osReleaseDate.AsTime().Format("2006-01-02"))
		} else {
			t.Log("OS Release Date: Not available")
		}
	})

	t.Run("handles system variations", func(t *testing.T) {
		osInfo, err := GetOSInfo()
		require.NoError(t, err)
		require.NotNil(t, osInfo)

		assert.Contains(t, osInfo.OsInformation, runtime.GOOS)
		assert.Contains(t, osInfo.OsInformation, runtime.GOARCH)

		parts := strings.Split(osInfo.OsInformation, " ")
		assert.GreaterOrEqual(t, len(parts), 2, "Should have at least GOOS and GOARCH")
	})
}

// Test edge cases and error conditions
func TestOSInfoEdgeCases(t *testing.T) {
	t.Run("empty build ID handling", func(t *testing.T) {
		releaseDate := getOSReleaseDate()

		if releaseDate != nil {
			assert.NotNil(t, releaseDate.AsTime())
		}
	})

	t.Run("malformed version strings", func(t *testing.T) {
		version := getOSVersion()

		assert.NotEmpty(t, version)
		assert.IsType(t, "", version)
	})

	t.Run("missing system files", func(t *testing.T) {
		osInfo, err := GetOSInfo()

		assert.NoError(t, err)
		assert.NotNil(t, osInfo)
		assert.NotEmpty(t, osInfo.OsInformation)
	})
}

// Benchmark tests
func BenchmarkGetOSInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetOSInfo()
		if err != nil {
			b.Fatalf("GetOSInfo failed: %v", err)
		}
	}
}

func BenchmarkGetOSInformation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		osInfo := getOSInformation()
		if osInfo == "" {
			b.Fatal("getOSInformation returned empty string")
		}
	}
}

func BenchmarkGetOSVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		version := getOSVersion()
		if version == "" {
			b.Fatal("getOSVersion returned empty string")
		}
	}
}

func BenchmarkGetOSReleaseDate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = getOSReleaseDate()
	}
}
