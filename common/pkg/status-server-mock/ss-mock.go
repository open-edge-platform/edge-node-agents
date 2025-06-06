// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ssmock

import (
	"context"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
)

var version string
var commit string

type mockStatusServer struct {
	pb.UnimplementedStatusServiceServer
}

// ReportStatus implements the ReportStatus method of the StatusServiceServer interface.
func (*mockStatusServer) ReportStatus(_ context.Context, in *pb.ReportStatusRequest) (*pb.ReportStatusResponse, error) {
	fmt.Printf("Received status from agent %s: %s\n", in.AgentName, in.Status)
	return &pb.ReportStatusResponse{}, nil
}

// GetStatusInterval implements the GetStatusInterval method of the StatusServiceServer interface.
func (*mockStatusServer) GetStatusInterval(context.Context, *pb.GetStatusIntervalRequest) (*pb.GetStatusIntervalResponse, error) {
	return &pb.GetStatusIntervalResponse{IntervalSeconds: 10}, nil
}

// RunMockStatusServer starts a mock gRPC server that simulates the status service.
// Returns an error if the server fails to start or serve.
func RunMockStatusServer() error {
	fmt.Printf("Status Server Mock %s-%v\n", version, commit)
	if err := os.RemoveAll("/tmp/status-server.sock"); err != nil {
		return fmt.Errorf("failed to cleanup existing socket: %w", err)
	}

	lis, err := net.Listen("unix", "/tmp/status-server.sock")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer func() {
		lis.Close()
		os.RemoveAll("/tmp/status-server.sock")
	}()

	s := grpc.NewServer()
	pb.RegisterStatusServiceServer(s, &mockStatusServer{})

	fmt.Println("Status Server is running...")
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
