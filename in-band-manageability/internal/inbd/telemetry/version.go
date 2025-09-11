/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"github.com/spf13/afero"
	"os"
	"os/exec"
	"strings"
	"time"

	utils "github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Build-time variables (set via ldflags during build)
var (
	Version   = "dev"     // Set via -ldflags "-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.Version=x.x.x"
	GitCommit = "unknown" // Set via -ldflags "-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.GitCommit=commit_hash"
	BuildDate = "unknown" // Set via -ldflags "-X github.com/open-edge-platform/edge-node-agents/in-band-manageability/internal/inbd/telemetry.BuildDate=build_date"
)

// GetVersionInfo retrieves version information
func GetVersionInfo() (*pb.VersionInfo, error) {
	// Get version from build-time variable or fallback to dynamic detection
	version := Version
	if version == "dev" || version == "" || strings.HasPrefix(version, "dev-") {
		version = getDynamicVersion()
	}

	// Get git commit from build-time variable or fallback to dynamic detection
	gitCommit := GitCommit
	if gitCommit == "unknown" || gitCommit == "" {
		gitCommit = getGitCommit()
	}

	// Get INBM version commit
	inbmVersionCommit := gitCommit

	// Parse build date or use current time
	buildDate := parseBuildDate()

	return &pb.VersionInfo{
		Version:           version,
		InbmVersionCommit: inbmVersionCommit,
		BuildDate:         buildDate,
		GitCommit:         gitCommit,
	}, nil
}

// getDynamicVersion attempts to get version from git tags or environment
func getDynamicVersion() string {
	if version := getVersionFromGitTag(); version != "" {
		return version
	}

	if version := os.Getenv("INBM_VERSION"); version != "" {
		return version
	}

	if version := getVersionFromFile(); version != "" {
		return version
	}

	// Fallback to development version with date
	fallback := "dev-" + time.Now().Format("20060102")
	return fallback
}

// getVersionFromGitTag gets version from git tag with multiple strategies
func getVersionFromGitTag() string {
	// Strategy 1: Get exact tag for current commit
	if tag := getExactTag(); tag != "" {
		return tag
	}

	// Strategy 2: Get the most recent tag
	if tag := getMostRecentTag(); tag != "" {
		return tag
	}

	// Strategy 3: Get recent tag with distance info (fallback)
	if tag := getRecentTagWithDistance(); tag != "" {
		// Clean up the tag if it has distance info
		if strings.Contains(tag, "-g") {
			// Extract just the tag part (before the distance)
			parts := strings.Split(tag, "-")
			if len(parts) >= 3 {
				// Reconstruct tag without distance info
				tagParts := parts[:len(parts)-2] // Remove last 2 parts (-15-g5debbb1)
				return strings.Join(tagParts, "-")
			}
		}
		return tag
	}

	return ""
}

// getExactTag gets the exact tag for current commit
func getExactTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

// getRecentTagWithDistance gets recent tag with commit distance
func getRecentTagWithDistance() string {
	cmd := exec.Command("git", "describe", "--tags", "--always")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

// getMostRecentTag gets the most recent tag
func getMostRecentTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	if output, err := cmd.Output(); err == nil {
		tag := strings.TrimSpace(string(output))
		if tag != "" {
			return tag
		}
	}
	return ""
}

// getVersionFromFile reads version from VERSION file
func getVersionFromFile() string {
	// Check common version file locations
	versionFiles := []string{
		"VERSION",
		"version.txt",
		"./VERSION",
		"../VERSION",
	}

	for _, file := range versionFiles {
		var fs = afero.NewOsFs()
		if data, err := utils.ReadFile(fs, file); err == nil {
			version := strings.TrimSpace(string(data))
			if version != "" {
				return version
			}
		}
	}
	return ""
}

// getGitCommit gets the current git commit hash
func getGitCommit() string {
	// Try to get full commit hash
	cmd := exec.Command("git", "rev-parse", "HEAD")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}

	// Try to get short commit hash
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}

	// Try to get from environment variable
	if commit := os.Getenv("GIT_COMMIT"); commit != "" {
		return commit
	}

	return "unknown"
}

// parseBuildDate parses build date from build-time variable or uses current time
func parseBuildDate() *timestamppb.Timestamp {
	if BuildDate != "unknown" && BuildDate != "" {
		// Try to parse build date from build-time variable
		if parsedTime, err := time.Parse(time.RFC3339, BuildDate); err == nil {
			return timestamppb.New(parsedTime)
		}
		// Try alternative formats
		formats := []string{
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if parsedTime, err := time.Parse(format, BuildDate); err == nil {
				return timestamppb.New(parsedTime)
			}
		}
	}

	// Fallback to current time
	return timestamppb.New(time.Now())
}
