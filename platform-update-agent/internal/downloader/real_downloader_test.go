// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package downloader

import (
	"context"
	"sync"
	"testing"

	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type MockCommandRunner struct {
	mu       sync.Mutex
	calls    []CommandCall
	response []byte
	err      error
}

type CommandCall struct {
	Ctx  context.Context
	Name string
	Args []string
}

func (m *MockCommandRunner) RunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, CommandCall{Ctx: ctx, Name: name, Args: args})
	return m.response, m.err
}

func (m *MockCommandRunner) SetResponse(response []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = response
	m.err = err
}

func (m *MockCommandRunner) Calls() []CommandCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestRealDownloadExecutor_Download_Success(t *testing.T) {
	logger := logrus.New()
	logEntry := logrus.NewEntry(logger)
	mockRunner := &MockCommandRunner{}

	r := &RealDownloadExecutor{
		log:           logEntry,
		commandRunner: mockRunner,
	}

	url := "https://example.com/image"
	sha := "1234567890abcdef"

	source := &pb.OSProfileUpdateSource{
		OsImageUrl:  url,
		OsImageSha:  sha,
		ProfileName: "test-profile",
	}

	// Set success response for commands
	mockRunner.SetResponse([]byte("command success"), nil)

	err := r.Download(context.Background(), "", source)

	// Assert no error occurred
	assert.NoError(t, err, "Download() should not return an error")

	// Verify that the correct number of commands were called
	calls := mockRunner.Calls()
	assert.Len(t, calls, 1, "Expected 1 command to be run")

	// verify the command's arguments
	expectedCommand := []string{"inbc", "sota", "--mode", "download-only", "--signature", sha, "--uri", url}
	assert.Equal(t, expectedCommand, calls[0].Args, "Command args mismatch.")
}

func TestNewDownloadExecutor_NoError(t *testing.T) {
	executor := NewDownloadExecutor(nil)
	assert.NotNil(t, executor, "NewDownloadExecutor(nil) should not return nil")
}

func TestRealDownloadExecutor_Download_WithPrepend(t *testing.T) {
	logger := logrus.New()
	logEntry := logrus.NewEntry(logger)
	mockRunner := &MockCommandRunner{}

	r := &RealDownloadExecutor{
		log:           logEntry,
		commandRunner: mockRunner,
	}

	prependURL := "https://files-rs.example.com/"
	relativeURL := "repository/images/image.raw.gz"
	sha := "1234567890abcdef"

	source := &pb.OSProfileUpdateSource{
		OsImageUrl:  relativeURL,
		OsImageSha:  sha,
		ProfileName: "test-profile",
	}

	mockRunner.SetResponse([]byte("command success"), nil)

	err := r.Download(context.Background(), prependURL, source)

	assert.NoError(t, err)
	calls := mockRunner.Calls()
	assert.Len(t, calls, 1)

	expectedURL := prependURL + relativeURL
	expectedCommand := []string{"inbc", "sota", "--mode", "download-only", "--signature", sha, "--uri", expectedURL}
	assert.Equal(t, expectedCommand, calls[0].Args)
}

func TestRealDownloadExecutor_Download_WithCompleteURL(t *testing.T) {
	logger := logrus.New()
	logEntry := logrus.NewEntry(logger)
	mockRunner := &MockCommandRunner{}

	r := &RealDownloadExecutor{
		log:           logEntry,
		commandRunner: mockRunner,
	}

	prependURL := "https://files-rs.example.com/"
	completeURL := "https://artifactory.example.com/repository/images/image.raw.gz"
	sha := "1234567890abcdef"

	source := &pb.OSProfileUpdateSource{
		OsImageUrl:  completeURL,
		OsImageSha:  sha,
		ProfileName: "test-profile",
	}

	mockRunner.SetResponse([]byte("command success"), nil)

	err := r.Download(context.Background(), prependURL, source)

	assert.NoError(t, err)
	calls := mockRunner.Calls()
	assert.Len(t, calls, 1)

	// Should use complete URL without prepending
	expectedCommand := []string{"inbc", "sota", "--mode", "download-only", "--signature", sha, "--uri", completeURL}
	assert.Equal(t, expectedCommand, calls[0].Args)
}

func TestIsCompleteURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"https URL", "https://example.com/path", true},
		{"http URL", "http://example.com/path", true},
		{"relative path", "repository/images/file.gz", false},
		{"absolute path", "/var/cache/file.gz", false},
		{"empty string", "", false},
		{"ftp URL", "ftp://example.com/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCompleteURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
