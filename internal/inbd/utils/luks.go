/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

const (
	DefaultNVIndex     = "0x1500016"
	TPM2Device         = "/dev/tpmrm0" // Device file for TPM 2.0
	CrypttabFilePath   = "/etc/crypttab"
	FstabFilePath      = "/etc/fstab"
	MapperDevicePrefix = "/dev/mapper/"
)

// IsTPM2Available determines if TPM 2.0 is available on the system.
func IsTPM2Available() (bool, error) {
	// Use IsFileExist from file_service.go for TPM device check
	fs := afero.NewOsFs()
	if !IsFileExist(fs, TPM2Device) {
		return false, nil // TPM 2.0 is not available
	}
	return true, nil // TPM 2.0 is available
}

// SetupLUKSVolume sets up and mounts a new LUKS volume
func SetupLUKSVolume(fs afero.Fs, cfg *Configurations) error {
	if cfg == nil {
		return fmt.Errorf("LUKS configuration is nil")
	}

	if cfg.LUKS.UseTPM {
		isTPM2Available, err := IsTPM2Available()
		if err != nil {
			return fmt.Errorf("error checking TPM 2.0 availability: %w", err)
		}
		if !isTPM2Available {
			return fmt.Errorf("TPM 2.0 not available on this system, reconfigure to use keyfile")
		}
	}

	// Generate high entropy password
	password, err := GenerateLUKSKey(cfg.LUKS.PasswordLength)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}
	cfg.LUKS.Password = password

	log.Printf("Creating LUKS volume ...")
	if err := CreateLUKSVolume(fs, cfg.LUKS.VolumePath, password, cfg.LUKS.Size, cfg.LUKS.UseTPM); err != nil {
		return fmt.Errorf("failed to create LUKS volume: %w", err)
	}

	log.Printf("Opening LUKS volume ...")
	if err := OpenLUKSVolume(cfg); err != nil {
		return fmt.Errorf("failed to open LUKS volume: %w", err)
	}

	log.Printf("Formatting LUKS volume ...")
	if err := FormatLUKSVolume(cfg.LUKS.MapperName); err != nil {
		return fmt.Errorf("failed to format LUKS volume: %w", err)
	}

	log.Printf("Mounting LUKS volume ...")
	if err := MountLUKSVolume(cfg); err != nil {
		return fmt.Errorf("failed to mount LUKS volume: %w", err)
	}

	return nil
}

// UnmountAndCloseLUKSVolume unmounts and closes the LUKS volume.
func UnmountAndCloseLUKSVolume(cfg *Configurations) error {
	if cfg == nil {
		return fmt.Errorf("LUKS configuration is nil")
	}

	log.Printf("Unmounting LUKS volume...")
	if err := UnmountLUKSVolume(cfg.LUKS.MountPoint); err != nil {
		return fmt.Errorf("failed to unmount LUKS volume: %w", err)
	}

	log.Printf("Closing LUKS volume...")
	if err := CloseLUKSVolume(cfg.LUKS.MapperName); err != nil {
		return fmt.Errorf("failed to close LUKS volume: %w", err)
	}

	return nil
}

// CreateLUKSVolume set up a new LUKS volume with the specified size and password
func CreateLUKSVolume(fs afero.Fs, filePath string, password []byte, sizeMB int, useTPM bool) error {
	if sizeMB < 1 || sizeMB > 64 {
		return fmt.Errorf("size must be between 1MB and 64MB")
	}

	// Create a sparse file of the specified size
	if err := createSparseFile(fs, filePath, sizeMB); err != nil {
		return fmt.Errorf("failed to create sparse file: %w", err)
	}

	// Optionally store the password in the TPM
	if useTPM {
		// Remove the password from the TPM if it already exists
		if err := removePasswordFromTPM(DefaultNVIndex); err != nil {
			log.Printf("failed to remove existing password from TPM: %s", err)
		}
		if err := storePasswordInTPM(password, DefaultNVIndex); err != nil {
			return fmt.Errorf("failed to store password in TPM: %w", err)
		}
	}

	// Format the file as a LUKS volume
	if err := luksFormat(filePath, password); err != nil {
		return fmt.Errorf("failed to format LUKS volume: %w", err)
	}

	return nil
}

// OpenLUKSVolume opens an existing LUKS volume
func OpenLUKSVolume(cfg *Configurations) error {
	mappedDevice := MapperDevicePrefix + cfg.LUKS.MapperName

	// Check if the mapping already exists
	if _, err := os.Stat(mappedDevice); err == nil {
		// If the device exists, close it first
		cmd := exec.Command("cryptsetup", "luksClose", cfg.LUKS.MapperName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to close existing mapping: err: %w, output:%s", err, string(output))
		}
	}

	if cfg.LUKS.UseTPM {
		// Retrieve the password from the TPM
		password, err := retrievePasswordFromTPM(DefaultNVIndex, cfg.LUKS.PasswordLength)
		if err != nil {
			return fmt.Errorf("failed to retrieve password from TPM: %w", err)
		}
		cfg.LUKS.Password = password
	}

	log.Printf("DEBUG: executing command cryptsetup luksOpen %s %s", cfg.LUKS.VolumePath, cfg.LUKS.MapperName)
	cmd := exec.Command("cryptsetup", "luksOpen", cfg.LUKS.VolumePath, cfg.LUKS.MapperName)
	cmd.Stdin = createPasswordInput(cfg.LUKS.Password, true)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup luksOpen command failed: err: %w, output:%s", err, string(output))
	}

	return nil
}

