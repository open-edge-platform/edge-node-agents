// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"device-discovery/internal/auth"
	"device-discovery/internal/config"
	"device-discovery/internal/logger"
)

// Credentials holds username and password from TTY input.
type Credentials struct {
	Username string
	Password string
}

// TTYAuthenticator handles TTY/PTY-based authentication for interactive mode.
type TTYAuthenticator struct {
	keycloakURL string
	caCertPath  string
	extraHosts  string
	ttyDevices  []string
	maxAttempts int
	logFile     string
}

// NewTTYAuthenticator creates a TTY authenticator with configuration loaded from validated-config.env.
func NewTTYAuthenticator(configPath string) (*TTYAuthenticator, error) {
	// Load configuration from file
	cfg := &config.Config{}
	if err := config.LoadFromFile(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	return &TTYAuthenticator{
		keycloakURL: cfg.KeycloakURL,
		caCertPath:  cfg.CaCertPath,
		extraHosts:  cfg.ExtraHosts,
		ttyDevices:  []string{"ttyS0", "ttyS1", "tty0"}, // Default devices from bash script
		maxAttempts: 3,
		logFile:     "/var/log/client-auth/client-auth.log",
	}, nil
}

// Authenticate performs the full authentication flow with retry logic.
// It attempts to read credentials from TTY devices, validate them with Keycloak,
// and fetch both IDP access token and release token.
func (t *TTYAuthenticator) Authenticate(ctx context.Context) error {
	// Ensure log directory exists
	if err := os.MkdirAll("/var/log/client-auth", 0755); err != nil {
		logger.Logger.Warnf("Failed to create log directory: %v", err)
	}

	// Update /etc/hosts with extra hosts if provided
	if t.extraHosts != "" {
		if err := config.UpdateHosts(t.extraHosts); err != nil {
			logger.Logger.Warnf("Failed to update /etc/hosts: %v", err)
		}
	}

	// Try up to maxAttempts times
	for attempt := 1; attempt <= t.maxAttempts; attempt++ {
		t.logToFile(fmt.Sprintf("Attempt %d to read username and password", attempt))

		creds, err := t.collectCredentials(ctx, attempt)
		if err != nil {
			t.logToFile(fmt.Sprintf("Failed to collect credentials: %v", err))
			continue
		}

		// Validate credentials by fetching tokens
		if err := t.validateAndFetchTokens(ctx, creds); err != nil {
			t.logToFile(fmt.Sprintf("Authentication failed: %v", err))
			t.showErrorToAllTTYs("Incorrect username and password provided.")
			if attempt < t.maxAttempts {
				time.Sleep(2 * time.Second) // Brief pause before retry
			}
			continue
		}

		// Success!
		t.logToFile("Authentication successful, IDP Access token saved")
		return nil
	}

	// All attempts failed
	t.showErrorToAllTTYs("Incorrect username and password provided.")
	return fmt.Errorf("authentication failed after %d attempts", t.maxAttempts)
}

// collectCredentials attempts to read credentials from multiple TTYs concurrently.
// Returns the first successfully collected credentials or an error if timeout/all fail.
func (t *TTYAuthenticator) collectCredentials(ctx context.Context, attemptNum int) (*Credentials, error) {
	// Create context with timeout (50 seconds to match bash script: 10 checks x 5 seconds)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	// Channels for coordination
	credsChan := make(chan *Credentials, 1)
	errChan := make(chan error, len(t.ttyDevices))

	// Launch a goroutine for each TTY device
	for _, device := range t.ttyDevices {
		go func(dev string) {
			creds, err := t.collectFromDevice(ctxWithTimeout, dev, attemptNum)
			if err != nil {
				errChan <- err
				return
			}
			// Try to send credentials (first one wins)
			select {
			case credsChan <- creds:
				// Successfully sent
			default:
				// Someone else already sent credentials
			}
		}(device)
	}

	// Wait for first success or timeout
	select {
	case creds := <-credsChan:
		cancel() // Cancel other goroutines
		return creds, nil
	case <-ctxWithTimeout.Done():
		return nil, fmt.Errorf("timeout waiting for user input")
	}
}

// collectFromDevice attempts to read credentials from a single TTY device.
func (t *TTYAuthenticator) collectFromDevice(ctx context.Context, devicePath string, attemptNum int) (*Credentials, error) {
	// Open the device
	reader, err := NewDeviceReader(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", devicePath, err)
	}
	defer reader.Close()

	// Display initial prompt
	if err := reader.Prompt("\nProvide Username and password for the IDP\n"); err != nil {
		return nil, err
	}

	// Read username
	username, err := reader.ReadUsername(ctx, "Username: ")
	if err != nil {
		return nil, fmt.Errorf("failed to read username: %w", err)
	}

	// Read password
	password, err := reader.ReadPassword(ctx, "Password: ")
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	// Display processing message
	if err := reader.Prompt("\nUsername, Password received: Processing\n"); err != nil {
		// Non-fatal, just log
		logger.Logger.Warnf("Failed to display processing message: %v", err)
	}

	// Sanitize inputs (matching bash script: tr -d " \n ;")
	username = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == ';' {
			return -1 // Remove character
		}
		return r
	}, username)

	password = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == ';' {
			return -1
		}
		return r
	}, password)

	// Validate minimum length (matching bash script)
	if len(username) < 3 || len(password) < 3 {
		return nil, fmt.Errorf("username or password too short")
	}

	t.logToFile(fmt.Sprintf("%s: Username and password received", devicePath))

	return &Credentials{
		Username: username,
		Password: password,
	}, nil
}

