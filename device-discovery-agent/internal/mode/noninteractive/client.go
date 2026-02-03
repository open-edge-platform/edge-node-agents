// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package noninteractive

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"device-discovery/internal/config"
)

// StreamResult holds the result of a streaming onboarding attempt.
type StreamResult struct {
	ClientID       string
	ClientSecret   string
	ProjectID      string
	ShouldFallback bool
	Error          error
}

// Client handles non-interactive (streaming) device onboarding.
type Client struct {
	address    string
	port       int
	mac        string
	uuid       string
	serial     string
	ipAddress  string
	caCertPath string
}

// NewClient creates a new non-interactive mode client.
func NewClient(address string, port int, mac, uuid, serial, ipAddress, caCertPath string) *Client {
	return &Client{
		address:    address,
		port:       port,
		mac:        mac,
		uuid:       uuid,
		serial:     serial,
		ipAddress:  ipAddress,
		caCertPath: caCertPath,
	}
}

// createSecureConnection creates a secure gRPC connection with TLS.
func createSecureConnection(target string, caCertPath string) (*grpc.ClientConn, error) {
	// Load the CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	// Create a certificate pool from the CA certificate
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to cert pool")
	}

	// Create the credentials using the certificate pool
	creds := credentials.NewClientTLSFromCert(certPool, "")

	// Create the gRPC connection with TLS credentials
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Onboard performs non-interactive device onboarding using streaming gRPC.
// Returns StreamResult containing credentials or error with fallback flag.
func (c *Client) Onboard(ctx context.Context) StreamResult {
	result := StreamResult{ShouldFallback: false}
	target := fmt.Sprintf("%s:%d", c.address, c.port)
	conn, err := createSecureConnection(target, c.caCertPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to connect: %v", err)
		return result
	}
	defer conn.Close()

	cli := pb.NewNonInteractiveOnboardingServiceClient(conn)

	// Establish a stream with the server
	stream, err := cli.OnboardNodeStream(ctx)
	if err != nil {
		result.Error = fmt.Errorf("could not create stream: %v", err)
		return result
	}
	defer stream.CloseSend()

	// Send a request over the stream
	request := &pb.OnboardNodeStreamRequest{
		MacId:     c.mac,
		Uuid:      c.uuid,
		Serialnum: c.serial,
		HostIp:    c.ipAddress,
	}

	// Receiving response from server with exponential backoff
	backoff := 2 * time.Second
	maxBackoff := 32 * time.Second

	for {
		if err := stream.Send(request); err != nil {
			result.Error = fmt.Errorf("could not send data to server: %v", err)
			return result
		}

		// Ensure stream is not nil
		if stream == nil {
			result.Error = fmt.Errorf("stream is nil")
			return result
		}

		// Receive response from the server
		resp, err := stream.Recv()
		if err == io.EOF {
			result.Error = fmt.Errorf("stream closed by server")
			return result
		}
		if err != nil {
			result.Error = fmt.Errorf("error receiving response from server: %v", err)
			return result
		}

		// Ensure the response is not nil
		if resp == nil {
			result.Error = fmt.Errorf("received nil response from server")
			return result
		}

		// Handle different node states
		if resp.Status.Code == int32(codes.OK) {
			switch resp.NodeState {
			case pb.OnboardNodeStreamResponse_NODE_STATE_REGISTERED:
				fmt.Println("Edge node registered. Waiting for the edge node to become ready for onboarding...")

				// Sleep for a randomized backoff duration
				time.Sleep(backoff + time.Duration(rand.Intn(1000))*time.Millisecond)

				// Double the backoff time, but cap it at maxBackoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = 2 * time.Second
				}

			case pb.OnboardNodeStreamResponse_NODE_STATE_ONBOARDED:
				clientID := resp.ClientId
				clientSecret := resp.ClientSecret
				projectID := resp.ProjectId

				// Ensure the Project ID is not empty
				if projectID == "" {
					result.Error = fmt.Errorf("received empty Project ID")
					return result
				}

				// Save the Project ID to a file
				if err := config.SaveToFile(config.ProjectIDPath, projectID); err != nil {
					result.Error = fmt.Errorf("failed to save Project ID to file: %v", err)
					return result
				}

				// Ensure both clientID and clientSecret are not empty
				if clientID == "" || clientSecret == "" {
					result.Error = fmt.Errorf("received empty clientID or clientSecret")
					return result
				}

				result.ClientID = clientID
				result.ClientSecret = clientSecret
				result.ProjectID = projectID
				return result

			case pb.OnboardNodeStreamResponse_NODE_STATE_UNSPECIFIED:
				result.Error = fmt.Errorf("edge node state is unspecified or unknown")
				return result

			default:
				result.Error = fmt.Errorf("unknown node state: %v", resp.NodeState)
				return result
			}
		} else if resp.Status.Code == int32(codes.NotFound) {
			// Device not found - trigger fallback to interactive mode
			result.ShouldFallback = true
			result.Error = fmt.Errorf("%s", resp.Status.Message)
			return result
		} else {
			result.Error = fmt.Errorf("%s", resp.Status.Message)
			return result
		}
	}
}
