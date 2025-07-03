package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestGetFreeDiskSpaceInBytes_Success(t *testing.T) {
	// Mock Statfs to return valid data
	mockStatfs := func(path string, stat *unix.Statfs_t) error {
		stat.Bavail = 1000
		stat.Bsize = 4096
		return nil
	}

	// Call GetFreeDiskSpaceInBytes
	freeSpace, err := GetFreeDiskSpaceInBytes("/valid/path", mockStatfs)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000*4096), freeSpace, "Free space calculation is incorrect")
}

func TestGetFreeDiskSpaceInBytes_InvalidPath(t *testing.T) {
	// Mock Statfs to return an error for an invalid path
	mockStatfs := func(path string, stat *unix.Statfs_t) error {
		return fmt.Errorf("invalid path")
	}

	// Call GetFreeDiskSpaceInBytes
	freeSpace, err := GetFreeDiskSpaceInBytes("/invalid/path", mockStatfs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get filesystem stats")
	assert.Equal(t, uint64(0), freeSpace, "Free space should be 0 on error")
}

func TestGetFreeDiskSpaceInBytes_StatfsError(t *testing.T) {
	// Mock Statfs to simulate a system error
	mockStatfs := func(path string, stat *unix.Statfs_t) error {
		return errors.New("mock system error")
	}

	// Call GetFreeDiskSpaceInBytes
	freeSpace, err := GetFreeDiskSpaceInBytes("/error/path", mockStatfs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get filesystem stats")
	assert.Equal(t, uint64(0), freeSpace, "Free space should be 0 on error")
}