// validateAndFetchTokens validates credentials by fetching tokens from Keycloak and release server.
func (t *TTYAuthenticator) validateAndFetchTokens(ctx context.Context, creds *Credentials) error {
	// Check if CA certificate exists
	if t.caCertPath != "" {
		if _, err := os.Stat(t.caCertPath); err != nil {
			t.logToFile(fmt.Sprintf("IDP ca cert not found at the expected location: %s", t.caCertPath))
			return fmt.Errorf("CA certificate not found: %w", err)
		}
	}

	// Fetch IDP access token using password grant
	accessToken, err := auth.FetchPasswordGrantToken(ctx, auth.PasswordGrantParams{
		KeycloakURL: t.keycloakURL,
		TokenPath:   config.KeycloakTokenURL,
		Username:    creds.Username,
		Password:    creds.Password,
		ClientID:    "system-client",
		Scope:       "openid",
		CACertPath:  t.caCertPath,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch IDP access token: %w", err)
	}

	// Derive release server URL (replace "keycloak" with "release")
	releaseURL := strings.Replace(t.keycloakURL, "keycloak", "release", 1) + config.ReleaseTokenURL

	// Fetch release token
	releaseToken, err := auth.FetchReleaseToken(ctx, releaseURL, accessToken, t.caCertPath)
	if err != nil {
		return fmt.Errorf("failed to fetch release token: %w", err)
	}

	// Write tokens to /dev/shm
	if err := t.writeTokens(accessToken, releaseToken); err != nil {
		return fmt.Errorf("failed to write tokens: %w", err)
	}

	return nil
}

// writeTokens writes access and release tokens to /dev/shm.
func (t *TTYAuthenticator) writeTokens(accessToken, releaseToken string) error {
	// Write IDP access token
	if err := config.SaveToFile(config.AccessTokenFile, accessToken); err != nil {
		return fmt.Errorf("failed to save access token: %w", err)
	}

	// Write release token
	if err := config.SaveToFile(config.ReleaseTokenFile, releaseToken); err != nil {
		return fmt.Errorf("failed to save release token: %w", err)
	}

	return nil
}

// showErrorToAllTTYs displays an error message to all configured TTY devices.
func (t *TTYAuthenticator) showErrorToAllTTYs(message string) {
	for _, device := range t.ttyDevices {
		reader, err := NewDeviceReader(device)
		if err != nil {
			continue // Skip devices that can't be opened
		}
		reader.Prompt(fmt.Sprintf("\n%s\n", message))
		reader.Close()
	}
}

// logToFile writes a log message to the authentication log file.
func (t *TTYAuthenticator) logToFile(message string) {
	// Also log using the standard logger
	logger.Logger.Info(message)

	// Append to log file
	f, err := os.OpenFile(t.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "[%s] %s\n", timestamp, message)
}
