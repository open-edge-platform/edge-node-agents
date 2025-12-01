/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/afero"
)

// ReadJWTToken reads the JWT token that is used for accessing RS server.
func ReadJWTToken(fs afero.Fs, path string, isTokenExpiredFunc func(string) (bool, error)) (string, error) {
	token, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", err
	}

	if len(token) == 0 {
		// Allowed to return empty token if the file is empty.
		// This is useful for cases where the token is not required.
		return "", nil
	}

	// Check if token is "anonymous" - treat as empty token (no authentication)
	tokenStr := strings.TrimSpace(string(token))
	if strings.ToLower(tokenStr) == "anonymous" {
		log.Println("JWT token file contains 'anonymous'. Treating as no authentication.")
		return "", nil
	}

	expired, err := isTokenExpiredFunc(tokenStr)
	if err != nil {
		return "", fmt.Errorf("error checking token expiration: %w", err)
	}
	if expired {
		return "", fmt.Errorf("token is expired")
	}
	return tokenStr, nil
}

// IsTokenExpired checks if a JWT token is expired.
func IsTokenExpired(tokenString string) (bool, error) {
	// Parse the token without validating the signature
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return false, fmt.Errorf("error parsing token: %w", err)
	}

	// Extract the claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("error extracting claims from token")
	}

	// Check the "exp" claim
	exp, ok := claims["exp"].(float64) // "exp" is usually a float64
	if !ok {
		return false, fmt.Errorf("token does not have a valid 'exp' claim")
	}

	// Compare the expiration time with the current time
	expirationTime := time.Unix(int64(exp), 0)
	return time.Now().After(expirationTime), nil
}
