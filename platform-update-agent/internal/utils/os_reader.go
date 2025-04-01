// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
)

// FileReader defines an interface for reading files.
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// RealFileReader is the implementation of FileReader that reads from the disk.
type RealFileReader struct{}

func (r *RealFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
