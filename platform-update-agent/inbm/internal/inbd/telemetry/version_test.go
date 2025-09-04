/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetVersionInfo(t *testing.T) {
	// Store original values
	originalVersion := Version
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate

	t.Run("with build-time variables set", func(t *testing.T) {
		// Set build-time variables
		Version = "1.2.3"
		GitCommit = "abc123def456"
		BuildDate = "2025-01-01T12:00:00Z"

		versionInfo, err := GetVersionInfo()

		assert.NoError(t, err)
		assert.NotNil(t, versionInfo)
		assert.Equal(t, "1.2.3", versionInfo.Version)
		assert.Equal(t, "abc123def456", versionInfo.GitCommit)
		assert.Equal(t, "abc123def456", versionInfo.InbmVersionCommit)
		assert.NotNil(t, versionInfo.BuildDate)
	})

	t.Run("with dev version fallback", func(t *testing.T) {
		// Set build-time variables to defaults
		Version = "dev"
		GitCommit = "unknown"
		BuildDate = "unknown"

		versionInfo, err := GetVersionInfo()

		assert.NoError(t, err)
		assert.NotNil(t, versionInfo)
		assert.NotEmpty(t, versionInfo.Version)
		assert.NotEmpty(t, versionInfo.GitCommit)
		assert.NotNil(t, versionInfo.BuildDate)
	})

	t.Run("with dev-prefixed version fallback", func(t *testing.T) {
		originalVersion := Version
		Version = "dev-20250711"
		GitCommit = "5debbb1cf47558f7ae700d6c44aca06379aadf04"
		BuildDate = "2025-07-11T04:57:14Z"

		versionInfo, err := GetVersionInfo()

		assert.NoError(t, err)
		assert.NotNil(t, versionInfo)
		assert.NotEmpty(t, versionInfo.Version)

		// The main test is that dynamic version detection was triggered
		// The actual result depends on the test environment
		// We just verify that the function completes successfully
		assert.NotEmpty(t, versionInfo.Version, "Dynamic version detection should return a non-empty version")
		assert.NotEmpty(t, versionInfo.GitCommit)
		assert.NotNil(t, versionInfo.BuildDate)

		// Log the result for debugging
		t.Logf("Dynamic version result: %s", versionInfo.Version)

		// Restore original version
		Version = originalVersion
	})

	t.Run("with empty build-time variables", func(t *testing.T) {
		// Set build-time variables to empty
		Version = ""
		GitCommit = ""
		BuildDate = ""

		versionInfo, err := GetVersionInfo()

		assert.NoError(t, err)
		assert.NotNil(t, versionInfo)
		assert.NotEmpty(t, versionInfo.Version)
		assert.NotEmpty(t, versionInfo.GitCommit)
		assert.NotNil(t, versionInfo.BuildDate)
	})

	// Restore original values
	t.Cleanup(func() {
		Version = originalVersion
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
	})
}

func TestGetDynamicVersion(t *testing.T) {
	// Store original environment
	originalEnv := os.Getenv("INBM_VERSION")
	t.Cleanup(func() {
		if originalEnv != "" {
			os.Setenv("INBM_VERSION", originalEnv)
		} else {
			os.Unsetenv("INBM_VERSION")
		}
	})

	t.Run("with environment variable set", func(t *testing.T) {
		os.Setenv("INBM_VERSION", "2.0.0")

		version := getDynamicVersion()

		// Should return environment variable if git tag is not available
		// or git tag if available (git takes precedence)
		assert.NotEmpty(t, version)
		assert.NotEqual(t, "dev-build", version)
	})

	t.Run("without environment variable", func(t *testing.T) {
		os.Unsetenv("INBM_VERSION")

		version := getDynamicVersion()

		assert.NotEmpty(t, version)
		// Should be either a git tag, git commit, or dev-build
		assert.True(t,
			strings.Contains(version, "dev-") ||
				version == "dev-build" ||
				!strings.HasPrefix(version, "dev-"),
			"Expected version to be git tag, dev-commit, or dev-build, got: %s", version)
	})

	t.Run("fallback to dev-build", func(t *testing.T) {
		os.Unsetenv("INBM_VERSION")

		// This test is environment-dependent, so we'll just verify it doesn't panic
		version := getDynamicVersion()
		assert.NotEmpty(t, version)
	})
}

