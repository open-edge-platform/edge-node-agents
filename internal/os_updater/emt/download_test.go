/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package emt provides the implementation for updating the EMT OS.
package emt

import (
	"errors"
	"io"
	"net/http"

	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
)

func TestDownloader_downloadFile(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		downloader := &Downloader{
			fs: fs,
			request: &pb.UpdateSystemSoftwareRequest{
				Url: "http://example.com/file.txt",
			},
			readJWTTokenFunc: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			httpClient: &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("file content")),
					}
				}),
			},
			requestCreator: http.NewRequest,
		}

		err := downloader.downloadFile()
		assert.NoError(t, err)

		exists, err := afero.Exists(fs, DownloadDir+"/file.txt")
		assert.NoError(t, err)
		assert.True(t, exists)

		content, err := afero.ReadFile(fs, DownloadDir+"/file.txt")
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	// t.Run("error loading config", func(t *testing.T) {
	// 	downloader := &EMTDownloader{
	// 		request: &pb.UpdateSystemSoftwareRequest{
	// 			Url: "http://example.com/file.txt",
	// 		},
	// 	}

	// 	LoadConfig = func(path string) (Config, error) {
	// 		return Config{}, errors.New("config error")
	// 	}

	// 	err := downloader.Download()
	// 	assert.EqualError(t, err, "error loading config: config error")
	// })

	t.Run("error creating request", func(t *testing.T) {
		downloader := &Downloader{
			request: &pb.UpdateSystemSoftwareRequest{
				Url: "http://example.com/file.txt",
			},
			readJWTTokenFunc: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			httpClient: &http.Client{},
			requestCreator: func(method, url string, body io.Reader) (*http.Request, error) {
				return nil, errors.New("some error")
			},
		}

		err := downloader.downloadFile()
		assert.EqualError(t, err, "error creating request: some error")
	})

	t.Run("error reading JWT token", func(t *testing.T) {
		downloader := &Downloader{
			request: &pb.UpdateSystemSoftwareRequest{
				Url: "http://example.com/file.txt",
			},
			readJWTTokenFunc: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "", errors.New("error")
			},
			httpClient:     &http.Client{},
			requestCreator: http.NewRequest,
		}

		err := downloader.downloadFile()
		assert.EqualError(t, err, "error reading JWT token: error")
	})

	t.Run("error performing request", func(t *testing.T) {
		downloader := &Downloader{
			request: &pb.UpdateSystemSoftwareRequest{
				Url: "http://example.com/file.txt",
			},
			readJWTTokenFunc: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			httpClient: &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 500,
						Header:     http.Header{"Content-Length": []string{"4096001"}},
						Body:       http.NoBody,
					}
				}),
			},
			requestCreator: http.NewRequest,
		}

		err := downloader.downloadFile()
		assert.EqualError(t, err, "Status code: 500. Expected 200/Success.")
	})

	// TODO:  This one runs in IDE, but not in Earthly
	// t.Run("error creating file", func(t *testing.T) {
	// 	fs := afero.NewBasePathFs(afero.NewOsFs(), downloadDir)
	// 	downloader := &EMTDownloader{
	// 		fs: fs,
	// 		request: &pb.UpdateSystemSoftwareRequest{
	// 			Url: "http://example.com/file.txt",
	// 		},
	// 		readJWTTokenFunc: func(afero.Afero, string) (string, error) {
	// 			return "valid-token", nil
	// 		},
	// 		httpClient: &http.Client{
	// 			Transport: roundTripperFunc(func(req *http.Request) *http.Response {
	// 				return &http.Response{
	// 					StatusCode: 200,
	// 					Header:     http.Header{"Content-Length": []string{"4096"}},
	// 					Body:       io.NopCloser(strings.NewReader("file content")),
	// 				}
	// 			}),
	// 		},
	// 		requestCreator: http.NewRequest,
	// 	}

	// 	err := downloader.downloadFile()
	// 	assert.Error(t, err)
	// 	assert.Contains(t, err.Error(), "permission denied")
	// })

	t.Run("error copying response body", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		downloader := &Downloader{
			fs: fs,
			request: &pb.UpdateSystemSoftwareRequest{
				Url: "http://example.com/file.txt",
			},
			readJWTTokenFunc: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			httpClient: &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(errReader{}),
					}
				}),
			},
			requestCreator: http.NewRequest,
		}

		err := downloader.downloadFile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error copying response body")
	})
}

