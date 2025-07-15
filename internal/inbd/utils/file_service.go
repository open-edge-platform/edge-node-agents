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
	"/var/cache/manageability/repository-tool/sota",
	"/var/intel-manageability",
	"/var/log",
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
		if !strings.HasPrefix(relPath, "..") && relPath != "." {
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

// ReadFile reads a file at the given path using afero and checks for symlinks and canonical paths.
func ReadFile(fs afero.Fs, filePath string) ([]byte, error) {
	if err := isFilePathAbsolute(filePath); err != nil {
		return nil, err
	}
	if err := isFilePathSymLink(filePath); err != nil {
		return nil, err
	}
	return afero.ReadFile(fs, filePath)
}

// WriteFile writes data to a file at the given path using afero and checks for symlinks and canonical paths.
func WriteFile(fs afero.Fs, filePath string, data []byte, perm os.FileMode) error {
	if err := isFilePathAbsolute(filePath); err != nil {
		return err
	}
	if err := isFilePathSymLink(filePath); err != nil {
		return err
	}
	return afero.WriteFile(fs, filePath, data, perm)
}

// CreateTempFile creates a temp file, checks for canonical path and symlinks, and returns the file handle.
func CreateTempFile(fs afero.Fs, dir, pattern string) (*os.File, error) {
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