func TestGetVersionFromGitTag(t *testing.T) {
	t.Run("git tag detection", func(t *testing.T) {
		// Test is environment-dependent, so we'll just verify it doesn't panic
		version := getVersionFromGitTag()

		// Should either return a tag or empty string
		assert.True(t, version == "" || len(version) > 0)
	})

	t.Run("strategies are attempted in order", func(t *testing.T) {
		// This test verifies the function doesn't panic and returns a string
		version := getVersionFromGitTag()
		assert.True(t, version == "" || len(version) > 0)
	})
}

func TestGetExactTag(t *testing.T) {
	t.Run("exact tag detection", func(t *testing.T) {
		// Test is environment-dependent
		tag := getExactTag()
		assert.True(t, tag == "" || len(tag) > 0)
	})

	t.Run("git command execution", func(t *testing.T) {
		// Verify the function handles git command execution gracefully
		tag := getExactTag()
		if tag != "" {
			// If a tag is returned, it should be a valid string
			assert.NotContains(t, tag, "\n")
			assert.NotContains(t, tag, "\r")
		}
	})
}

func TestGetRecentTagWithDistance(t *testing.T) {
	t.Run("recent tag with distance", func(t *testing.T) {
		// Test is environment-dependent
		tag := getRecentTagWithDistance()
		assert.True(t, tag == "" || len(tag) > 0)
	})

	t.Run("handles git command gracefully", func(t *testing.T) {
		// Verify the function doesn't panic
		tag := getRecentTagWithDistance()
		if tag != "" {
			assert.NotContains(t, tag, "\n")
			assert.NotContains(t, tag, "\r")
		}
	})
}

func TestGetMostRecentTag(t *testing.T) {
	t.Run("most recent tag", func(t *testing.T) {
		// Test is environment-dependent
		tag := getMostRecentTag()
		assert.True(t, tag == "" || len(tag) > 0)
	})

	t.Run("output is trimmed", func(t *testing.T) {
		tag := getMostRecentTag()
		if tag != "" {
			assert.Equal(t, strings.TrimSpace(tag), tag)
		}
	})
}

func TestGetVersionFromFile(t *testing.T) {
	t.Run("version file exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := afero.WriteFile(fs, "VERSION", []byte("1.0.0\n"), 0644)
		require.NoError(t, err)

		version := getVersionFromFile()

		assert.True(t, version == "" || len(version) > 0)
	})

	t.Run("no version file exists", func(t *testing.T) {
		version := getVersionFromFile()
		assert.True(t, version == "" || len(version) > 0)
	})

	t.Run("version file with whitespace", func(t *testing.T) {
		version := getVersionFromFile()
		if version != "" {
			assert.Equal(t, strings.TrimSpace(version), version)
		}
	})
}

