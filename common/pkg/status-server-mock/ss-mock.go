// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ssmock

import (
	"context"
	"fmt"
	"net"
	"os"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
	"google.golang.org/grpc"
)

var version string
var commit string

type mockStatusServer struct {
	pb.UnimplementedStatusServiceServer
}

func (s *mockStatusServer) ReportStatus(ctx context.Context, in *pb.ReportStatusRequest) (*pb.ReportStatusResponse, error) {
	fmt.Printf("Received status from agent %s: %s\n", in.AgentName, in.Status)
	return &pb.ReportStatusResponse{}, nil
}

func (s *mockStatusServer) GetStatusInterval(ctx context.Context, in *pb.GetStatusIntervalRequest) (*pb.GetStatusIntervalResponse, error) {
	return &pb.GetStatusIntervalResponse{IntervalSeconds: 10}, nil
}

func RunMockStatusServer() {
	fmt.Printf("Status Server Mock %s-%v\n", version, commit)
	err := os.RemoveAll("/tmp/status-server.sock")
	if err != nil {
		fmt.Printf("failed to cleanup existing socket: %v", err)
		os.Exit(1)
	}
	lis, err := net.Listen("unix", "/tmp/status-server.sock")
	defer os.RemoveAll("/tmp/status-server.sock")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		os.Exit(1)
	}
	defer lis.Close()

	s := grpc.NewServer()
	pb.RegisterStatusServiceServer(s, &mockStatusServer{})

	fmt.Println("Status Server is running...")
	if err := s.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
		os.Exit(1)
	}
}
