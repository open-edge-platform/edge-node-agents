/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"path/filepath"

	utils "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Dialer is a type for dialing a gRPC client
type Dialer func(ctx context.Context, addr string) (pb.InbServiceClient, *grpc.ClientConn, error)

// Dial returns a new gRPC client with mTLS
func Dial(ctx context.Context, addr string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
	// Get TLS secret directory path from configuration
	tlsSecretDir := utils.GetTLSDirSecret()

	// Load client cert and key from secret directory
	certPath := filepath.Join(tlsSecretDir, "inbc.crt")
	keyPath := filepath.Join(tlsSecretDir, "inbc.key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	// Load CA cert from secret directory
	caCertPath := filepath.Join(tlsSecretDir, "ca.crt")
	caCert, err := utils.ReadFile(afero.NewOsFs(), caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, nil, fmt.Errorf("failed to append CA certificate")
	}
	creds := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, // TLS 1.2
			// TLS 1.3 cipher suites are not configurable in Go, but TLS_AES_256_GCM_SHA384 is always enabled for TLS 1.3
		},
	})

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		// cut off the unix:// part
		addr = addr[7:]
		return net.Dial("unix", addr)
	}

	conn, err := grpc.NewClient("unix://"+addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}

	return pb.NewInbServiceClient(conn), conn, nil
}
