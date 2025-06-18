// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

// TestSendSuccess verifies that BackendSender.Send sends data successfully when all files and HTTP succeed.
func TestSendSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("https://localhost:12345"), 0640), "Should write endpoint file")
	require.NoError(t, os.WriteFile(tokenFile, []byte("user:pass"), 0640), "Should write token file")

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			user, pass, _ := req.BasicAuth()
			require.Equal(t, "user", user, "Username should match")
			require.Equal(t, "pass", pass, "Password should match")
			require.Equal(t, "reporting-v1", req.Header.Get("X-Scope-OrgID"), "OrgID header should match")
			require.Equal(t, "application/json", req.Header.Get("Content-Type"), "Content-Type header should match")
			require.Equal(t, "https://localhost:12345", req.URL.String(), "Endpoint should match")
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err, "Should read request body without error")
			require.Contains(t, string(body), `"streams"`, "Payload should contain streams")
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}
		}),
	}

	sender := newTestBackendSenderWithClient(t, endpointFile, tokenFile, client)

	data := &model.Root{
		Identity: model.Identity{GroupID: "gid"},
	}
	cfg := config.Config{
		Backend: config.BackendConfig{
			Backoff: config.BackendBackoffConfig{
				MaxTries: 3,
			},
		},
	}
	err := sender.Send(cfg, data)
	require.NoError(t, err, "Send should succeed when everything is correct")
}

// TestSendEndpointFileError checks that Send returns error if endpoint file is missing.
func TestSendEndpointFileError(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "not-exist-endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("user:pass"), 0640), "Should write token file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "failed to read endpoint file", "Should error if endpoint file is missing")
}

// TestSendEndpointInvalidURL checks that Send returns error if endpoint file is not a valid URL.
func TestSendEndpointInvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("not-a-url"), 0640), "Should write invalid endpoint file")
	require.NoError(t, os.WriteFile(tokenFile, []byte("user:pass"), 0640), "Should write token file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.Error(t, err, "Should error if endpoint file is not a valid URL")
	require.Contains(t, err.Error(), "invalid URI", "Should return parse error for invalid URL")
}

func TestSendEndpointInvalidScheme(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("http://example.com"), 0640), "Should write endpoint file with http scheme")
	require.NoError(t, os.WriteFile(tokenFile, []byte("user:pass"), 0640), "Should write token file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "invalid endpoint scheme", "Should error if endpoint scheme is not https")
}

func TestSendEndpointMissingHost(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("https:///"), 0640), "Should write endpoint file with missing host")
	require.NoError(t, os.WriteFile(tokenFile, []byte("user:pass"), 0640), "Should write token file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "invalid endpoint URL, missing host", "Should error if endpoint host is missing")
}

// TestSendTokenFileError checks that Send returns error if token file is missing.
func TestSendTokenFileError(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "not-exist-token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("https://localhost:12345"), 0640), "Should write endpoint file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "failed to read token file", "Should error if token file is missing")
}

// TestSendTokenInvalidFormat checks that Send returns error if token file is not username:password.
func TestSendTokenInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	tokenFile := filepath.Join(tmpDir, "token")
	require.NoError(t, os.WriteFile(endpointFile, []byte("https://localhost:12345"), 0640), "Should write endpoint file")
	require.NoError(t, os.WriteFile(tokenFile, []byte("notcolon"), 0640), "Should write invalid token file")

	sender := NewBackendSender(endpointFile, tokenFile)
	err := sender.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "invalid token format", "Should error if token file is not username:password")
}

// TestSendTokenTooLongUsernameOrPassword checks that Send returns error if username or password in token file is too long.
func TestSendTokenTooLongUsernameOrPassword(t *testing.T) {
	tmpDir := t.TempDir()
	endpointFile := filepath.Join(tmpDir, "endpoint")
	require.NoError(t, os.WriteFile(endpointFile, []byte("https://localhost:12345"), 0640), "Should write endpoint file")

	// Username too long
	tokenFile1 := filepath.Join(tmpDir, "token1")
	longUser := strings.Repeat("a", 257)
	require.NoError(t, os.WriteFile(tokenFile1, []byte(longUser+":pass"), 0640), "Should write token file")
	sender1 := NewBackendSender(endpointFile, tokenFile1)
	err := sender1.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "username too long", "Should error if username is too long")

	// Password too long
	tokenFile2 := filepath.Join(tmpDir, "token2")
	longPass := strings.Repeat("b", 257)
	require.NoError(t, os.WriteFile(tokenFile2, []byte("user:"+longPass), 0640), "Should write token file")
	sender2 := NewBackendSender(endpointFile, tokenFile2)
	err = sender2.Send(config.Config{}, &model.Root{})
	require.ErrorContains(t, err, "password too long", "Should error if password is too long")
}

// TestBuildPayloadSuccess checks that buildPayload returns valid JSON payload.
func TestBuildPayloadSuccess(t *testing.T) {
	data := &model.Root{
		Identity: model.Identity{GroupID: "gid"},
	}
	payload, err := buildPayload(data)
	require.NoError(t, err, "BuildPayload should not return error for valid data")
	require.Contains(t, string(payload), `"streams"`, "Payload should contain streams")
	require.Contains(t, string(payload), `\"gid\"`, "Payload should contain marshaled model.Root data")
}

// TestSendRequestSuccess checks that sendRequest returns nil on 2xx response.
func TestSendRequestSuccess(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
			}
		}),
	}
	sender := newTestBackendSenderWithClient(t, "endpoint", "token", client)
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 2,
		},
	}
	err := sender.sendRequest("http://localhost:12345", "user", "pass", []byte("{}"), backendCfg)
	require.NoError(t, err, "SendRequest should succeed on 2xx response")
}

