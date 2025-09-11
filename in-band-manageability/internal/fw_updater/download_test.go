/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package fwupdater

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestNewDownloader(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	downloader := NewDownloader(request)

	assert.NotNil(t, downloader)
	assert.Equal(t, request, downloader.request)
	assert.NotNil(t, downloader.isDiskSpaceAvailable)
	assert.NotNil(t, downloader.statfs)
	assert.NotNil(t, downloader.httpClient)
	assert.NotNil(t, downloader.requestCreator)
	assert.NotNil(t, downloader.fs)
}

func TestDownloader_download_Success(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://trusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
		isDiskSpaceAvailable: func(url string,
			readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
			getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error),
			getFileSizeInBytesFunc func(string, string) (int64, error),
			isTokenExpiredFunc func(string) (bool, error),
			fs afero.Fs) (bool, error) {
			return true, nil // Mock sufficient disk space
		},
		downloadFileFunc: func(afero.Fs, string, string, *http.Client,
			func(string, string, io.Reader) (*http.Request, error),
			func(afero.Fs, string, func(string) (bool, error)) (string, error),
			func(string) (bool, error)) error {
			return nil // Mock successful download
		},
	}

	err = downloader.download()
	assert.NoError(t, err)
}

func TestDownloader_download_ConfigLoadError(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://example.com/firmware.bin",
	}

	// Create empty filesystem (no config file)
	fs := afero.NewMemMapFs()

	downloader := &Downloader{
		request: request,
		fs:      fs,
	}

	err := downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading config")
}

func TestDownloader_download_UntrustedRepository(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://untrusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file with different trusted repo
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not in the list of trusted repositories")
}

func TestDownloader_download_DiskSpaceCheckError(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://trusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
		isDiskSpaceAvailable: func(url string,
			readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
			getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error),
			getFileSizeInBytesFunc func(string, string) (int64, error),
			isTokenExpiredFunc func(string) (bool, error),
			fs afero.Fs) (bool, error) {
			return false, errors.New("disk space check failed")
		},
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error checking disk space")
	assert.Contains(t, err.Error(), "disk space check failed")
}

func TestDownloader_download_InsufficientDiskSpace(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://trusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
		isDiskSpaceAvailable: func(url string,
			readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
			getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error),
			getFileSizeInBytesFunc func(string, string) (int64, error),
			isTokenExpiredFunc func(string) (bool, error),
			fs afero.Fs) (bool, error) {
			return false, nil // Insufficient disk space
		},
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient disk space")
}

func TestDownloader_download_DownloadFileError(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://trusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
		isDiskSpaceAvailable: func(string,
			func(afero.Fs, string, func(string) (bool, error)) (string, error),
			func(string, func(string, *unix.Statfs_t) error) (uint64, error),
			func(string, string) (int64, error),
			func(string) (bool, error),
			afero.Fs) (bool, error) {
			return true, nil // Sufficient disk space
		},
		downloadFileFunc: func(afero.Fs, string, string, *http.Client,
			func(string, string, io.Reader) (*http.Request, error),
			func(afero.Fs, string, func(string) (bool, error)) (string, error),
			func(string) (bool, error)) error {
			return errors.New("download failed")
		},
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error downloading the file")
	assert.Contains(t, err.Error(), "download failed")
}

func TestDownloader_download_InvalidConfigJSON(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "http://trusted-repo.com/firmware.bin",
	}

	// Create memory filesystem with invalid config
	fs := afero.NewMemMapFs()

	// Create config directory and file with invalid JSON
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"
		}
	}` // Missing closing bracket
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading config")
}

func TestDownloader_download_EmptyURL(t *testing.T) {
	request := &pb.UpdateFirmwareRequest{
		Url: "", // Empty URL
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": ["http://trusted-repo.com"]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	downloader := &Downloader{
		request: request,
		fs:      fs,
	}

	err = downloader.download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not in the list of trusted repositories")
}

func TestDownloader_download_ComplexScenario(t *testing.T) {
	// Test with a more complex scenario including multiple trusted repositories
	request := &pb.UpdateFirmwareRequest{
		Url:         "https://secure-repo.example.com/path/to/firmware-v2.1.bin",
		Username:    "testuser",
		Signature:   "abc123",
	}

	// Create memory filesystem with mock config
	fs := afero.NewMemMapFs()

	// Create config directory and file with multiple trusted repos
	err := fs.MkdirAll("/etc/intel_manageability", 0755)
	assert.NoError(t, err)

	configContent := `{
		"os_updater": {
			"trustedRepositories": [
				"http://trusted-repo.com",
				"https://secure-repo.example.com",
				"ftp://legacy-repo.company.internal"
			]
		}
	}`
	err = afero.WriteFile(fs, utils.ConfigFilePath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Mock successful execution
	diskSpaceCheckCalled := false
	downloadFileCalled := false

	downloader := &Downloader{
		request: request,
		fs:      fs,
		isDiskSpaceAvailable: func(url string,
			readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
			getFreeDiskSpaceInBytes func(string, func(string, *unix.Statfs_t) error) (uint64, error),
			getFileSizeInBytesFunc func(string, string) (int64, error),
			isTokenExpiredFunc func(string) (bool, error),
			fs afero.Fs) (bool, error) {
			diskSpaceCheckCalled = true
			assert.Equal(t, request.Url, url)
			return true, nil
		},
		downloadFileFunc: func(fs afero.Fs, url string, destDir string, 
			client *http.Client,
			requestCreator func(string, string, io.Reader) (*http.Request, error),
			readJWTTokenFunc func(afero.Fs, string, func(string) (bool, error)) (string, error),
			isTokenExpiredFunc func(string) (bool, error)) error {
			downloadFileCalled = true
			assert.Equal(t, request.Url, url)
			return nil
		},
	}

	err = downloader.download()
	assert.NoError(t, err)
	assert.True(t, diskSpaceCheckCalled, "Disk space check should have been called")
	assert.True(t, downloadFileCalled, "Download file should have been called")
}
