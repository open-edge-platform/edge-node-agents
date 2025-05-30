// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

// TestGetPartnerIDSuccess checks that GetPartnerID returns the correct value when the file exists and is non-empty.
func TestGetPartnerIDSuccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "partner_id")
	require.NoError(t, os.WriteFile(file, []byte("partner-xyz"), 0640), "Should write partner_id file with 0640 permissions")
	idt := newTestIdentity(dir)
	id, err := idt.GetPartnerID()
	require.NoError(t, err, "GetPartnerID should not return error for valid file")
	require.Equal(t, "partner-xyz", id, "GetPartnerID should return correct partner ID")
}

// TestGetPartnerIDFileNotExist checks that GetPartnerID returns an error when the file does not exist.
func TestGetPartnerIDFileNotExist(t *testing.T) {
	idt := newTestIdentityCustom("", "", "", "/not/existing/file", "")
	_, err := idt.GetPartnerID()
	require.ErrorContains(t, err, "failed to get partner ID file stat", "GetPartnerID should error if file does not exist")
}

// TestGetPartnerIDEmptyFile checks that GetPartnerID returns an error when the file is empty.
func TestGetPartnerIDEmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "partner_id")
	require.NoError(t, os.WriteFile(file, []byte(""), 0640), "Should write empty partner_id file with 0640 permissions")
	idt := newTestIdentity(dir)
	_, err := idt.GetPartnerID()
	require.ErrorContains(t, err, "partner ID file is empty", "GetPartnerID should error if file is empty")
}

// TestGetPartnerIDReadError checks that GetPartnerID returns an error when the path is a directory (simulates read error).
func TestGetPartnerIDReadError(t *testing.T) {
	// Simulate read error by pointing to a directory instead of a file
	idt := newTestIdentityCustom("", "", "", t.TempDir(), "")
	_, err := idt.GetPartnerID()
	require.ErrorContains(t, err, "failed to read partner ID file", "GetPartnerID should error if file is not readable")
}

// TestCalculateMachineIDSuccess checks that CalculateMachineID returns a valid hash when all system info is available.
func TestCalculateMachineIDSuccess(t *testing.T) {
	// Use testutil to mock command outputs for system and network dependencies
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, []byte("uuid"), nil)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-serial-number"}, []byte("serial"), nil)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, []byte(`[{"serial":"n2"},{"serial":"n1"}]`), nil)

	idt := newTestIdentity("")
	id, err := idt.CalculateMachineID(testutil.TestCmdExecutor)
	require.NoError(t, err, "CalculateMachineID should not return error for valid system info")
	require.Len(t, id, 64, "CalculateMachineID should return a 64-character hash")
}

// TestCalculateMachineIDUUIDError checks that CalculateMachineID returns an error when getting the system UUID fails.
func TestCalculateMachineIDUUIDError(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, nil, errors.New("fail uuid"))

	idt := newTestIdentity("")
	_, err := idt.CalculateMachineID(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get system UUID", "CalculateMachineID should error if system UUID fails")
}

// TestCalculateMachineIDSerialError checks that CalculateMachineID returns an error when getting the serial number fails.
func TestCalculateMachineIDSerialError(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, []byte("uuid"), nil)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-serial-number"}, nil, errors.New("fail serial"))

	idt := newTestIdentity("")
	_, err := idt.CalculateMachineID(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get system serial number", "CalculateMachineID should error if serial number fails")
}

// TestCalculateMachineIDNetworkError checks that CalculateMachineID returns an error when getting network serials fails.
func TestCalculateMachineIDNetworkError(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, []byte("uuid"), nil)
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-serial-number"}, []byte("serial"), nil)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, nil, errors.New("fail net"))

	idt := newTestIdentity("")
	_, err := idt.CalculateMachineID(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get network serials", "CalculateMachineID should error if network serials fail")
}

// TestSaveMachineIDsSuccess checks that SaveMachineIDs writes and reads machine IDs correctly in the happy path.
func TestSaveMachineIDsSuccess(t *testing.T) {
	idt := newTestIdentity(t.TempDir())
	id, err := idt.SaveMachineIDs("hash-abc")
	require.NoError(t, err, "SaveMachineIDs should not return error for valid paths")
	require.Equal(t, "hash-abc", id, "SaveMachineIDs should return correct initial machine ID")
	b, err := os.ReadFile(idt.initialMachineIDFilePath)
	require.NoError(t, err, "Should read initial machine ID file without error")
	require.Equal(t, "hash-abc", string(b), "Initial machine ID file should contain correct value")
	b, err = os.ReadFile(idt.currentMachineIDFilePath)
	require.NoError(t, err, "Should read current machine ID file without error")
	require.Equal(t, "hash-abc", string(b), "Current machine ID file should contain correct value")
}

// TestSaveMachineIDsMkdirAllMetricsError checks that SaveMachineIDs returns an error if metricsPath cannot be created.
func TestSaveMachineIDsMkdirAllMetricsError(t *testing.T) {
	idt := newTestIdentityCustom("", "", "", "", "")
	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to create metrics ID directory", "SaveMachineIDs should error if metricsPath cannot be created")
}

