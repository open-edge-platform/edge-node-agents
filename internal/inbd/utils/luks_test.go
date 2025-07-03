/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// cfgForTest returns a Configurations struct with LUKS fields set for testing.
func cfgForTest(volumePath, mapperName, mountPoint string, password []byte, useTPM bool, user string, group string) *Configurations {
	return &Configurations{
		LUKS: struct {
			VolumePath     string `json:"volumePath"`
			MapperName     string `json:"mapperName"`
			MountPoint     string `json:"mountPoint"`
			PasswordLength int    `json:"passwordLength"`
			Size           int    `json:"size"`
			UseTPM         bool   `json:"useTPM"`
			User           string `json:"user"`
			Group          string `json:"group"`
			Password       []byte `json:"password,omitempty"`
		}{
			VolumePath:     volumePath,
			MapperName:     mapperName,
			MountPoint:     mountPoint,
			Password:       password,
			UseTPM:         useTPM,
			User:           user,
			Group:          group,
			PasswordLength: 32, // Default to 32 bytes for test
			Size:           8,  // Default to 8 MB for test
		},
	}
}

func TestCreateLUKSVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that changes system state")
	}

	// Use /tmp for test files and afero.NewOsFs() for real filesystem
	fs := afero.NewOsFs()
	testFile := "/tmp/test/test-luks-volume.img"
	password := []byte("MyStr0ngP@ssw0rd!")
	sizeMB := 32
	useTPM := false

	defer func() {
		if err := RemoveFile(fs, testFile); err != nil {
			t.Logf("cleanup RemoveFile error: %v", err)
		}
	}()

	// Ensure parent directory exists using MkdirAll from file_service.go
	if err := MkdirAll(fs, "/tmp/test", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := CreateLUKSVolume(fs, testFile, password, sizeMB, useTPM); err != nil {
		t.Fatalf("CreateLUKSVolume() error = %v, want nil", err)
	}

	// Verify the file exists using IsFileExist from file_service.go
	if !IsFileExist(fs, testFile) {
		t.Fatalf("LUKS volume file does not exist: %v", testFile)
	}
}

func TestCreateLUKSVolumeWithTPM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that changes system state")
	}

	fs := afero.NewOsFs()
	testFile := "/tmp/test/test-luks-volume-with-tpm.img"
	password := []byte("MyStr0ngP@ssw0rd!")
	sizeMB := 32
	useTPM := true

	defer func() {
		if err := RemoveFile(fs, testFile); err != nil {
			t.Logf("cleanup RemoveFile error: %v", err)
		}
	}()

	if err := MkdirAll(fs, "/tmp/test", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if !isTPMAvailable() {
		t.Skip("Skipping test: TPM not available on this system")
	}

	if err := CreateLUKSVolume(fs, testFile, password, sizeMB, useTPM); err != nil {
		t.Fatalf("Failed to create LUKS volume with TPM: %v", err)
	}

	if !IsFileExist(fs, testFile) {
		t.Fatalf("LUKS volume file does not exist: %v", testFile)
	}
}

func TestAllOpLUKSVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that changes system state")
	}

	fs := afero.NewOsFs()
	testFile := "/tmp/test/test-luks-volume.img"
	password := []byte("MyStr0ngP@ssw0rd!")
	sizeMB := 32
	useTPM := false
	mapperName := "test-luks-mapper"
	mountPoint := "/tmp/test/mnt/test-luks"

	if err := MkdirAll(fs, "/tmp/test", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := MkdirAll(fs, "/tmp/test/mnt", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	cfg := cfgForTest(testFile, mapperName, mountPoint, password, useTPM, "root", "root")
	cfg.LUKS.PasswordLength = 32
	cfg.LUKS.Size = sizeMB

	if err := CreateLUKSVolume(fs, testFile, password, sizeMB, useTPM); err != nil {
		t.Fatalf("CreateLUKSVolume() error = %v, want nil", err)
	}

	if err := OpenLUKSVolume(cfg); err != nil {
		t.Fatalf("OpenLUKSVolume() error = %v, want nil", err)
	}

	if err := FormatLUKSVolume(cfg.LUKS.MapperName); err != nil {
		t.Fatalf("Format LUKS volume error: %v, want nil", err)
	}

	if err := MountLUKSVolume(cfg); err != nil {
		t.Fatalf("MountLUKSVolume() error = %v, want nil", err)
	}

	if err := RemoveLUKSVolume(cfg); err != nil {
		t.Fatalf("RemoveLUKSVolume() error = %v, want nil", err)
	}

	if IsFileExist(fs, testFile) {
		t.Fatalf("LUKS volume file still exists: %v", testFile)
	}
}

// isTPMAvailable checks if the TPM is available on the system.
func isTPMAvailable() bool {
	// Check if TPM is accessible using tpm2-tools
	cmd := exec.Command("tpm2_getcap", "properties-fixed")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func TestCreateSparseFile(t *testing.T) {
	fs := afero.NewOsFs()
	filePath := "/tmp/test/sparsefile.img"
	sizeMB := 32

	parentDir := "/tmp/test"
	if err := MkdirAll(fs, parentDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	_ = RemoveFile(fs, filePath)

	err := createSparseFile(fs, filePath, sizeMB)
	if err != nil {
		t.Fatalf("createSparseFile failed: %v", err)
	}

	if !IsFileExist(fs, filePath) {
		t.Fatalf("File does not exist: %s", filePath)
	}

	fileInfo, err := fs.Stat(filePath)
	if err != nil {
		t.Fatalf("File does not exist or stat failed: %v", err)
	}
	expectedSize := int64(sizeMB) * 1024 * 1024
	if fileInfo.Size() != expectedSize {
		t.Errorf("File size mismatch: expected %d, got %d", expectedSize, fileInfo.Size())
	}
}

func TestCreateSparseFile_InvalidDir(t *testing.T) {
	fs := afero.NewOsFs()
	// Use a path outside allowedBaseDirs to trigger error
	filePath := "/notallowed/sparsefile.img"
	sizeMB := 8
	err := createSparseFile(fs, filePath, sizeMB)
	assert.Error(t, err)
}

func TestCreateSparseFile_InvalidSize(t *testing.T) {
	fs := afero.NewOsFs()
	filePath := "/tmp/test/invalidsize.img"
	// Negative size
	err := createSparseFile(fs, filePath, -1)
	assert.Error(t, err)
}

func TestCreateLUKSVolume_InvalidDir(t *testing.T) {
	fs := afero.NewOsFs()
	// Use a path outside allowedBaseDirs to trigger error
	filePath := "/notallowed/luks.img"
	password := []byte("password")
	sizeMB := 8
	err := CreateLUKSVolume(fs, filePath, password, sizeMB, false)
	assert.Error(t, err)
}

func TestOpenLUKSVolume_InvalidMapper(t *testing.T) {
	cfg := &Configurations{
		LUKS: struct {
			VolumePath     string `json:"volumePath"`
			MapperName     string `json:"mapperName"`
			MountPoint     string `json:"mountPoint"`
			PasswordLength int    `json:"passwordLength"`
			Size           int    `json:"size"`
			UseTPM         bool   `json:"useTPM"`
			User           string `json:"user"`
			Group          string `json:"group"`
			Password       []byte `json:"password,omitempty"`
		}{
			VolumePath:     "/tmp/test/nonexistent.img",
			MapperName:     "nonexistent-mapper",
			MountPoint:     "/tmp/test/mnt/nonexistent",
			PasswordLength: 16,
			Size:           8,
			UseTPM:         false,
			User:           "root",
			Group:          "root",
			Password:       []byte("badpass"),
		},
	}
	err := OpenLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestFormatLUKSVolume_InvalidMapper(t *testing.T) {
	err := FormatLUKSVolume("nonexistent-mapper")
	assert.Error(t, err)
}

func TestMountLUKSVolume_InvalidDevice(t *testing.T) {
	cfg := &Configurations{
		LUKS: struct {
			VolumePath     string `json:"volumePath"`
			MapperName     string `json:"mapperName"`
			MountPoint     string `json:"mountPoint"`
			PasswordLength int    `json:"passwordLength"`
			Size           int    `json:"size"`
			UseTPM         bool   `json:"useTPM"`
			User           string `json:"user"`
			Group          string `json:"group"`
			Password       []byte `json:"password,omitempty"`
		}{
			VolumePath:     "/tmp/test/nonexistent.img",
			MapperName:     "nonexistent-mapper",
			MountPoint:     "/tmp/test/mnt/nonexistent",
			PasswordLength: 16,
			Size:           8,
			UseTPM:         false,
			User:           "root",
			Group:          "root",
			Password:       []byte("badpass"),
		},
	}
	err := MountLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestRemoveLUKSVolume_InvalidMount(t *testing.T) {
	cfg := &Configurations{
		LUKS: struct {
			VolumePath     string `json:"volumePath"`
			MapperName     string `json:"mapperName"`
			MountPoint     string `json:"mountPoint"`
			PasswordLength int    `json:"passwordLength"`
			Size           int    `json:"size"`
			UseTPM         bool   `json:"useTPM"`
			User           string `json:"user"`
			Group          string `json:"group"`
			Password       []byte `json:"password,omitempty"`
		}{
			VolumePath:     "/tmp/test/nonexistent.img",
			MapperName:     "nonexistent-mapper",
			MountPoint:     "/tmp/test/mnt/nonexistent",
			PasswordLength: 16,
			Size:           8,
			UseTPM:         false,
			User:           "root",
			Group:          "root",
			Password:       []byte("badpass"),
		},
	}
	err := RemoveLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestUnmountAndCloseLUKSVolume_NilConfig(t *testing.T) {
	err := UnmountAndCloseLUKSVolume(nil)
	assert.Error(t, err)
}

func TestSetupLUKSVolume_NilConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := SetupLUKSVolume(fs, nil)
	assert.Error(t, err)
}

func TestGenerateLUKSKey_InvalidLength(t *testing.T) {
	_, err := GenerateLUKSKey(4)
	assert.Error(t, err)
}

func TestSetupLUKSVolume_TPMUnavailable(t *testing.T) {
	// Since IsTPM2Available cannot be mocked, simulate by removing TPM2Device
	fs := afero.NewMemMapFs()
	_ = RemoveFile(fs, TPM2Device)
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt", []byte("pass"), true, "root", "root")
	errSetup := SetupLUKSVolume(fs, cfg)
	assert.Error(t, errSetup)
}

func TestSetupLUKSVolume_GenerateKeyFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt", []byte("pass"), false, "root", "root")
	// Simulate failure by passing an invalid key length
	cfg.LUKS.PasswordLength = 4 // less than minKeyLength in GenerateLUKSKey
	err := SetupLUKSVolume(fs, cfg)
	assert.Error(t, err)
}

func TestUnmountAndCloseLUKSVolume_UnmountFail(t *testing.T) {
	// Since UnmountLUKSVolume cannot be mocked, simulate failure by passing a mount point that will fail
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt/nonexistent", []byte("pass"), false, "root", "root")
	err := UnmountAndCloseLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestUnmountAndCloseLUKSVolume_CloseFail(t *testing.T) {
	// Since CloseLUKSVolume cannot be mocked, simulate failure by passing a non-existent mapper name
	cfg := cfgForTest("/tmp/test.img", "nonexistent-mapper", "/tmp/test/mnt", []byte("pass"), false, "root", "root")
	err := UnmountAndCloseLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestCreateLUKSVolume_InvalidSize(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := CreateLUKSVolume(fs, "/tmp/test.img", []byte("pass"), 0, false)
	assert.Error(t, err)
	err = CreateLUKSVolume(fs, "/tmp/test.img", []byte("pass"), 1000, false)
	assert.Error(t, err)
}

func TestCreateLUKSVolume_CreateSparseFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Simulate failure by passing a file path in a directory that cannot be created
	filePath := "/notallowed/test.img"
	err := CreateLUKSVolume(fs, filePath, []byte("pass"), 8, false)
	assert.Error(t, err)
}

func TestCreateLUKSVolume_TPMFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Simulate TPM failure by passing an invalid password length (storePasswordInTPM will reject)
	invalidPassword := make([]byte, 65) // > 64 bytes, will trigger error in storePasswordInTPM
	err := CreateLUKSVolume(fs, "/tmp/test.img", invalidPassword, 8, true)
	assert.Error(t, err)
}

func TestCreateLUKSVolume_LUKSFormatFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Simulate luksFormat failure by passing a file path in a directory that cannot be created
	filePath := "/notallowed/test.img"
	err := CreateLUKSVolume(fs, filePath, []byte("pass"), 8, false)
	assert.Error(t, err)
}

func TestOpenLUKSVolume_AlreadyMapped(t *testing.T) {
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt", []byte("pass"), false, "root", "root")
	// Create fake /dev/mapper/mapper
	mapperPath := MapperDevicePrefix + cfg.LUKS.MapperName
	fs := afero.NewOsFs()
	_ = MkdirAll(fs, filepath.Dir(mapperPath), 0755)
	file, err := OpenFile(fs, mapperPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Skipf("Cannot create fake mapper device: %v", err)
		return
	}
	file.Close()
	defer func() {
		if err := RemoveFile(fs, mapperPath); err != nil {
			t.Logf("cleanup RemoveFile error: %v", err)
		}
	}()
	// Patch os.Stat to avoid nil pointer dereference if possible (not trivial in Go), so just skip if not running as root or /dev/mapper is not writable.
	if _, statErr := os.Stat(mapperPath); statErr != nil {
		t.Skipf("Cannot stat fake mapper device: %v", statErr)
		return
	}
	err = OpenLUKSVolume(cfg)
	assert.Error(t, err) // Should fail to close mapping (cryptsetup not mocked)
}

func TestOpenLUKSVolume_TPMFail(t *testing.T) {
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt", []byte("pass"), true, "root", "root")
	// Use an invalid NV index to force retrievePasswordFromTPM to fail
	cfg.LUKS.PasswordLength = 32
	err := OpenLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestRemoveLUKSVolume_AllFail(t *testing.T) {
	// Since RemoveLUKSVolume cannot be mocked, simulate failure by passing invalid paths and a TPM-enabled config
	cfg := cfgForTest("/notallowed/test.img", "mapper", "/notallowed/mnt", []byte("pass"), true, "root", "root")
	err := RemoveLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestMountLUKSVolume_UserGroupMissing(t *testing.T) {
	cfg := cfgForTest("/tmp/test.img", "mapper", "/tmp/mnt", []byte("pass"), false, "", "")
	err := MountLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestMountLUKSVolume_MkdirFail(t *testing.T) {
	cfg := cfgForTest("/notallowed/test.img", "mapper", "/notallowed/mnt", []byte("pass"), false, "root", "root")
	// MkdirAll will fail due to notallowed path
	err := MountLUKSVolume(cfg)
	assert.Error(t, err)
}

func TestUnmountLUKSVolume_LazySuccess(t *testing.T) {
	// Try to unmount a non-existent mount point, should fallback to lazy unmount and fail
	err := UnmountLUKSVolume("/tmp/nonexistent")
	assert.Error(t, err)
}

// The following tests cannot mock package-level functions that are not variables.
// Instead, test error paths by passing arguments that will cause the functions to fail.

func TestCreateSparseFile_OpenFileFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/notallowed/openfail.img"
	sizeMB := 8
	// Directory does not exist and is not allowed, so OpenFile will fail
	err := createSparseFile(fs, filePath, sizeMB)
	assert.Error(t, err)
}

// Test luksFormat error path
func TestLuksFormat_TmpFileFail(t *testing.T) {
	// Directly invoke luksFormat with a path that will fail (e.g., directory does not exist)
	err := luksFormat("/nonexistentdir/test.img", []byte("pass"))
	assert.Error(t, err)
}

func TestLuksFormat_WriteFail(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := "/tmp/luks-test-writefail"
	_ = MkdirAll(fs, tmpDir, 0700)
	tmpFile := tmpDir + "/readonly.img"
	f, _ := OpenFile(fs, tmpFile, os.O_CREATE|os.O_RDWR, 0600)
	f.Close()
	_ = os.Chmod(tmpFile, 0400) // read-only

	err := luksFormat(tmpFile, []byte("pass"))
	assert.Error(t, err)

	_ = RemoveFile(fs, tmpFile)
	_ = RemoveFile(fs, tmpDir)
}

// Test storePasswordInTPM error paths
func TestStorePasswordInTPM_InvalidLen(t *testing.T) {
	err := storePasswordInTPM([]byte{}, DefaultNVIndex)
	assert.Error(t, err)
	err = storePasswordInTPM(make([]byte, 65), DefaultNVIndex)
	assert.Error(t, err)
}

// Test removePasswordFromTPM error path
func TestRemovePasswordFromTPM_Error(t *testing.T) {
	// This will fail unless tpm2_nvundefine is present and NV index exists
	err := removePasswordFromTPM("0xdeadbeef")
	assert.Error(t, err)
}

// Test retrievePasswordFromTPM error path
func TestRetrievePasswordFromTPM_Error(t *testing.T) {
	_, err := retrievePasswordFromTPM("0xdeadbeef", 32)
	assert.Error(t, err)
}

// Test GenerateLUKSKey error path
func TestGenerateLUKSKey_TooShort(t *testing.T) {
	_, err := GenerateLUKSKey(4)
	assert.Error(t, err)
}
