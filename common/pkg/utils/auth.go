// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
)

// GetAuthConfig returns a TLS configuration for secure communication with the server.
func GetAuthConfig(_ context.Context, optionalCert *x509.Certificate) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get system CA certs: %w", err)
	}
	if optionalCert != nil {
		caCertPool.AddCert(optionalCert)
	}

	return &tls.Config{
		RootCAs:    caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
		},
	}, nil
}

// GetAuthContext creates a new context with an authorization header.
func GetAuthContext(ctx context.Context, tokenPath string) context.Context {
	tBytes, _ := ReadFileNoLinks(tokenPath) //nolint:errcheck // Ignoring error, if something goes wrong, bearer token will be empty
	tString := fmt.Sprintf("Bearer %s", tBytes)
	header := metadata.New(map[string]string{"authorization": strings.TrimSpace(tString)})

	return metadata.NewOutgoingContext(ctx, header)
}

// GetPerRPCCreds returns a PerRPCCredentials object that can be used for gRPC authentication.
func GetPerRPCCreds(tokenPath string) oauth.TokenSource {
	return oauth.TokenSource{TokenSource: getTokenSource(tokenPath)}
}

func getTokenSource(tokenPath string) oauth2.TokenSource {
	tSource := oauth2.StaticTokenSource(fetchToken(tokenPath))
	return oauth2.ReuseTokenSource(nil, tSource)
}

func fetchToken(tokenPath string) *oauth2.Token {
	tString, err := ReadFileNoLinks(tokenPath)
	if err != nil {
		fmt.Println("token file could not be read")
		return nil
	}
	return &oauth2.Token{
		AccessToken: string(tString),
		TokenType:   "Bearer",
	}
}
