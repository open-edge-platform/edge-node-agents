/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"github.com/spf13/afero"
	"os/exec"
	"runtime"
	"strings"
	"time"

	utils "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	osUpdater "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/os_updater"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// SWBOM_BYTES_SIZE equivalent - maximum packages per chunk
	maxPackagesPerChunk = 100
	menderPath          = "/etc/mender/artifact_info"
	unknown             = "Unknown"
)

// GetSoftwareBOM retrieves software BOM information
func GetSoftwareBOM() (*pb.SWBOMInfo, error) {
	swbom := &pb.SWBOMInfo{
		CollectionTimestamp: timestamppb.New(time.Now()),
		CollectionMethod:    getCollectionMethod(),
	}

	packages, err := getSoftwareBOMList()
	if err != nil {
		return nil, fmt.Errorf("failed to get software BOM list: %w", err)
	}

	swbom.Packages = packages
	return swbom, nil
}

// getSoftwareBOMList returns the software BOM list based on OS type
func getSoftwareBOMList() ([]*pb.SoftwarePackage, error) {
	osType, err := osUpdater.DetectOS()
	if err != nil {
		return nil, fmt.Errorf("failed to detect OS: %w", err)
	}

	var packages []*pb.SoftwarePackage

	switch osType {
	case "Ubuntu", "Deby":
		packages, err = getDebianPackages()
	case "YoctoX86_64", "YoctoARM":
		packages, err = getRPMPackages()
		// Add mender version for Yocto systems
		if menderPkg := getMenderVersion(); menderPkg != nil {
			packages = append(packages, menderPkg)
		}
	default:
		return nil, fmt.Errorf("unsupported OS type: %s", osType)
	}

	if err != nil {
		return nil, err
	}

	return packages, nil
}

// getDebianPackages gets packages using dpkg-query (Ubuntu/Debian)
func getDebianPackages() ([]*pb.SoftwarePackage, error) {
	cmd := exec.Command("dpkg-query", "-f", "${Package} ${Version}\n", "-W")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run dpkg-query: %w", err)
	}

	return parsePackageOutput(string(output)), nil
}

// getRPMPackages gets packages using rpm (Yocto/RPM-based)
func getRPMPackages() ([]*pb.SoftwarePackage, error) {
	cmd := exec.Command("rpm", "-qa")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rpm: %w", err)
	}

	return parseRPMOutput(string(output)), nil
}

// parsePackageOutput parses dpkg-query output format "package version"
func parsePackageOutput(output string) []*pb.SoftwarePackage {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var packages []*pb.SoftwarePackage

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, &pb.SoftwarePackage{
				Name:    parts[0],
				Version: parts[1],
				Type:    "deb",
			})
		}
	}

	return packages
}

// parseRPMOutput parses rpm output format "package-version-release.arch"
func parseRPMOutput(output string) []*pb.SoftwarePackage {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var packages []*pb.SoftwarePackage

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse RPM package name format: name-version-release.arch
		pkg := parseRPMPackageName(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	return packages
}

// parseRPMPackageName parses RPM package name format
func parseRPMPackageName(packageName string) *pb.SoftwarePackage {
	// Split by last dot to separate architecture
	parts := strings.Split(packageName, ".")
	if len(parts) < 2 {
		return &pb.SoftwarePackage{
			Name: packageName,
			Type: "rpm",
		}
	}

	arch := parts[len(parts)-1]
	nameVersionRelease := strings.Join(parts[:len(parts)-1], ".")

	// Split by hyphens to separate name, version, and release
	// This is a simplified approach - RPM naming can be complex
	hyphenParts := strings.Split(nameVersionRelease, "-")
	if len(hyphenParts) >= 3 {
		// Last two parts are usually version and release
		name := strings.Join(hyphenParts[:len(hyphenParts)-2], "-")
		version := hyphenParts[len(hyphenParts)-2]
		release := hyphenParts[len(hyphenParts)-1]

		return &pb.SoftwarePackage{
			Name:         name,
			Version:      version + "-" + release,
			Architecture: arch,
			Type:         "rpm",
		}
	}

	return &pb.SoftwarePackage{
		Name:         nameVersionRelease,
		Architecture: arch,
		Type:         "rpm",
	}
}

// getMenderVersion reads mender version from artifact_info file
func getMenderVersion() *pb.SoftwarePackage {
	version := readMenderFile(menderPath, unknown)
	if version == unknown {
		return nil
	}

	return &pb.SoftwarePackage{
		Name:    "mender",
		Version: version,
		Type:    "mender",
		Vendor:  "Mender",
	}
}

// readMenderFile reads mender version from the specified file
func readMenderFile(path, notFoundDefault string) string {
	var fs = afero.NewOsFs()
	if !utils.IsFileExist(fs, path) {
		return notFoundDefault
	}

	data, err := utils.ReadFile(fs, path)
	if err != nil {
		return "Error reading mender version: " + err.Error()
	}

	// Split by null byte and take first part, then trim newlines
	content := strings.Split(string(data), "\x00")[0]
	return strings.TrimSpace(content)
}

// getCollectionMethod returns the method used to collect software BOM
func getCollectionMethod() string {
	osType, err := osUpdater.DetectOS()
	if err != nil {
		return "unknown"
	}

	switch osType {
	case "Ubuntu", "Deby":
		return "dpkg-query"
	case "YoctoX86_64", "YoctoARM":
		return "rpm"
	default:
		return "unknown"
	}
}

// ChunkSoftwareBOM splits software BOM into chunks for transmission
// This is useful for large package lists that need to be sent in multiple parts
func ChunkSoftwareBOM(packages []*pb.SoftwarePackage, maxChunkSize int) [][]*pb.SoftwarePackage {
	if maxChunkSize <= 0 {
		maxChunkSize = maxPackagesPerChunk
	}

	var chunks [][]*pb.SoftwarePackage
	for i := 0; i < len(packages); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(packages) {
			end = len(packages)
		}
		chunks = append(chunks, packages[i:end])
	}

	return chunks
}

// GetSoftwareBOMSummary returns a summary of the software BOM
func GetSoftwareBOMSummaryInfo() (map[string]interface{}, error) {
	packages, err := getSoftwareBOMList()
	if err != nil {
		return nil, err
	}

	typeCount := make(map[string]int32)
	for _, pkg := range packages {
		typeCount[pkg.Type]++
	}

	summary := map[string]interface{}{
		"total_packages":       int32(len(packages)),
		"collection_timestamp": time.Now(),
		"os_type":              getOSType(),
		"architecture":         runtime.GOARCH,
		"packages_by_type":     typeCount,
	}

	return summary, nil
}

// getOSType returns the OS type for the summary
func getOSType() string {
	osType, err := osUpdater.DetectOS()
	if err != nil {
		return runtime.GOOS
	}
	return osType
}
