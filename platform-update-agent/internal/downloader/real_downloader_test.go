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
	expectedCommand := []string{"inbc", "sota", "-m", "download-only", "-s", sha, "-u", url}
	assert.Equal(t, expectedCommand, calls[0].Args, "Command args mismatch.")
}

func TestNewDownloadExecutor_NoError(t *testing.T) {
	executor := NewDownloadExecutor(nil)
	assert.NotNil(t, executor, "NewDownloadExecutor(nil) should not return nil")
}