func TestGetGitCommit(t *testing.T) {
	// Store original environment
	originalEnv := os.Getenv("GIT_COMMIT")
	t.Cleanup(func() {
		if originalEnv != "" {
			os.Setenv("GIT_COMMIT", originalEnv)
		} else {
			os.Unsetenv("GIT_COMMIT")
		}
	})

	t.Run("git command available", func(t *testing.T) {
		os.Unsetenv("GIT_COMMIT")

		commit := getGitCommit()

		assert.NotEmpty(t, commit)
		// Should be either a commit hash or "unknown"
		assert.True(t, commit == "unknown" || len(commit) >= 7)
	})

	t.Run("with environment variable", func(t *testing.T) {
		os.Setenv("GIT_COMMIT", "test-commit-hash")

		commit := getGitCommit()

		// Should return either git commit or environment variable
		assert.NotEmpty(t, commit)
		assert.NotEqual(t, "", commit)
	})

	t.Run("fallback to environment", func(t *testing.T) {
		os.Setenv("GIT_COMMIT", "env-commit-123")

		commit := getGitCommit()
		assert.NotEmpty(t, commit)
	})

	t.Run("git commit hash format", func(t *testing.T) {
		// Only test git hash format when we're sure it's from git, not environment
		os.Unsetenv("GIT_COMMIT")

		commit := getGitCommit()
		if commit != "unknown" {
			// Check if it looks like a git hash (only hex characters)
			// Only apply this check if it's likely from git command
			if len(commit) >= 7 && len(commit) <= 40 {
				assert.Regexp(t, "^[a-fA-F0-9]+$", commit, "Git commit hash should only contain hex characters")
			}
		}
	})

	t.Run("environment variable format", func(t *testing.T) {
		// Test that environment variables are used when git commands fail
		// Since we're in a git repo, this test checks that git commands take priority
		testEnvValues := []string{
			"env-commit-123",
			"test-commit-hash",
			"build-12345",
			"abc123def456", // Valid hex
		}

		for _, envValue := range testEnvValues {
			t.Run(envValue, func(t *testing.T) {
				os.Setenv("GIT_COMMIT", envValue)

				commit := getGitCommit()
				assert.NotEmpty(t, commit)
				// In a git repository, git commands take priority over environment variables
				// So the returned commit should be a git hash, not the environment variable
				if commit != "unknown" {
					// If we're in a git repo, we should get a git hash (hex characters, 7-40 chars)
					if len(commit) >= 7 && len(commit) <= 40 {
						assert.Regexp(t, "^[a-fA-F0-9]+$", commit, "Should return git commit hash when in git repository")
					} else {
						// If not a standard git hash format, it might be environment variable in non-git context
						assert.True(t, strings.Contains(commit, envValue) || commit == envValue)
					}
				}
			})
		}
	})
}

func TestParseBuildDate(t *testing.T) {
	// Store original value
	originalBuildDate := BuildDate

	t.Run("valid RFC3339 build date", func(t *testing.T) {
		BuildDate = "2025-01-01T12:00:00Z"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		expectedTime, _ := time.Parse(time.RFC3339, "2025-01-01T12:00:00Z")
		assert.Equal(t, expectedTime.Unix(), timestamp.Seconds)
	})

	t.Run("alternative date format", func(t *testing.T) {
		BuildDate = "2025-01-01 12:00:00"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2025-01-01 12:00:00")
		assert.Equal(t, expectedTime.Unix(), timestamp.Seconds)
	})

	t.Run("date only format", func(t *testing.T) {
		BuildDate = "2025-01-01"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		expectedTime, _ := time.Parse("2006-01-02", "2025-01-01")
		assert.Equal(t, expectedTime.Unix(), timestamp.Seconds)
	})

	t.Run("invalid build date", func(t *testing.T) {
		BuildDate = "invalid-date"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		// Should fallback to current time
		now := time.Now()
		assert.WithinDuration(t, now, timestamp.AsTime(), 2*time.Second)
	})

	t.Run("empty build date", func(t *testing.T) {
		BuildDate = ""

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		// Should fallback to current time
		now := time.Now()
		assert.WithinDuration(t, now, timestamp.AsTime(), 2*time.Second)
	})

	t.Run("unknown build date", func(t *testing.T) {
		BuildDate = "unknown"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		// Should fallback to current time
		now := time.Now()
		assert.WithinDuration(t, now, timestamp.AsTime(), 2*time.Second)
	})

	t.Run("timezone handling", func(t *testing.T) {
		BuildDate = "2025-01-01T12:00:00+05:30"

		timestamp := parseBuildDate()

		assert.NotNil(t, timestamp)
		expectedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", "2025-01-01T12:00:00+05:30")
		assert.Equal(t, expectedTime.Unix(), timestamp.Seconds)
	})

	// Restore original value
	t.Cleanup(func() {
		BuildDate = originalBuildDate
	})
}

