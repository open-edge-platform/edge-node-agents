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
func Open(fs afero.Fs, filePath string) (afero.File, error) {
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

func isFilePathAbsolute(filePath string) error {
	// Resolve the absolute path of the file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("error resolving absolute path: %w", err)
	}

	// Ensure the file is within one of the allowed base directories
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
		return fmt.Errorf("access to the file is outside the allowed directories: %s", absPath)
	}
	return nil
}

func isFilePathSymLink(filePath string) error {
	// Issues with Afero not supporting symlink like os package
	// so we need to use os package to check if the file is a symlink
	// and for the unit tests.
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		return fmt.Errorf("error checking file info: %w", err)
	}

	// Check if the file is a symlink
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("file is a symlink, refusing to open: %s", filePath)
	}

	return nil
}
