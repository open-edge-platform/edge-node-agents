// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// PasswordGrantParams holds parameters for OAuth2 password grant flow.
type PasswordGrantParams struct {
	KeycloakURL string
	TokenPath   string // e.g., "/realms/master/protocol/openid-connect/token"
	Username    string
	Password    string
	ClientID    string
	Scope       string
	CACertPath  string
}

// ClientCredentialsParams holds parameters for OAuth2 client_credentials flow.
type ClientCredentialsParams struct {
	KeycloakURL  string
	TokenPath    string // e.g., "/realms/master/protocol/openid-connect/token"
	ClientID     string
	ClientSecret string
	CACertPath   string
}

// FetchPasswordGrantToken fetches a Keycloak access token using password grant flow.
// This is used in interactive mode where the user provides credentials via TTY.
func FetchPasswordGrantToken(ctx context.Context, params PasswordGrantParams) (string, error) {
	// Prepare the request data
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", params.ClientID)
	data.Set("username", params.Username)
	data.Set("password", params.Password)
	if params.Scope != "" {
		data.Set("scope", params.Scope)
	}

	reqBody := bytes.NewBufferString(data.Encode())

	// Construct the full URL
	tokenURL := "https://" + params.KeycloakURL + params.TokenPath

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create HTTP client with TLS
	client, err := NewHTTPClientWithTLS(params.CACertPath, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get access token, status: %s, body: %s", resp.Status, string(body))
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract the access token
	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("access token not found in response")
	}

	return token, nil
}

// FetchClientCredentialsToken fetches a Keycloak access token using client_credentials flow.
// This is used in non-interactive mode after the device has been onboarded and has client credentials.
func FetchClientCredentialsToken(ctx context.Context, params ClientCredentialsParams) (string, error) {
	// Prepare the request data
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", params.ClientID)
	data.Set("client_secret", params.ClientSecret)

	reqBody := bytes.NewBufferString(data.Encode())

	// Construct the full URL
	tokenURL := "https://" + params.KeycloakURL + params.TokenPath

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create HTTP client with TLS
	client, err := NewHTTPClientWithTLS(params.CACertPath, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get access token, status: %s, body: %s", resp.Status, string(body))
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract the access token
	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("access token not found in response")
	}

	return token, nil
}

// FetchReleaseToken fetches a release service token using an IDP access token.
// The release server URL is derived from the Keycloak URL by replacing "keycloak" with "release".
func FetchReleaseToken(ctx context.Context, releaseURL, accessToken, caCertPath string) (string, error) {
	// Ensure the access token is not empty
	if accessToken == "" {
		return "", fmt.Errorf("access token is required")
	}

	// Construct the HTTP request
	fullURL := "https://" + releaseURL
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add the authorization header with the bearer token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create HTTP client with TLS
	client, err := NewHTTPClientWithTLS(caCertPath, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is 200 OK
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get release token, status: %s, body: %s", resp.Status, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert the response body to a string (the token)
	token := string(body)

	// Validate the received token
	if token == "null" || token == "" {
		return "", fmt.Errorf("invalid token received")
	}

	return token, nil
}