// TestSaveMachineIDsReadInitialError checks that SaveMachineIDs returns an error if reading the initial machine ID fails.
func TestSaveMachineIDsReadInitialError(t *testing.T) {
	idt := newTestIdentity(t.TempDir())
	// Simulate read error by creating a directory instead of a file
	require.NoError(t, os.RemoveAll(idt.initialMachineIDFilePath))
	require.NoError(t, os.Mkdir(idt.initialMachineIDFilePath, 0750))
	// WriteFile will succeed (because file does not exist), but ReadFile will fail (because it's a directory)
	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to read initial machine ID", "Should error if reading initial machine ID fails")
}

// TestSaveMachineIDsMkdirAllMachineIDError checks that SaveMachineIDs returns an error if machineIDPath cannot be created.
func TestSaveMachineIDsMkdirAllMachineIDError(t *testing.T) {
	dir := t.TempDir()
	machineDir := filepath.Join(dir, "machine_id_dir")
	idt := newTestIdentityCustom(
		dir,
		machineDir,
		filepath.Join(dir, "machine_id"),
		filepath.Join(dir, "partner_id"),
		filepath.Join(machineDir, "metrics"),
	)
	require.NoError(t, os.WriteFile(idt.initialMachineIDFilePath, []byte("abc"), 0640), "Should write file with 0640 permissions")

	// Create a file at machineIDPath to ensure MkdirAll fails everywhere
	require.NoError(t, os.WriteFile(machineDir, []byte("not a dir"), 0640), "Should create file at machineIDPath to force MkdirAll error")

	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to create current machine ID directory", "Should error if machineIDPath cannot be created")
}

// TestSaveMachineIDsWriteCurrentError checks that SaveMachineIDs returns an error if writing the current machine ID fails.
func TestSaveMachineIDsWriteCurrentError(t *testing.T) {
	idt := newTestIdentity(t.TempDir())
	require.NoError(t, os.WriteFile(idt.initialMachineIDFilePath, []byte("abc"), 0640))
	// Simulate write error by creating a directory instead of a file
	require.NoError(t, os.RemoveAll(idt.currentMachineIDFilePath))
	require.NoError(t, os.Mkdir(idt.currentMachineIDFilePath, 0750))
	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to write current machine ID", "Should error if writing current machine ID fails")
}

// TestSaveMachineIDsReadInitialMachineIDDirectory simulates the case where initialMachineIDFilePath is a directory,
// so os.Stat succeeds but ReadFile fails (covers the "stat ok, but not a file" branch).
func TestSaveMachineIDsReadInitialMachineIDDirectory(t *testing.T) {
	idt := newTestIdentity(t.TempDir())
	require.NoError(t, os.Mkdir(idt.initialMachineIDFilePath, 0750), "Should create directory at initialMachineIDFilePath to force read error")

	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to read initial machine ID", "Should error if reading initial machine ID fails when path is a directory")
}

// TestSaveMachineIDsStatErrorNotExist simulates the case where os.Stat returns an error that is NOT os.ErrNotExist
// by creating a file at the path where a directory is expected, so os.Stat fails with "not a directory".
func TestSaveMachineIDsStatErrorNotExist(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0750), "Should create subdir before test")
	idt := newTestIdentityCustom(
		dir,
		dir,
		filepath.Join(dir, "subdir", "machine_id"),
		filepath.Join(dir, "partner_id"),
		filepath.Join(dir, "metrics"),
	)

	// Create a file at the parent directory path, so Stat on a child path fails with "not a directory"
	parentDir := filepath.Dir(idt.initialMachineIDFilePath)
	require.NoError(t, os.RemoveAll(parentDir), "Should remove parentDir before test")
	require.NoError(t, os.WriteFile(parentDir, []byte("not a dir"), 0640), "Should create file at parentDir to force stat error")

	_, err := idt.SaveMachineIDs("hash")
	require.ErrorContains(t, err, "failed to get initial machine ID file stat", "Should error if stat fails with error other than os.ErrNotExist")
}

func newTestIdentity(dir string) *Identity {
	return &Identity{
		metricsPath:              dir,
		machineIDPath:            dir,
		initialMachineIDFilePath: filepath.Join(dir, "machine_id"),
		partnerIDFilePath:        filepath.Join(dir, "partner_id"),
		currentMachineIDFilePath: filepath.Join(dir, "metrics"),
	}
}

func newTestIdentityCustom(metricsPath, machineIDPath, initialMachineIDFilePath, partnerIDFilePath, currentMachineIDFilePath string) *Identity {
	return &Identity{
		metricsPath:              metricsPath,
		machineIDPath:            machineIDPath,
		initialMachineIDFilePath: initialMachineIDFilePath,
		partnerIDFilePath:        partnerIDFilePath,
		currentMachineIDFilePath: currentMachineIDFilePath,
	}
}
