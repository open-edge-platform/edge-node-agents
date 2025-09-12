/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"google.golang.org/grpc"
)

// ClientInterface is an interface for the client package
type ClientInterface interface {
	AddApplicationSource(context.Context, *pb.AddApplicationSourceRequest, ...grpc.CallOption) (*pb.UpdateResponse, error)
	RemoveApplicationSource(context.Context, pb.RemoveApplicationSourceRequest, ...grpc.CallOption) (*pb.UpdateResponse, error)
	UpdateOSSource(context.Context, *pb.UpdateOSSourceRequest, ...grpc.CallOption) (*pb.UpdateResponse, error)
	UpdateSystemSoftware(context.Context, *pb.UpdateSystemSoftwareRequest, ...grpc.CallOption) (*pb.UpdateResponse, error)
	UpdateFirmware(context.Context, *pb.UpdateFirmwareRequest, ...grpc.CallOption) (*pb.UpdateResponse, error)
	LoadConfig(context.Context, *pb.LoadConfigRequest, ...grpc.CallOption) (*pb.ConfigResponse, error)
	GetConfig(context.Context, *pb.GetConfigRequest, ...grpc.CallOption) (*pb.GetConfigResponse, error)
	SetConfig(context.Context, *pb.SetConfigRequest, ...grpc.CallOption) (*pb.ConfigResponse, error)
	AppendConfig(context.Context, *pb.AppendConfigRequest, ...grpc.CallOption) (*pb.ConfigResponse, error)
	RemoveConfig(context.Context, *pb.RemoveConfigRequest, ...grpc.CallOption) (*pb.ConfigResponse, error)
	Query(context.Context, *pb.QueryRequest, ...grpc.CallOption) (*pb.QueryResponse, error)
}

// RealClient wraps the client package calls.
type RealClient struct {
	client pb.InbServiceClient
}

// AddApplicationSource is a mock implementation of the AddApplicationSource function.
func (c *RealClient) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	return c.client.AddApplicationSource(ctx, req, opts...)
}

// RemoveApplicationSource is a mock implementation of the RemoveApplicationSource function.
func (c *RealClient) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest, opts_ ...grpc.CallOption) (*pb.UpdateResponse, error) {
	return c.client.RemoveApplicationSource(ctx, req, opts_...)
}

// UpdateOSSource is a mock implementation of the UpdateOSSource function.
func (c *RealClient) UpdateOSSource(ctx context.Context, req *pb.UpdateOSSourceRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	return c.client.UpdateOSSource(ctx, req, opts...)
}

// UpdateSystemSoftware is a mock implementation of the UpdateSystemSoftware function.
func (c *RealClient) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	return c.client.UpdateSystemSoftware(ctx, req, opts...)
}

// LoadConfig is a real implementation of the LoadConfig function.
func (c *RealClient) LoadConfig(ctx context.Context, req *pb.LoadConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	return c.client.LoadConfig(ctx, req, opts...)
}

// GetConfig is a real implementation of the GetConfig function.
func (c *RealClient) GetConfig(ctx context.Context, req *pb.GetConfigRequest, opts ...grpc.CallOption) (*pb.GetConfigResponse, error) {
	return c.client.GetConfig(ctx, req, opts...)
}

// SetConfig is a real implementation of the SetConfig function.
func (c *RealClient) SetConfig(ctx context.Context, req *pb.SetConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	return c.client.SetConfig(ctx, req, opts...)
}

// AppendConfig is a real implementation of the AppendConfig function.
func (c *RealClient) AppendConfig(ctx context.Context, req *pb.AppendConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	return c.client.AppendConfig(ctx, req, opts...)
}

// RemoveConfig is a real implementation of the RemoveConfig function.
func (c *RealClient) RemoveConfig(ctx context.Context, req *pb.RemoveConfigRequest, opts ...grpc.CallOption) (*pb.ConfigResponse, error) {
	return c.client.RemoveConfig(ctx, req, opts...)
}

// UpdateFirmware is a mock implementation of the UpdateFirmware function.
func (c *RealClient) UpdateFirmware(ctx context.Context, req *pb.UpdateFirmwareRequest, opts ...grpc.CallOption) (*pb.UpdateResponse, error) {
	return c.client.UpdateFirmware(ctx, req, opts...)
}

// Query is a real implementation of the Query function.
func (c *RealClient) Query(ctx context.Context, req *pb.QueryRequest, opts ...grpc.CallOption) (*pb.QueryResponse, error) {
	return c.client.Query(ctx, req, opts...)
}
