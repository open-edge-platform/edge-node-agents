// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package helper

import (
	"bufio"
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logger"
)

var log = logger.Logger
var AgentId string = "N/A"
var Kubectl = "kubectl"

func RunExec(ctx context.Context, asyncFlg bool, args ...string) (bool, string, error) {

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	// Let K3s handle KUBECONFIG through system environment - no explicit setting needed

	// Starting to record output of udevadm monitor
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed creating StdoutPipe : %v\n", err)
		return false, "", err
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		log.Printf("failed starting command: %v\n", err)
		return false, "", err
	}

	//no waiting
	if asyncFlg {
		return true, "", nil
	}

	// Create a scanner to read the output
	scanner := bufio.NewScanner(cmdReader)
	// Read and print each line from the command output
	var output string
	for scanner.Scan() {
		output += scanner.Text() + "\n"
	}

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		log.Errorf("command failed: %s cmd: %s", err, cmd)
		return false, "", err
	}

	return true, output, nil
}

func RunStringCommand(ctx context.Context, _ bool, command string) (string, error) {
	if command == "" {
		return "", errors.New("command is empty")
	}
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	_, output, err := RunExec(ctx, false, cmd.Args...)
	if err != nil {
		log.Errorf("Failed to run string command: %s", err)
		return "", err
	}

	return output, nil
}

func SplitStringAsSections(sectionsKey, bigString string) []string {
	lines := strings.Split(bigString, "\n") // Split string into lines
	var results []string
	var sections string
	for _, line := range lines {
		if strings.HasPrefix(line, sectionsKey) {
			if sections != "" {
				results = append(results, sections)
			}
			sections = line + "\n"
		} else {
			sections += line + "\n"
		}
	}
	//append last data
	if sections != "" {
		results = append(results, sections)
	}

	return results
}

func GetSectionBasedonKey(strSet []string, key string) string {
	result := ""
	for _, set := range strSet {
		lines := strings.Split(set, "\n")
		for _, line := range lines {
			if strings.Contains(line, key) {
				result = set
				break
			}
		}
	}
	if result == "" {
		result = strings.Join(strSet, "\n")
	}
	return result
}