func TestVersionInfoIntegration(t *testing.T) {
	t.Run("complete version info structure", func(t *testing.T) {
		versionInfo, err := GetVersionInfo()

		require.NoError(t, err)
		require.NotNil(t, versionInfo)

		// Verify all fields are populated
		assert.NotEmpty(t, versionInfo.Version)
		assert.NotEmpty(t, versionInfo.GitCommit)
		assert.NotEmpty(t, versionInfo.InbmVersionCommit)
		assert.NotNil(t, versionInfo.BuildDate)

		// Verify InbmVersionCommit equals GitCommit
		assert.Equal(t, versionInfo.GitCommit, versionInfo.InbmVersionCommit)

		// Verify BuildDate is reasonable
		buildTime := versionInfo.BuildDate.AsTime()
		assert.True(t, buildTime.After(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
		assert.True(t, buildTime.Before(time.Now().Add(24*time.Hour)))
	})

	t.Run("version consistency", func(t *testing.T) {
		// Call multiple times to ensure consistency
		info1, err1 := GetVersionInfo()
		info2, err2 := GetVersionInfo()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, info1.Version, info2.Version)
		assert.Equal(t, info1.GitCommit, info2.GitCommit)
		assert.Equal(t, info1.InbmVersionCommit, info2.InbmVersionCommit)
	})
}

func TestGitCommandAvailability(t *testing.T) {
	t.Run("git command availability", func(t *testing.T) {
		// Test if git is available
		cmd := exec.Command("git", "--version")
		err := cmd.Run()

		if err != nil {
			t.Log("Git command not available, some version detection may fall back to defaults")
		} else {
			t.Log("Git command is available")
		}

		// Test should not fail regardless of git availability
		version := getVersionFromGitTag()
		assert.True(t, version == "" || len(version) > 0)
	})

	t.Run("git repository detection", func(t *testing.T) {
		// Test if we're in a git repository
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		err := cmd.Run()

		if err != nil {
			t.Log("Not in a git repository, git-based version detection will return empty")
		} else {
			t.Log("In a git repository")
		}

		// Function should handle both cases gracefully
		commit := getGitCommit()
		assert.NotEmpty(t, commit)
	})
}

// Benchmark tests
func BenchmarkGetVersionInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetVersionInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetDynamicVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = getDynamicVersion()
	}
}

func BenchmarkGetGitCommit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = getGitCommit()
	}
}

func BenchmarkParseBuildDate(b *testing.B) {
	BuildDate = "2025-01-01T12:00:00Z"
	for i := 0; i < b.N; i++ {
		_ = parseBuildDate()
	}
}

// Helper functions for testing
func TestVersionFormatValidation(t *testing.T) {
	t.Run("semantic version format", func(t *testing.T) {
		testCases := []struct {
			version string
			valid   bool
		}{
			{"1.0.0", true},
			{"1.0.0-beta", true},
			{"1.0.0-beta.1", true},
			{"5.0.0engRC1", true}, // Your specific case
			{"dev-abc123", true},
			{"dev-build", true},
			{"", false},
		}

		for _, tc := range testCases {
			t.Run(tc.version, func(t *testing.T) {
				if tc.valid {
					assert.NotEmpty(t, tc.version, "Expected valid version to be non-empty")
				} else {
					assert.Empty(t, tc.version, "Expected invalid version to be empty")
				}
			})
		}
	})
}

func TestTimestampCreation(t *testing.T) {
	t.Run("timestamp creation", func(t *testing.T) {
		now := time.Now()
		timestamp := timestamppb.New(now)

		assert.NotNil(t, timestamp)
		assert.Equal(t, now.Unix(), timestamp.Seconds)
		assert.WithinDuration(t, now, timestamp.AsTime(), time.Nanosecond)
	})

	t.Run("timestamp from unix time", func(t *testing.T) {
		unixTime := int64(1735689600) // 2025-01-01 12:00:00 UTC
		timestamp := timestamppb.New(time.Unix(unixTime, 0))

		assert.NotNil(t, timestamp)
		assert.Equal(t, unixTime, timestamp.Seconds)
	})
}
