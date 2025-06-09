// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

// BackendSender holds configuration for backend communication.
type BackendSender struct {
	endpointPath string
	tokenPath    string
	httpClient   *http.Client
	auditLog     *zap.SugaredLogger
}

// NewBackendSender creates a new BackendSender with the given endpoint and token paths.
func NewBackendSender(endpointPath, tokenPath string) *BackendSender {
	return &BackendSender{
		endpointPath: endpointPath,
		tokenPath:    tokenPath,
		httpClient:   &http.Client{},
		auditLog:     createAuditLogger(),
	}
}

// Send sends the provided model.Root as a log entry to the backend using configured paths.
func (s *BackendSender) Send(data *model.Root) error {
	endpoint, err := s.readEndpointURL()
	if err != nil {
		return err
	}

	username, password, err := s.readAuthCredentials()
	if err != nil {
		return err
	}

	payload, err := buildPayload(data)
	if err != nil {
		return err
	}

	return s.sendRequest(endpoint, username, password, payload)
}

// readEndpointURL reads the endpoint URL from the configured file path and validates it.
func (s *BackendSender) readEndpointURL() (endpoint string, err error) {
	endpoint, err = utils.ReadFileTrimmed(s.endpointPath)
	if err != nil {
		return "", fmt.Errorf("failed to read endpoint file: %w", err)
	}

	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid endpoint URL: %s", endpoint)
	}

	return endpoint, nil
}

// readAuthCredentials reads the username and password from the configured file path.
// The file must contain "username:password".
func (s *BackendSender) readAuthCredentials() (username, password string, err error) {
	creds, err := utils.ReadFileTrimmed(s.tokenPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read token file: %w", err)
	}

	parts := strings.SplitN(creds, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid token format, expected username:password")
	}

	return parts[0], parts[1], nil
}

// buildPayload builds the backend payload for the log entry.
func buildPayload(data *model.Root) ([]byte, error) {
	logJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data for backend: %w", err)
	}

	nowNano := strconv.FormatInt(time.Now().UnixNano(), 10)
	payload := map[string]interface{}{
		"streams": []interface{}{
			map[string]interface{}{
				"stream": map[string]string{
					"Language": "Go",
					"source":   "Code",
				},
				"values": [][]string{
					{nowNano, string(logJSON)},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backend payload: %w", err)
	}

	return payloadBytes, nil
}

// sendRequest sends the HTTP request to the backend.
func (s *BackendSender) sendRequest(endpoint, username, password string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create backend request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Scope-OrgID", "staging-v3.1") // TODO: change this later
	req.SetBasicAuth(username, password)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx status returned: %s", resp.Status)
	}

	// Log the payload to audit log on success
	s.auditLog.Infow("Payload sent", "payload", string(payload))

	return nil
}

// createLogger initializes a new logger with a lumberjack writer for log rotation.
func createAuditLogger() *zap.SugaredLogger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.StacktraceKey = ""                      // disable stacktrace key
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // time in ISO8601 format (e.g. "2006-01-02T15:04:05.000Z0700")

	logWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/var/log/edge-node/reporting-audit.log",
		MaxAge:     90, // days
		MaxBackups: 5,
		Compress:   false,
	})

	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), logWriter, zapcore.InfoLevel)
	logger := zap.New(core)

	return logger.Sugar()
}
