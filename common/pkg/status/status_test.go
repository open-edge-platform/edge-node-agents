// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
)

func TestInitClient(t *testing.T) {
	serverAddr := "unix:///run/node-agent/test.sock"
	client, err := InitClient(serverAddr)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, serverAddr, client.ServerAddr)
	require.NotNil(t, client.Conn)
	require.NotNil(t, client.Client)
}

type MockStatusServiceClient struct {
	mock.Mock
}

func (m *MockStatusServiceClient) ReportStatus(ctx context.Context, in *pb.ReportStatusRequest, _ ...grpc.CallOption) (*pb.ReportStatusResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.ReportStatusResponse), args.Error(1)
}

func (m *MockStatusServiceClient) GetStatusInterval(
	ctx context.Context, in *pb.GetStatusIntervalRequest, _ ...grpc.CallOption) (*pb.GetStatusIntervalResponse, error,
) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.GetStatusIntervalResponse), args.Error(1)
}

func TestSendStatusReady(t *testing.T) {
	serverAddr := "unix:///run/node-agent/test.sock"
	client, err := InitClient(serverAddr)
	require.NoError(t, err)
	require.NotNil(t, client)

	mockClient := new(MockStatusServiceClient)
	client.Client = mockClient

	agentName := "test-agent"
	status := pb.Status_STATUS_READY

	mockClient.On("ReportStatus", mock.Anything, &pb.ReportStatusRequest{
		AgentName: agentName,
		Status:    status,
	}).Return(&pb.ReportStatusResponse{}, nil)

	err = client.SendStatusReady(t.Context(), agentName)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestSendStatusNotReady(t *testing.T) {
	serverAddr := "unix:///run/node-agent/test.sock"
	client, err := InitClient(serverAddr)
	require.NoError(t, err)
	require.NotNil(t, client)

	mockClient := new(MockStatusServiceClient)
	client.Client = mockClient

	agentName := "test-agent"
	status := pb.Status_STATUS_NOT_READY

	mockClient.On("ReportStatus", mock.Anything, &pb.ReportStatusRequest{
		AgentName: agentName,
		Status:    status,
	}).Return(&pb.ReportStatusResponse{}, nil)

	err = client.SendStatusNotReady(t.Context(), agentName)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestGetStatusInterval(t *testing.T) {
	serverAddr := "unix:///run/node-agent/test.sock"
	client, err := InitClient(serverAddr)
	require.NoError(t, err)
	require.NotNil(t, client)

	mockClient := new(MockStatusServiceClient)
	client.Client = mockClient

	agentName := "test-agent"
	expectedInterval := time.Duration(30) * time.Second

	mockClient.On("GetStatusInterval", mock.Anything, &pb.GetStatusIntervalRequest{
		AgentName: agentName,
	}).Return(&pb.GetStatusIntervalResponse{IntervalSeconds: int32(30)}, nil)

	interval, err := client.GetStatusInterval(t.Context(), agentName)
	require.NoError(t, err)
	require.Equal(t, expectedInterval, interval)
	mockClient.AssertExpectations(t)
}

func TestGetStatusIntervalError(t *testing.T) {
	serverAddr := "unix:///run/node-agent/test.sock"
	client, err := InitClient(serverAddr)
	require.NoError(t, err)
	require.NotNil(t, client)

	mockClient := new(MockStatusServiceClient)
	client.Client = mockClient

	agentName := "test-agent"

	mockClient.On("GetStatusInterval", mock.Anything, &pb.GetStatusIntervalRequest{
		AgentName: agentName,
	}).Return(&pb.GetStatusIntervalResponse{}, errors.New("some error"))

	interval, err := client.GetStatusInterval(t.Context(), agentName)
	require.Error(t, err)
	require.Equal(t, time.Duration(0), interval)
	mockClient.AssertExpectations(t)
}

type MockServer struct {
	pb.UnimplementedStatusServiceServer
}

func (*MockServer) ReportStatus(context.Context, *pb.ReportStatusRequest) (*pb.ReportStatusResponse, error) {
	return &pb.ReportStatusResponse{}, nil
}

func startMockServer(t *testing.T, lis net.Listener) *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterStatusServiceServer(server, &MockServer{})
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("Error serving server: %v", err)
		}
	}()
	return server
}

func TestStatusServiceDisconnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lis, err := net.Listen("unix", "/tmp/test.sock")
	require.NoError(t, err)

	server := startMockServer(t, lis)
	// Wait for the server to start
	time.Sleep(2 * time.Second)

	sClient, err := InitClient("unix:///tmp/test.sock")
	require.NoError(t, err)

	err = sClient.SendStatusReady(t.Context(), "test-agent")
	require.NoError(t, err)

	server.Stop()
	lis.Close()
	time.Sleep(2 * time.Second) // Wait for the server to stop

	err = sClient.SendStatusReady(t.Context(), "test-agent")
	require.Error(t, err)

	// Restart the server
	lis, err = net.Listen("unix", "/tmp/test.sock") // Listener needs to be recreated
	require.NoError(t, err)
	defer lis.Close()
	server = startMockServer(t, lis)
	defer server.Stop()

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	err = sClient.SendStatusReady(t.Context(), "test-agent")
	require.NoError(t, err)
}
