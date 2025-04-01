// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package instrument

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

const (
	CloudInitExecutableName  = "cloud-init"
	SystemdAnalyzeExecutable = "systemd-analyze"

	SkipLineMostRecentBoot = "Most Recent Boot Record"
	SkipLineEndSuccessful  = "successful"
)

// Initialize logger
var log = logger.Logger

func ReportBootStats() {
	reportKernelBootStats()
	reportCloudInitStats()
}

func reportCloudInitStats() {
	if exists := cloudInitExecutableExists(); !exists {
		log.Errorf("cloud-init executable not found, boot stats won't be reported!")
		return
	}

	cloudInitAnalyzeBootLines, err := executeCloudInitAnalyzeBoot()
	if err != nil {
		log.Errorf("Failed to read cloud-init analyze output: %v", err)
		return
	}

	for _, line := range cloudInitAnalyzeBootLines {
		if strings.Contains(line, SkipLineMostRecentBoot) ||
			len(strings.TrimSpace(line)) == 0 {
			continue
		}

		log.Info("Cloud-Init Analyze output: ", strings.TrimSpace(line))
	}

	// report cloud-init total time
	if totalTimeLine := getCloudInitTotalTimeLine(); totalTimeLine != "" {
		log.Infof("Cloud-init Execution %s", totalTimeLine)
	} else {
		log.Errorf("Failed to report cloud-init total time")
	}
}

func reportKernelBootStats() {
	if exists := systemdAnalyzeExecutableExists(); !exists {
		log.Errorf("systemd-analyze executable not found, kernel boot stats won't be reported!")
		return
	}

	systemdAnalyzeLines, err := readCommand(SystemdAnalyzeExecutable)
	if err != nil {
		log.Errorf("Failed to read systemd-analyze output: %v", err)
		return
	}

	for _, line := range systemdAnalyzeLines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		log.Info("Systemd-analyze output: ", strings.TrimSpace(line))
	}
}

func systemdAnalyzeExecutableExists() bool {
	_, err := readCommand("systemd-analyze", "--version")
	if err != nil {
		log.Errorf("Failed to check if systemd-analyze exists: %v", err)
	}

	return err == nil
}

func cloudInitExecutableExists() bool {
	_, err := readCommand(CloudInitExecutableName, "--version")
	if err != nil {
		log.Errorf("Failed to check if cloud-init exists: %v", err)
	}

	return err == nil
}

func getCloudInitTotalTimeLine() string {
	cloudInitShowLines, err := readCommand(CloudInitExecutableName, "analyze", "show")
	if err != nil {
		return ""
	}

	for _, line := range cloudInitShowLines {
		if strings.Contains(line, "Total Time:") {
			return strings.TrimSpace(line)
		}
	}

	return ""
}

func executeCloudInitAnalyzeBoot() ([]string, error) {
	var errbuf strings.Builder

	cmd := exec.Command(CloudInitExecutableName, "analyze", "boot")
	cmd.Stderr = &errbuf
	out, err := cmd.Output()
	// for unknown reason, cloud-init analyze boot exits with exit code 1 and successful stderr.
	if err != nil && !strings.Contains(errbuf.String(), "successful") {
		return nil, fmt.Errorf("%v; %v", errbuf.String(), err)
	}

	outputLines := strings.SplitAfter(string(out), "\n")
	return outputLines, nil
}

func readCommand(command string, args ...string) ([]string, error) {
	var errbuf strings.Builder

	cmd := exec.Command(command, args...)
	cmd.Stderr = &errbuf
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%v; %v", errbuf.String(), err)
	}

	outputLines := strings.SplitAfter(string(out), "\n")
	return outputLines, nil
}