// TestSendRequestCreateError checks that sendRequest returns error if request creation fails.
func TestSendRequestCreateError(t *testing.T) {
	sender := NewBackendSender("endpoint", "token")
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 2,
		},
	}
	err := sender.sendRequest(":", "user", "pass", []byte("{}"), backendCfg) // invalid URL
	require.ErrorContains(t, err, "failed to create backend request", "Should error if request creation fails")
}

// TestSendRequestHTTPError checks that sendRequest returns error if HTTP client fails.
func TestSendRequestHTTPError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFuncErr(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("http fail")
		}),
	}
	sender := newTestBackendSenderWithClient(t, "endpoint", "token", client)
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 2,
		},
	}
	err := sender.sendRequest("http://localhost:12345", "user", "pass", []byte("{}"), backendCfg)
	require.ErrorContains(t, err, "failed to send request to backend", "Should error if HTTP client fails")
}

// TestSendRequestNon2xxStatus checks that sendRequest returns error if backend returns non-2xx status.
func TestSendRequestNon2xxStatus(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Body:       io.NopCloser(bytes.NewBufferString("bad request")),
			}
		}),
	}
	sender := newTestBackendSenderWithClient(t, "endpoint", "token", client)
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 2,
		},
	}
	err := sender.sendRequest("http://localhost:12345", "user", "pass", []byte("{}"), backendCfg)
	require.ErrorContains(t, err, "non-2xx status returned", "Should error if backend returns non-2xx status")
}

// TestSendRequestBackoffFailure simulates repeated HTTP failures to test backoff.Retry not succeeding.
func TestSendRequestBackoffFailure(t *testing.T) {
	failCount := 0
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) *http.Response {
			failCount++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       io.NopCloser(bytes.NewBufferString("fail")),
			}
		}),
	}
	sender := newTestBackendSenderWithClient(t, "endpoint", "token", client)
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 3,
		},
	}
	err := sender.sendRequest("http://localhost:12345", "user", "pass", []byte("{}"), backendCfg)
	require.ErrorContains(t, err, "non-2xx status returned", "Should error after maxTries exceeded")
	require.GreaterOrEqual(t, failCount, 3, "Should attempt at least maxTries times")
}

// TestSendRequestBackoffEventuallySuccess simulates initial HTTP failures followed by a success to verify retry stops on 2xx.
func TestSendRequestBackoffEventuallySuccess(t *testing.T) {
	failCount := 0
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) *http.Response {
			failCount++
			if failCount < 3 {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewBufferString("fail")),
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
			}
		}),
	}
	sender := newTestBackendSenderWithClient(t, "endpoint", "token", client)
	backendCfg := config.BackendConfig{
		Backoff: config.BackendBackoffConfig{
			MaxTries: 5,
		},
	}
	err := sender.sendRequest("http://localhost:12345", "user", "pass", []byte("{}"), backendCfg)
	require.NoError(t, err, "SendRequest should succeed after retries when a 2xx is eventually returned")
	require.Equal(t, 3, failCount, "Should stop retrying after first 2xx response")
}

// FuzzReadAuthCredentials checks that readAuthCredentials never panics and always returns a valid error or username/password for random file contents.
func FuzzReadAuthCredentials(f *testing.F) {
	// Seed with valid and invalid credentials
	f.Add([]byte("user:pass"))
	f.Add([]byte("useronly"))
	f.Add([]byte(":passonly"))
	f.Add([]byte("user:pass:extra"))
	f.Add([]byte(""))
	f.Add([]byte("user:"))
	f.Add([]byte(":"))
	f.Add([]byte("user:pass\n"))
	f.Add([]byte("user:pass with spaces"))
	f.Add([]byte("user:pass:with:colons"))
	f.Add([]byte(strings.Repeat("a", 257) + ":pass")) // too long username
	f.Add([]byte("user:" + strings.Repeat("b", 257))) // too long password

	f.Fuzz(func(t *testing.T, creds []byte) {
		tmpDir := t.TempDir()
		tokenFile := filepath.Join(tmpDir, "token")
		require.NoError(t, os.WriteFile(tokenFile, creds, 0640), "Should write token file")

		sender := NewBackendSender("endpoint", tokenFile)
		username, password, err := sender.readAuthCredentials()

		// Should never panic, and if error is nil, username and password must be non-empty, <=256 chars, and separated by the first colon
		if err == nil {
			require.NotEmpty(t, username, "Username should not be empty if no error")
			require.NotEmpty(t, password, "Password should not be empty if no error")
			require.LessOrEqual(t, len(username), 256, "Username should not exceed 256 characters")
			require.LessOrEqual(t, len(password), 256, "Password should not exceed 256 characters")
			require.Contains(t, string(creds), ":", "Input must contain a colon if no error")
		}
	})
}

// --- Helpers ---

// newTestBackendSenderWithClient creates a new BackendSender with a custom http.Client (for testing) and a test audit logger.
func newTestBackendSenderWithClient(t *testing.T, endpointPath, tokenPath string, client *http.Client) *BackendSender {
	return &BackendSender{
		endpointPath: endpointPath,
		tokenPath:    tokenPath,
		httpClient:   client,
		auditLog:     zaptest.NewLogger(t).Sugar(),
	}
}

// roundTripFunc is a helper to mock http.RoundTripper with no error.
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// roundTripFuncErr is a helper to mock http.RoundTripper with error.
type roundTripFuncErr func(req *http.Request) (*http.Response, error)

func (f roundTripFuncErr) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
