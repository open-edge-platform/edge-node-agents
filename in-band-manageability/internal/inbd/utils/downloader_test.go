/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestDownloadFile_Success(t *testing.T) {
	// Create a test server that returns a successful response
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is present
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testContent))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Ensure download directory exists
	err := fs.MkdirAll(IntelManageabilityCachePathPrefix, 0755)
	assert.NoError(t, err)

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err = DownloadFile(
		fs,
		server.URL+"/testfile.txt",
		IntelManageabilityCachePathPrefix,
		server.Client(),
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the result
	assert.NoError(t, err)

	// Check if file was created
	exists, err := afero.Exists(fs, IntelManageabilityCachePathPrefix+"/testfile.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check file content
	content, err := afero.ReadFile(fs, IntelManageabilityCachePathPrefix+"/testfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestDownloadFile_SuccessWithoutToken(t *testing.T) {
	// Create a test server that verifies no Authorization header
	testContent := "test file content without token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no Authorization header is present
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testContent))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a memory filesystem
	fs := afero.NewMemMapFs()

	// Ensure download directory exists
	err := fs.MkdirAll(IntelManageabilityCachePathPrefix, 0755)
	assert.NoError(t, err)

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "", nil // Return empty token
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err = DownloadFile(
		fs,
		server.URL+"/notoken.txt",
		IntelManageabilityCachePathPrefix,
		server.Client(),
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the result
	assert.NoError(t, err)

	// Check if file was created
	exists, err := afero.Exists(fs, IntelManageabilityCachePathPrefix+"/notoken.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check file content
	content, err := afero.ReadFile(fs, IntelManageabilityCachePathPrefix+"/notoken.txt")
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestDownloadFile_RequestCreationError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Mock request creator that returns an error
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("request creation failed")
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err := DownloadFile(
		fs,
		"http://example.com/file.txt",
		IntelManageabilityCachePathPrefix,
		&http.Client{},
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating request")
	assert.Contains(t, err.Error(), "request creation failed")
}

func TestDownloadFile_JWTTokenError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "", errors.New("token read failed")
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err := DownloadFile(
		fs,
		"http://example.com/file.txt",
		IntelManageabilityCachePathPrefix,
		&http.Client{},
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading JWT token")
	assert.Contains(t, err.Error(), "token read failed")
}

func TestDownloadFile_HTTPRequestError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Create a client that will fail
	client := &http.Client{
		Transport: &MockTransport{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		},
	}

	// Test the function
	err := DownloadFile(
		fs,
		"http://example.com/file.txt",
		IntelManageabilityCachePathPrefix,
		client,
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error performing request")
	assert.Contains(t, err.Error(), "network error")
}

func TestDownloadFile_HTTPStatusError(t *testing.T) {
	// Create a test server that returns an error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fs := afero.NewMemMapFs()

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err := DownloadFile(
		fs,
		server.URL+"/notfound.txt",
		IntelManageabilityCachePathPrefix,
		server.Client(),
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Status code: 404. Expected 200/Success.")
}

func TestDownloadFile_FileCreationError(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test content"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a read-only filesystem to simulate file creation error
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test the function
	err := DownloadFile(
		fs,
		server.URL+"/file.txt",
		IntelManageabilityCachePathPrefix,
		server.Client(),
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating file")
}

func TestDownloadFile_CopyError(t *testing.T) {
	// Create a test server with a large response that will cause copy to fail
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write some initial content
		_, err := w.Write([]byte("test"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a filesystem with limited space simulation
	fs := afero.NewMemMapFs()

	// Ensure download directory exists
	err := fs.MkdirAll(IntelManageabilityCachePathPrefix, 0755)
	assert.NoError(t, err)

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Create a mock client that returns a response with a body that will error during copy
	client := &http.Client{
		Transport: &MockTransport{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       &ErrorReader{},
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	// Test the function
	err = DownloadFile(
		fs,
		"http://example.com/file.txt",
		IntelManageabilityCachePathPrefix,
		client,
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error downloading file")
}

func TestDownloadFile_ComplexFilename(t *testing.T) {
	// Test with complex URL that has query parameters and fragments
	testContent := "content for complex filename"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testContent))
		assert.NoError(t, err)
	}))
	defer server.Close()

	fs := afero.NewMemMapFs()

	// Ensure download directory exists
	err := fs.MkdirAll(IntelManageabilityCachePathPrefix, 0755)
	assert.NoError(t, err)

	// Mock functions
	mockRequestCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, url, body)
	}

	mockReadJWTToken := func(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
		return "test-token", nil
	}

	mockIsTokenExpired := func(token string) (bool, error) {
		return false, nil
	}

	// Test with complex URL
	complexURL := server.URL + "/path/to/complex-file.tar.gz?version=1.0&arch=x86_64#fragment"

	err = DownloadFile(
		fs,
		complexURL,
		IntelManageabilityCachePathPrefix,
		server.Client(),
		mockRequestCreator,
		mockReadJWTToken,
		mockIsTokenExpired,
	)

	// Verify the result
	assert.NoError(t, err)

	// The filename should be the base name from the URL path (without query parameters or fragments)
	expectedFilename := "complex-file.tar.gz"
	exists, err := afero.Exists(fs, IntelManageabilityCachePathPrefix+"/"+expectedFilename)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check file content
	content, err := afero.ReadFile(fs, IntelManageabilityCachePathPrefix+"/"+expectedFilename)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// MockTransport is a helper for mocking HTTP transport
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// ErrorReader is a mock io.Reader that always returns an error
type ErrorReader struct{}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func (e *ErrorReader) Close() error {
	return nil
}