// FormatLUKSVolume formats an existing LUKS volume
func FormatLUKSVolume(mapperName string) error {
	devicePath := MapperDevicePrefix + mapperName
	cmd := exec.Command("mkfs.ext4", devicePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to format LUKS volume at %s, err: %w, output:%s", devicePath, err, string(output))
	}

	log.Printf("Successfully formatted LUKS volume at %s", devicePath)
	return nil
}

// RemoveLUKSVolume unmounts and closes the LUKS volume and removes the mount point
func RemoveLUKSVolume(cfg *Configurations) error {
	var errs []string

	log.Printf("Unmounting LUKS volume...")
	if err := UnmountLUKSVolume(cfg.LUKS.MountPoint); err != nil {
		errs = append(errs, fmt.Sprintf("failed to unmount LUKS volume: %v", err))
	}

	log.Printf("Closing LUKS volume...")
	if err := CloseLUKSVolume(cfg.LUKS.MapperName); err != nil {
		errs = append(errs, fmt.Sprintf("failed to close LUKS volume: %v", err))
	}

	log.Printf("Removing mount directory...")
	fs := afero.NewOsFs()
	if err := RemoveFile(fs, cfg.LUKS.MountPoint); err != nil {
		errs = append(errs, fmt.Sprintf("failed to remove mount directory: %v", err))
	}

	log.Printf("Removing LUKS image file ...")
	if err := RemoveFile(fs, cfg.LUKS.VolumePath); err != nil {
		errs = append(errs, fmt.Sprintf("failed to remove LUKS image file: %v", err))
	}

	if cfg.LUKS.UseTPM {
		log.Printf("Removing password from TPM ...")
		if err := removePasswordFromTPM(DefaultNVIndex); err != nil {
			errs = append(errs, fmt.Sprintf("failed to remove password from TPM: %v", err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// MountLUKSVolume mounts the mapped LUKS volume to the specified mount point
func MountLUKSVolume(cfg *Configurations) error {
	devicePath := MapperDevicePrefix + cfg.LUKS.MapperName
	fs := afero.NewOsFs()
	if err := MkdirAll(fs, cfg.LUKS.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	log.Println("DEBUG: mounting " + devicePath + " on " + cfg.LUKS.MountPoint)
	cmd := exec.Command("mount", devicePath, cfg.LUKS.MountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount LUKS volume: %s", output)
	}

	// Change ownership of the mount point
	if cfg.LUKS.User == "" || cfg.LUKS.Group == "" {
		return fmt.Errorf("user and group must be specified")
	}
	cmd = exec.Command("chown", fmt.Sprintf("%s:%s", cfg.LUKS.User, cfg.LUKS.Group), cfg.LUKS.MountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to change ownership of mount point: err: %w, output:%s", err, string(output))
	}

	return nil
}

// UnmountLUKSVolume unmounts the mapped LUKS volume
func UnmountLUKSVolume(mountPoint string) error {
	cmd := exec.Command("umount", mountPoint)
	_, err := cmd.CombinedOutput()
	if err != nil {
		// Retry with lazy unmount
		log.Printf("Normal unmount failed: %v. Retrying with lazy unmount...", err)
		cmd = exec.Command("umount", "-l", mountPoint)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to unmount LUKS volume: err: %w, output:%s", err, string(output))
		}
	}
	return nil
}

// CloseLUKSVolume closes the mapped LUKS volume
func CloseLUKSVolume(mapperName string) error {
	cmd := exec.Command("cryptsetup", "luksClose", mapperName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to close LUKS volume: err: %w, output:%s", err, string(output))
	}
	return nil
}

// createSparseFile creates a sparse file of the specified size in MB
func createSparseFile(fs afero.Fs, filePath string, sizeMB int) error {
	// Extract the directory path
	dir := filepath.Dir(filePath)

	// Ensure the directory exists using secure MkdirAll from file_service.go
	if err := MkdirAll(fs, dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the file with secure permissions using OpenFile from file_service.go
	file, err := OpenFile(fs, filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Calculate the size in bytes
	sizeBytes := int64(sizeMB) * 1024 * 1024

	// Truncate the file to the desired size (creates a sparse file)
	if err := file.Truncate(sizeBytes); err != nil {
		return fmt.Errorf("failed to truncate file %s: %w", filePath, err)
	}

	return nil
}

// luksFormat formats the file as a LUKS volume
func luksFormat(filePath string, password []byte) error {
	// Create a temporary file to store the password using afero.TempFile
	fs := afero.NewOsFs()
	tmpFile, err := afero.TempFile(fs, "", "luks-password-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	// Ensure the file is removed after use, check error
	defer func() {
		if rmErr := fs.Remove(tmpFile.Name()); rmErr != nil {
			log.Printf("failed to remove temporary file %s: %v", tmpFile.Name(), rmErr)
		}
	}()

	// Write the password to the temporary file
	if _, err := tmpFile.Write(password); err != nil {
		return fmt.Errorf("failed to write password to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	cmd := exec.Command(
		"cryptsetup",
		"luksFormat",
		"--type=luks2",
		"--batch-mode",
		"--pbkdf-memory=2097152",
		"--pbkdf-parallel=8",
		"--cipher=aes-xts-plain64",
		"--key-file", tmpFile.Name(),
		filePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to format LUKS volume: %w, output: %s", err, string(output))
	}
	return nil
}

// createPasswordInput creates an *os.File to provide the password as input to a command.
// It writes the password (with optional newline) in a goroutine and returns the read end of the pipe.
func createPasswordInput(password []byte, addNewline bool) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatalf("Unable to create pipe: %s", err)
	}

	go func(pw []byte, newline bool, w *os.File) {
		defer w.Close()
		if newline {
			pw = append(pw, '\n')
		}
		if _, err := w.Write(pw); err != nil {
			log.Printf("Unable to write password to pipe: %s", err)
		}
	}(password, addNewline, w)

	return r
}

// storePasswordInTPM stores the LUKS password securely in the TPM.
func storePasswordInTPM(password []byte, nvIndex string) error {
	// Validate password length
	if len(password) < 1 || len(password) > 64 {
		return fmt.Errorf("password length (%d bytes) must be between 1 and 64 bytes", len(password))
	}

	// Define the NV index with the password length as the size
	cmd := exec.Command("tpm2_nvdefine",
		nvIndex,
		fmt.Sprintf("--size=%d", len(password)),
		"--attributes=ownerread|ownerwrite|authread|authwrite")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tpm2_nvdefine error: %w,  output: %s", err, string(output))
	}

	// Write the password to the NV index
	cmd = exec.Command("tpm2_nvwrite",
		nvIndex,
		"--input=-") // Use stdin for the input
	cmd.Stdin = createPasswordInput(password, false)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tpm2_nvwrite error: err: %w, output:%s", err, string(output))
	}

	return nil
}

// removePasswordFromTPM removes the LUKS password from the specified NV index in the TPM.
func removePasswordFromTPM(nvIndex string) error {
	cmd := exec.Command("tpm2_nvundefine", nvIndex)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore error if NV index does not exist
		if strings.Contains(string(output), "the handle is not correct for the use") ||
			strings.Contains(string(output), "Failed to read the public part of NV index") {
			return nil
		}
		return fmt.Errorf("tpm2_nvundefine error: err: %w, output:%s", err, string(output))
	}
	return nil
}

// retrievePasswordFromTPM retrieves the LUKS password from the TPM for the specified NV index and size.
func retrievePasswordFromTPM(nvindex string, size int) ([]byte, error) {
	// Construct the tpm2_nvread command with the provided NV index and size
	cmd := exec.Command("tpm2_nvread", nvindex, fmt.Sprintf("--size=%d", size))

	// Execute the command and capture the output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tpm2_nvread error for index %s: %w", nvindex, err)
	}

	// Return the output as a string
	return output, nil
}

// GenerateLUKSKey generates a random key of the specified length in bytes,
// using tpm2_getrandom if available, otherwise falling back to crypto/rand.
func GenerateLUKSKey(length int) ([]byte, error) {
	const minKeyLength = 8

	if length <= minKeyLength {
		return nil, fmt.Errorf("key length must be greater than %d", minKeyLength)
	}

	// Check if tpm2_getrandom is available.
	isTPMAvailable, err := IsTPM2Available()
	if err != nil {
		log.Printf("Error when checking TPM device: %v", err)
	} else if isTPMAvailable {
		key, err := getRandomBytesFromTPM2(length)
		if err == nil {
			return key, nil
		}
		log.Printf("Failed to use TPM: %v. Falling back to crypto/rand.", err)
	}

	// Fallback to crypto/rand.
	key := make([]byte, length)
	_, err = rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key using crypto/rand: %w", err)
	}

	log.Printf("Generated key of length %d successfully.", length)
	return key, nil
}

// getRandomBytesFromTPM2 fetches the specified number of random bytes using tpm2_getrandom.
func getRandomBytesFromTPM2(size int) ([]byte, error) {
	// Execute the tpm2_getrandom command to fetch `size` bytes in hex format.
	cmd := exec.Command("tpm2_getrandom", fmt.Sprintf("%d", size), "--hex")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute tpm2_getrandom: %w", err)
	}

	// Parse the output as a hex string.
	trimmedOutput := strings.TrimSpace(out.String())
	randomBytes, err := hex.DecodeString(trimmedOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tpm2_getrandom output: %w", err)
	}

	return randomBytes, nil
}
