// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/network"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/system"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

type Provider interface {
	GetGroupID() (string, error)
	CalculateMachineID(executor utils.CmdExecutor) (string, error)
	SaveMachineIDs(machineIDHash string) (initialMachineID string, err error)
}

type Identity struct {
	metricsPath              string
	machineIDPath            string
	initialMachineIDFilePath string
	groupIDFilePath          string
	currentMachineIDFilePath string
}

func (id *Identity) GetGroupID() (string, error) {
	fileStat, err := os.Stat(id.groupIDFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get group ID file stat: %w", err)
	}
	if fileStat.Size() == 0 {
		return "", errors.New("group ID file is empty")
	}

	groupID, err := utils.ReadFileTrimmed(id.groupIDFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read group ID file: %w", err)
	}

	return groupID, nil
}

func (*Identity) CalculateMachineID(executor utils.CmdExecutor) (string, error) {
	systemUUID, err := system.GetSystemUUID(executor)
	if err != nil {
		return "", fmt.Errorf("failed to get system UUID: %w", err)
	}
	systemSerial, err := system.GetSerialNumber(executor)
	if err != nil {
		return "", fmt.Errorf("failed to get system serial number: %w", err)
	}

	networkSerials, err := network.GetNetworkSerials(executor)
	if err != nil {
		return "", fmt.Errorf("failed to get network serials: %w", err)
	}

	// Sort ascending order to ensure consistent hashing
	sort.Strings(networkSerials)

	var builder strings.Builder
	builder.WriteString(systemUUID)
	builder.WriteString(systemSerial)
	for _, serial := range networkSerials {
		builder.WriteString(serial)
	}

	systemIDHash := sha256.Sum256([]byte(builder.String()))
	encodedHash := hex.EncodeToString(systemIDHash[:])
	return encodedHash, nil
}

func (id *Identity) SaveMachineIDs(machineIDHash string) (initialMachineID string, err error) {
	if err = os.MkdirAll(id.metricsPath, 0750); err != nil {
		return "", fmt.Errorf("failed to create metrics ID directory: %w", err)
	}
	initialMachineIDFileStat, err := os.Stat(id.initialMachineIDFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to get initial machine ID file stat: %w", err)
	}

	if errors.Is(err, os.ErrNotExist) || initialMachineIDFileStat.Size() == 0 {
		if err := os.WriteFile(id.initialMachineIDFilePath, []byte(machineIDHash), 0640); err != nil {
			return "", fmt.Errorf("failed to write initial machine ID: %w", err)
		}
	}

	initialMachineID, err = utils.ReadFileTrimmed(id.initialMachineIDFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read initial machine ID: %w", err)
	}

	if err := os.MkdirAll(id.machineIDPath, 0750); err != nil {
		return initialMachineID, fmt.Errorf("failed to create current machine ID directory: %w", err)
	}
	if err := os.WriteFile(id.currentMachineIDFilePath, []byte(machineIDHash), 0640); err != nil {
		return initialMachineID, fmt.Errorf("failed to write current machine ID: %w", err)
	}

	return initialMachineID, nil
}

func NewIdentity() Provider {
	const metricsPath = "/etc/edge-node/metrics"
	const machineIDPath = "/var/lib/edge-node"
	return &Identity{
		metricsPath:              metricsPath,
		machineIDPath:            machineIDPath,
		initialMachineIDFilePath: filepath.Join(metricsPath, "machine_id"),
		groupIDFilePath:          filepath.Join(metricsPath, "group_id"),
		currentMachineIDFilePath: filepath.Join(machineIDPath, "metrics"),
	}
}
