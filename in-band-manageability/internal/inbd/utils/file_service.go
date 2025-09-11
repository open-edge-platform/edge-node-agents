/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// List of allowed base directories
var allowedBaseDirs = []string{
	"/etc",
	"/tmp",
	"/usr/share",
	"/usr/bin",
	"/usr/sbin",
	"/opt",
	"/var/cache/manageability",
	"/var/intel-manageability",
	"/var/log",
	"/sys/class/dmi/id/",
	"/sys/devices/virtual/dmi/id/",
	"/proc",
}

// IsBTRFSFileSystem checks if the filesystem type of the given path is BTRFS.
func IsBTRFSFileSystem(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
	var stat unix.Statfs_t

	// Get filesystem statistics for the given path
	err := statfsFunc(path, &stat)
	if err != nil {
		return false, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// BTRFS filesystem type constant
	const BTRFSFileSystemType = 0x9123683E

	// Check if the filesystem type matches BTRFS
	return stat.Type == BTRFSFileSystemType, nil
}

// RemoveFile removes a file at the given path using the provided filesystem.
// It returns an error if the file is a symlink or if there was an error removing the file.
// The file must be an absolute path and within one of the allowed base directories.
func RemoveFile(fs afero.Fs, filePath string) error {
	if err := isFilePathAbsolute(filePath); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := isFilePathSymLink(filePath); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := fs.Remove(filePath); err != nil {
		return fmt.Errorf("error removing file: %w", err)
	}
	return nil
}

// IsFileExist checks if a file exists at the given path using the provided filesystem.
func IsFileExist(fs afero.Fs, filePath string) bool {
	if exists, err := afero.Exists(fs, filePath); err != nil {
		log.Printf("file does not exist: %s", filePath)
		return false
	} else if !exists {
		log.Printf("file does not exist: %s", filePath)
		return false
	}
	return true
}

// Open opens a file at the given path using the provided filesystem.
// It returns an error if the file is a symlink or if there was an error opening the file.
var Open = func(fs afero.Fs, filePath string) (afero.File, error) {
	if err := isFilePathAbsolute(filePath); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := isFilePathSymLink(filePath); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	file, err := fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	return file, nil
}

// OpenFile opens a file at the given path using the provided flag and permissions.
// It returns an error if the file is a symlink or if there was an error opening the file.
func OpenFile(fs afero.Fs, filePath string, flag int, perm os.FileMode) (afero.File, error) {
	if err := isFilePathAbsolute(filePath); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := isFilePathSymLink(filePath); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	file, err := fs.OpenFile(filePath, flag, os.FileMode(perm))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	return file, nil
}

// CopyFile copies a file from srcPath to destPath using the provided filesystem.
func CopyFile(fs afero.Fs, srcPath string, destPath string) error {
	if err := isFilePathAbsolute(srcPath); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := isFilePathAbsolute(destPath); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := isFilePathSymLink(srcPath); err != nil {
		return fmt.Errorf("%w", err)
	}

	srcFile, err := fs.Open(srcPath)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return nil
}

// MkdirAll creates a directory and all necessary parents securely.
// It checks that the directory path is absolute, within allowed directories, and not a symlink.
func MkdirAll(fs afero.Fs, dirPath string, perm os.FileMode) error {
	if err := isDirPathAbsolute(dirPath); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := isFilePathSymLink(dirPath); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := fs.MkdirAll(dirPath, perm); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	return nil
}

func isFilePathAbsolute(path string) error {
	// Resolve the absolute path (works for both files and directories)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error resolving absolute path: %w", err)
	}

	// Ensure the path is within one of the allowed base directories
	isAllowed := false
	for _, baseDir := range allowedBaseDirs {
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			return fmt.Errorf("error checking relative path: %w", err)
		}
		// Check if the resolved path is within the base directory
		// Allow DMI paths to be directly in the base directory (relPath == ".")
		isDMIPath := strings.HasPrefix(baseDir, "/sys/class/dmi/id/") ||
			strings.HasPrefix(baseDir, "/sys/devices/virtual/dmi/id/")

		if !strings.HasPrefix(relPath, "..") && (relPath != "." || isDMIPath) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return fmt.Errorf("access to the path is outside the allowed directories: %s", absPath)
	}
	return nil
}

// isDirPathAbsolute checks if the directory path is absolute and within allowed base directories.
// Unlike isFilePathAbsolute, this allows the base directory itself to be used.
func isDirPathAbsolute(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("error resolving absolute path: %w", err)
	}
	isAllowed := false
	for _, baseDir := range allowedBaseDirs {
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			return fmt.Errorf("error checking relative path: %w", err)
		}
		// Allow both the base directory itself and its subdirectories
		if !strings.HasPrefix(relPath, "..") {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return fmt.Errorf("access to the directory is outside the allowed directories: %s", absPath)
	}
	return nil
}

