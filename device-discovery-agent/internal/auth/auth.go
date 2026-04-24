// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
)

// ClientAuth handles authentication and retrieves tokens using client credentials.
// This function is used in non-interactive mode after the device has been onboarded.
func ClientAuth(clientID string, clientSecret string, keycloakURL string, accessTokenURL string, releaseTokenURL string, caCertPath string) (idpAccessToken string, releaseToken string, err error) {
	ctx := context.Background()

	// Fetch JWT access token from Keycloak using client_credentials flow
	idpAccessToken, err = FetchClientCredentialsToken(ctx, ClientCredentialsParams{
		KeycloakURL:  keycloakURL,
		TokenPath:    accessTokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CACertPath:   caCertPath,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get JWT access token from Keycloak: %v", err)
	}

	return idpAccessToken, "anonymous", nil
}
