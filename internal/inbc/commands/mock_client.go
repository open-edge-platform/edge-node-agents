/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"fmt"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockInbServiceClient is a mock implementation of the pb.InbServiceClient interface.
type MockInbServiceClient struct {
	mock.Mock
}

// AddApplicationSource is a mock implementation of the AddApplicationSource function.
func (m *MockInbServiceClient) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.UpdateResponse), args.Error(1)
}

// RemoveApplicationSource is a mock implementation of the RemoveApplicationSource function.
func (m *MockInbServiceClient) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.UpdateResponse), args.Error(1)
}

// UpdateOSSource is a mock implementation of the UpdateOSSource function.
func (m *MockInbServiceClient) UpdateOSSource(ctx context.Context, req *pb.UpdateOSSourceRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.UpdateResponse), args.Error(1)
}

// UpdateSystemSoftware is a mock implementation of the UpdateSystemSoftware function.
func (m *MockInbServiceClient) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.UpdateResponse), args.Error(1)
}

// LoadConfig is a mock implementation of the LoadConfig function.
func (m *MockInbServiceClient) LoadConfig(ctx context.Context, req *pb.LoadConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.ConfigResponse), args.Error(1)
}

// GetConfig is a mock implementation of the GetConfig function.
func (m *MockInbServiceClient) GetConfig(ctx context.Context, req *pb.GetConfigRequest, opts ...grpc.CallOption) (*pb.GetConfigResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.GetConfigResponse), args.Error(1)
}

// SetConfig is a mock implementation of the SetConfig function.
func (m *MockInbServiceClient) SetConfig(ctx context.Context, req *pb.SetConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.ConfigResponse), args.Error(1)
}

// AppendConfig is a mock implementation of the AppendConfig function.
func (m *MockInbServiceClient) AppendConfig(ctx context.Context, req *pb.AppendConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.ConfigResponse), args.Error(1)
}

// RemoveConfig is a mock implementation of the RemoveConfig function.
func (m *MockInbServiceClient) RemoveConfig(ctx context.Context, req *pb.RemoveConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.ConfigResponse), args.Error(1)
}

// UpdateFirmware is a mock implementation of the UpdateFirmware function.
func (m *MockInbServiceClient) UpdateFirmware(ctx context.Context, req *pb.UpdateFirmwareRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.UpdateResponse), args.Error(1)
}

// Query is a mock implementation of the Query function.
func (m *MockInbServiceClient) Query(ctx context.Context, req *pb.QueryRequest, opts ...grpc.CallOption) (*pb.QueryResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*pb.QueryResponse), args.Error(1)
}

// MockClientConn is a mock implementation of the grpc.ClientConnInterface interface.
type MockClientConn struct {
	grpc.ClientConnInterface
}

// Close is a mock implementation of the Close function.
func (m *MockClientConn) Close() error {
	return nil
}

// MockDialer is a mock implementation of the Dialer function
func MockDialer(_ context.Context, _ string, client *MockInbServiceClient, shouldError bool) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
	if shouldError {
		return nil, nil, fmt.Errorf("mock dialer error")
	}

	return client, &MockClientConn{}, nil
}