func TestDownloader_isDiskSpaceAvailable(t *testing.T) {
	tests := []struct {
		name                    string
		readJWTToken            func(afero.Fs, string, func(string) (bool, error)) (string, error)
		writeUpdateStatus       func(afero.Fs, string, string, string)
		writeGranularLog        func(string, string)
		expectedResult          bool
		expectedError           error
		getFreeDiskSpaceInBytes func(string) (uint64, error)
		getFileSizeInBytes      func(string, string) (int64, error)
	}{
		{
			name: "successful check with enough disk space",
			getFreeDiskSpaceInBytes: func(path string) (uint64, error) {
				return 1000 * 4096, nil
			},
			readJWTToken: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			getFileSizeInBytes: func(string, string) (int64, error) {
				return 1000 * 2048, nil
			},
			expectedResult: true,
			expectedError:  nil,
		},
		{
			name: "error getting disk space",
			readJWTToken: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "", nil
			},
			getFreeDiskSpaceInBytes: func(path string) (uint64, error) {
				return 0, errors.New("disk space error")
			},
			expectedResult: false,
			expectedError:  errors.New("disk space error"),
		},
		{
			name: "error reading JWT token",
			getFreeDiskSpaceInBytes: func(path string) (uint64, error) {
				return 1000 * 4096, nil
			},
			readJWTToken: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "", errors.New("token error")
			},
			writeUpdateStatus: func(afero.Fs, string, string, string) {
				// No-op implementation for testing
			},
			writeGranularLog: func(level, message string) {
				// No-op implementation for testing
			},
			expectedResult: false,
			expectedError:  errors.New("error reading JWT token: token error"),
		},
		{
			name: "error getting file size",
			getFreeDiskSpaceInBytes: func(path string) (uint64, error) {
				return 1000 * 4096, nil
			},
			readJWTToken: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			writeUpdateStatus: func(afero.Fs, string, string, string) {
				// No-op implementation for testing
			},
			writeGranularLog: func(level, message string) {
				// No-op implementation for testing
			},
			getFileSizeInBytes: func(string, string) (int64, error) {
				return 0, errors.New("error getting file size")
			},
			expectedResult: false,
			expectedError:  errors.New("error getting file size: error getting file size"),
		},
		{
			name: "not enough disk space",
			getFreeDiskSpaceInBytes: func(path string) (uint64, error) {
				return 1000 * 4096, nil
			},
			readJWTToken: func(afero.Fs, string, func(string) (bool, error)) (string, error) {
				return "valid-token", nil
			},
			getFileSizeInBytes: func(string, string) (int64, error) {
				return 1000 * 6130, nil
			},
			expectedResult: false,
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &Downloader{
				getFreeDiskSpaceInBytes: tt.getFreeDiskSpaceInBytes,
				getFileSizeInBytesFunc:  tt.getFileSizeInBytes,
				readJWTTokenFunc:        tt.readJWTToken,
				writeUpdateStatus:       tt.writeUpdateStatus,
				writeGranularLog:        tt.writeGranularLog,

				request: &pb.UpdateSystemSoftwareRequest{Url: "http://example.com"},
			}
			result, err := downloader.isDiskSpaceAvailable()
			assert.Equal(t, tt.expectedResult, result)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// roundTripperFunc is a helper type to mock http.RoundTripper
type roundTripperFunc func(req *http.Request) *http.Response

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
