// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

// Test constants to avoid security scanner alerts
const (
	testToken         = "test-token"
	testJWTToken      = "test-jwt-token"
	testValidToken    = "valid-token"
	testUsername      = "api"
	testWrongUsername = "wrong"
	testWrongToken    = "credentials"
	testBearerPrefix  = "Bearer "
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

func TestGetFileSizeInBytes_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns a successful response with Content-Length header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that a HEAD request is being made
		assert.Equal(t, "HEAD", r.Method)
		w.Header().Set("Content-Length", "12345")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testToken)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), size)
}

func TestGetFileSizeInBytes_WithoutToken(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that a HEAD request is being made
		assert.Equal(t, "HEAD", r.Method)
		// Verify no Authorization header is present
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Length", "54321")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(54321), size)
}

func TestGetFileSizeInBytes_WithToken(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that verifies the Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that a HEAD request is being made
		assert.Equal(t, "HEAD", r.Method)
		expectedAuth := testBearerPrefix + testToken
		assert.Equal(t, expectedAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Length", "98765")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testToken)
	assert.NoError(t, err)
	assert.Equal(t, int64(98765), size)
}

func TestGetFileSizeInBytes_InvalidURL(t *testing.T) {
	fs := afero.NewMemMapFs()
	size, err := GetFileSizeInBytes(fs, "://invalid-url", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating HEAD request")
	assert.Equal(t, int64(0), size)
}

func TestGetFileSizeInBytes_HTTPError(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns an error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HEAD request failed with status code: 404")
	assert.Equal(t, int64(0), size)
}

func TestGetFileSizeInBytes_MissingContentLength(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that uses chunked encoding (no Content-Length)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use Flusher to force chunked encoding which removes Content-Length
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		_, err := w.Write([]byte("test"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Content-Length header is missing")
	assert.Equal(t, int64(0), size)
}

func TestGetFileSizeInBytes_InvalidContentLength(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns invalid Content-Length
	// Note: Go's HTTP library will automatically reject invalid Content-Length headers
	// so this test verifies that we handle the resulting error correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "invalid-number")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, "")
	assert.Error(t, err)
	// The error will be from the HTTP request failing due to invalid Content-Length
	assert.Contains(t, err.Error(), "error performing HEAD request")
	assert.Equal(t, int64(0), size)
}

func TestGetFileSizeInBytes_FallbackToRange(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns 401 for HEAD but 206 for Range GET
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			// Verify that a HEAD request is being made first
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == "GET" && r.Header.Get("Range") == "bytes=0-0" {
			// Verify that a Range GET request is being made as fallback
			w.Header().Set("Content-Range", "bytes 0-0/54321")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			_, err := w.Write([]byte("x"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testToken)
	assert.NoError(t, err)
	assert.Equal(t, int64(54321), size)
}

func TestGetFileSizeInBytes_BothMethodsFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns 401 for both HEAD and Range GET
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Basic Auth HEAD request failed with status code: 401")
	assert.Equal(t, int64(0), size)
}

func TestGetFileSizeInBytes_BasicAuthFallback(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns 401 for Bearer token but accepts Basic Auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if strings.HasPrefix(authHeader, "Bearer ") {
			// Reject Bearer token like Artifactory does
			w.Header().Set("Www-Authenticate", "Basic realm=\"Artifactory Realm\"")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(authHeader, "Basic ") {
			// Accept Basic Auth
			if r.Method == "HEAD" {
				w.Header().Set("Content-Length", "98765")
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testJWTToken)
	assert.NoError(t, err)
	assert.Equal(t, int64(98765), size)
}

func TestGetFileSizeInBytes_AllMethodsFail(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create a test server that returns 401 for all authentication methods
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	size, err := GetFileSizeInBytes(fs, server.URL, testToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Basic Auth HEAD request failed with status code: 401")
	assert.Equal(t, int64(0), size)
}

func TestIsDiskSpaceAvailable_SufficientSpace(t *testing.T) {
	// Mock functions for successful scenario
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return testValidToken, nil
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 200000000, nil // 200MB available
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 50000000, nil // 50MB required (with buffer: 50MB * 1.2 + 100MB = 160MB total)
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.NoError(t, err)
	assert.True(t, available)
}

func TestIsDiskSpaceAvailable_InsufficientSpace(t *testing.T) {
	// Mock functions for insufficient space scenario
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return testValidToken, nil
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 50000000, nil // 50MB available
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 50000000, nil // 50MB required (with buffer: 50MB * 1.2 + 100MB = 160MB total)
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.NoError(t, err)
	assert.False(t, available)
}

func TestIsDiskSpaceAvailable_DiskSpaceError(t *testing.T) {
	// Mock functions with disk space error
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return testValidToken, nil
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 0, errors.New("disk space error")
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 50000000, nil // 50MB
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk space error")
	assert.False(t, available)
}

func TestIsDiskSpaceAvailable_TokenError(t *testing.T) {
	// Mock functions with JWT token error
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "", errors.New("token read error")
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 1000000, nil
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 500000, nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading JWT token")
	assert.False(t, available)
}

func TestIsDiskSpaceAvailable_FileSizeError(t *testing.T) {
	// Mock functions with file size error
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return testValidToken, nil
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 1000000, nil
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 0, errors.New("file size error")
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting file size")
	assert.False(t, available)
}

func TestIsDiskSpaceAvailable_ExactSpace(t *testing.T) {
	// Mock functions for exact space scenario (edge case)
	// Test case where available space exactly matches required space including buffer
	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return testValidToken, nil
	}

	mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
		return 160000000, nil // Exactly 160MB available
	}

	mockGetFileSize := func(url, token string) (int64, error) {
		return 50000000, nil // 50MB required (with buffer: 50MB * 1.2 + 100MB = 160MB total)
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	fs := afero.NewMemMapFs()

	available, err := IsDiskSpaceAvailable(
		"http://example.com/file",
		mockReadJWTToken,
		mockGetFreeDiskSpace,
		mockGetFileSize,
		mockIsTokenExpired,
		fs,
	)

	assert.NoError(t, err)
	assert.True(t, available) // Should be true because availableSpace >= requiredSpaceWithBuffer
}

func TestIsDiskSpaceAvailable_BufferLogic(t *testing.T) {
	// Test specifically for buffer calculation logic
	// For small files: minimum 100MB buffer should apply
	// For large files: 20% buffer should apply

	// Test case 1: Small file (10MB) - should use minimum 100MB buffer
	t.Run("small file uses minimum buffer", func(t *testing.T) {
		mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
			return testValidToken, nil
		}

		mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
			return 115000000, nil // 115MB available (just enough for 10MB + 100MB buffer + 5MB extra)
		}

		mockGetFileSize := func(url, token string) (int64, error) {
			return 10000000, nil // 10MB file (with buffer: 10MB * 1.2 = 12MB, but min buffer 100MB applies = 110MB total)
		}

		mockIsTokenExpired := func(token string) (bool, error) {
			return false, nil
		}

		fs := afero.NewMemMapFs()

		available, err := IsDiskSpaceAvailable(
			"http://example.com/file",
			mockReadJWTToken,
			mockGetFreeDiskSpace,
			mockGetFileSize,
			mockIsTokenExpired,
			fs,
		)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	// Test case 2: Large file (1GB) - should use 20% buffer
	t.Run("large file uses percentage buffer", func(t *testing.T) {
		mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
			return testValidToken, nil
		}

		mockGetFreeDiskSpace := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (uint64, error) {
			return 1300000000, nil // 1.3GB available (just enough for 1GB * 1.2 = 1.2GB total)
		}

		mockGetFileSize := func(url, token string) (int64, error) {
			return 1000000000, nil // 1GB file (with buffer: 1GB * 1.2 = 1.2GB total)
		}

		mockIsTokenExpired := func(token string) (bool, error) {
			return false, nil
		}

		fs := afero.NewMemMapFs()

		available, err := IsDiskSpaceAvailable(
			"http://example.com/file",
			mockReadJWTToken,
			mockGetFreeDiskSpace,
			mockGetFileSize,
			mockIsTokenExpired,
			fs,
		)

		assert.NoError(t, err)
		assert.True(t, available)
	})
}

// Test for tryBasicAuthWithCredentials function
func TestTryBasicAuthWithCredentials(t *testing.T) {
	t.Run("Success with correct credentials", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Create a test server that expects Basic Auth
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request has Basic Auth header
			user, pass, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Expect test credentials
			if user != testUsername || pass != testToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Return success with Content-Length
			w.Header().Set("Content-Length", "12345")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		size, err := tryBasicAuthWithCredentials(fs, server.URL, testUsername, testToken)
		assert.NoError(t, err)
		assert.Equal(t, int64(12345), size)
	})

	t.Run("Failure with wrong credentials", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Create a test server that rejects Basic Auth
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		size, err := tryBasicAuthWithCredentials(fs, server.URL, testWrongUsername, testWrongToken)
		assert.Error(t, err)
		assert.Equal(t, int64(0), size)
		assert.Contains(t, err.Error(), "failed with status code: 401")
	})

	t.Run("Missing Content-Length header", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Create a test server that returns 200 but no Content-Length
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		size, err := tryBasicAuthWithCredentials(fs, server.URL, testUsername, testToken)
		assert.Error(t, err)
		assert.Equal(t, int64(0), size)
		assert.Contains(t, err.Error(), "Content-Length header is missing")
	})
}
