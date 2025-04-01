// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
)

func OpenNoLinks(path string) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDONLY, 0)
}

func CreateNoLinks(path string, perm os.FileMode) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
}

func CreateExcl(path string, perm os.FileMode) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
}

func ReadFileNoLinks(path string) ([]byte, error) {
	f, err := OpenNoLinks(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func isHardLink(path string) (bool, error) {
	var stat syscall.Stat_t

	err := syscall.Stat(path, &stat)
	if err != nil {
		return false, err
	}

	if stat.Nlink > 1 {
		return true, nil
	}

	return false, nil
}

func openFileNoLinks(path string, flags int, perm os.FileMode) (*os.File, error) {
	// O_NOFOLLOW - If the trailing component (i.e., basename) of pathname is a symbolic link,
	// then the open fails, with the error ELOOP.
	file, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW, perm)
	if err != nil {
		return nil, err
	}

	hardLink, err := isHardLink(path)
	if err != nil {
		file.Close()
		return nil, err
	}

	if hardLink {
		file.Close()
		return nil, fmt.Errorf("%v is a hardlink", path)
	}

	return file, nil
}