// isFilePathSymLink checks for symlinks and canonical paths (works for both files and directories).
func isFilePathSymLink(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error resolving absolute path: %w", err)
	}

	// Allow DMI paths even if they contain symlinks - these are system-controlled and safe
	if strings.HasPrefix(path, "/sys/class/dmi/id/") || strings.HasPrefix(path, "/sys/devices/virtual/dmi/id/") {
		return nil
	}

	// Check if path exists first
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		// For non-existent paths, check if any parent directories contain symlinks
		dir := filepath.Dir(absPath)
		for dir != "/" && dir != "." {
			if _, err := os.Lstat(dir); err == nil {
				// Directory exists, check if it's canonical
				canonDir, err := filepath.EvalSymlinks(dir)
				if err != nil {
					return fmt.Errorf("failed to evaluate symlinks in path: %w", err)
				}
				if dir != canonDir {
					return fmt.Errorf("path contains symlinks: %s", path)
				}
			}
			dir = filepath.Dir(dir)
		}
		return nil // No symlinks found in existing parent directories
	}

	// Path exists, do full canonical path checking
	canonPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("failed to evaluate symlinks: %w", err)
	}

	if absPath != canonPath {
		return fmt.Errorf("path contains symlinks: %s", path)
	}

	return nil
}

// ReadFile reads a file at the given path using afero with TOCTOU protection.
// It opens the file first, then performs security checks on the file descriptor.
func ReadFile(fs afero.Fs, filePath string) ([]byte, error) {
	if err := isFilePathAbsolute(filePath); err != nil {
		return nil, err
	}

	// Open the file first to get a file descriptor
	file, err := fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Perform security checks on the opened file descriptor
	if err := validateOpenedFile(file, filePath); err != nil {
		return nil, err
	}

	// Read from the validated file descriptor
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return data, nil
}

// WriteFile writes data to a file at the given path using afero with TOCTOU protection.
// It opens the file first, then performs security checks on the file descriptor.
func WriteFile(fs afero.Fs, filePath string, data []byte, perm os.FileMode) error {
	if err := isFilePathAbsolute(filePath); err != nil {
		return err
	}

	// Open/create the file first to get a file descriptor
	file, err := fs.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("error opening file for writing: %w", err)
	}
	defer file.Close()

	// Perform security checks on the opened file descriptor
	if err := validateOpenedFile(file, filePath); err != nil {
		return err
	}

	// Write to the validated file descriptor
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

// CreateTempFile creates a temp file, checks for canonical path and symlinks, and returns the file handle.
func CreateTempFile(fs afero.Fs, dir, pattern string) (*os.File, error) {
	// Handle empty directory (uses system default temp dir)
	if dir == "" {
		dir = os.TempDir() // This resolves to /tmp on most systems
	}

	// Validate the directory is within allowed base directories first
	isAllowed := false
	for _, baseDir := range allowedBaseDirs {
		if strings.HasPrefix(dir, baseDir) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return nil, fmt.Errorf("path not allowed: directory %s is outside allowed directories", dir)
	}

	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Set up cleanup - will only execute if we return an error
	var cleanupNeeded = true
	defer func() {
		if cleanupNeeded {
			// Close the file first
			if closeErr := tmpFile.Close(); closeErr != nil {
				log.Printf("Warning: failed to close temp file %s: %v", tmpFile.Name(), closeErr)
			}

			// Then remove it
			if rmErr := fs.Remove(tmpFile.Name()); rmErr != nil {
				log.Printf("Warning: failed to remove temp file %s: %v", tmpFile.Name(), rmErr)
			}
		}
	}()

	// Validate the temp file path
	if err := isFilePathAbsolute(tmpFile.Name()); err != nil {
		return nil, fmt.Errorf("temp file path not allowed: %w", err)
	}

	if err := isFilePathSymLink(tmpFile.Name()); err != nil {
		return nil, fmt.Errorf("temp file is a symlink: %w", err)
	}

	// Success - disable cleanup since we're returning the file
	cleanupNeeded = false
	return tmpFile, nil
}

// validateOpenedFile performs security checks on an already opened file descriptor
// to prevent TOCTOU vulnerabilities. It verifies the file is not a symlink and
// is within allowed directories using the file descriptor's actual path.
func validateOpenedFile(file afero.File, originalPath string) error {
	// Get the real path of the opened file descriptor
	// This works by getting the file info and checking its actual location
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// For files that support it, get the actual path from the file descriptor
	if osFile, ok := file.(*os.File); ok {
		// Get the real path that the file descriptor points to
		fdPath := fmt.Sprintf("/proc/self/fd/%d", osFile.Fd())
		realPath, err := os.Readlink(fdPath)

		if err != nil {
			// Fallback: if we can't read the symlink, use the original path
			// but still perform the mode check
			realPath = originalPath
		}

		// Check if the real path is still within allowed directories
		if err := isFilePathAbsolute(realPath); err != nil {
			return fmt.Errorf("opened file is outside allowed directories: %w", err)
		}

		// Special handling for DMI paths - allow both symlink and real device paths
		isDMIPath := strings.HasPrefix(originalPath, "/sys/class/dmi/id/") ||
			strings.HasPrefix(realPath, "/sys/devices/virtual/dmi/id/")

		if !isDMIPath {
			// Verify the real path matches what we expected (no symlink substitution)
			absOriginal, err := filepath.Abs(originalPath)
			if err != nil {
				return fmt.Errorf("failed to resolve original path: %w", err)
			}

			absReal, err := filepath.Abs(realPath)
			if err != nil {
				return fmt.Errorf("failed to resolve real path: %w", err)
			}

			if absOriginal != absReal {
				return fmt.Errorf("file path was changed via symlink: expected %s, got %s", absOriginal, absReal)
			}
		}
	}

	// Check file mode to ensure it's a regular file (not a device, pipe, etc.)
	mode := fileInfo.Mode()
	if !mode.IsRegular() {
		return fmt.Errorf("file is not a regular file: %s (mode: %s)", originalPath, mode)
	}

	return nil
}
