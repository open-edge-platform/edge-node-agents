// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"fmt"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StatusClient struct {
	ServerAddr string
	Conn       *grpc.ClientConn
	Client     pb.StatusServiceClient
}

// InitClient initializes a new StatusClient with a new gRPC channel for serverAddr.
//
// Parameters:
//   - serverAddr: The address of the server to connect to.
//
// Returns:
//   - *StatusClient: A pointer to the initialized StatusClient.
//   - error: An error if the client creation fails, otherwise nil.
func InitClient(serverAddr string) (*StatusClient, error) {

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithNoProxy())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", serverAddr, err)
	}
	return &StatusClient{
		ServerAddr: serverAddr,
		Conn:       conn,
		Client:     pb.NewStatusServiceClient(conn),
	}, nil
}

func (cli *StatusClient) sendStatusRequest(ctx context.Context, agentName string, agentStatus pb.Status) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req := &pb.ReportStatusRequest{
		AgentName: agentName,
		Status:    agentStatus,
	}
	// Ignore response
	_, err := cli.Client.ReportStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send status: %w", err)
	}

	return nil
}

// SendStatusReady sends a status request indicating that the agent is ready.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and timeouts.
//   - agentName: The name of the agent whose status is being updated.
//
// Returns:
//   - error: An error if the status request fails, otherwise nil.
func (cli *StatusClient) SendStatusReady(ctx context.Context, agentName string) error {
	return cli.sendStatusRequest(ctx, agentName, pb.Status_STATUS_READY)
}

// SendStatusNotReady sends a status request indicating that the agent is not ready.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and timeouts.
//   - agentName: The name of the agent whose status is being updated.
//
// Returns:
//   - error: An error if the status request fails, otherwise nil.
func (cli *StatusClient) SendStatusNotReady(ctx context.Context, agentName string) error {
	return cli.sendStatusRequest(ctx, agentName, pb.Status_STATUS_NOT_READY)
}

// GetStatusInterval retrieves the status interval from status service.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and timeouts.
//   - agentName: The name of the agent for which the status interval is being requested.
//
// Returns:
//   - time.Duration: The interval in seconds for the status updates.
//   - error: An error if the request fails, otherwise nil.
func (cli *StatusClient) GetStatusInterval(ctx context.Context, agentName string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	intervalResponse, err := cli.Client.GetStatusInterval(ctx, &pb.GetStatusIntervalRequest{AgentName: agentName})
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve status interval: %w", err)
	}

	return time.Duration(intervalResponse.IntervalSeconds) * time.Second, nil
}
