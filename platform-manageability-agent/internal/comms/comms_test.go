// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

type mockDeviceManagementServer struct {
	pb.UnimplementedDeviceManagementServer
	operationType             pb.OperationType
	onReportActivationResults func(context.Context, *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error)
}

type mockCommandExecutor struct {
	amtInfoOutput    []byte
	amtInfoError     error
	activationOutput []byte
	activationError  error
}

func (m *mockCommandExecutor) ExecuteWithRetries(command string, args []string) ([]byte, error) {
	if command == "sudo" && len(args) >= 2 && args[0] == "./rpc" && args[1] == "amtinfo" {
		return m.amtInfoOutput, m.amtInfoError
	}
	return nil, fmt.Errorf("unexpected command: %s %v", command, args)
}

func (m *mockCommandExecutor) ExecuteCommand(name string, args ...string) ([]byte, error) {
	if name == "sudo" && len(args) >= 2 && args[0] == "rpc" && args[1] == "activate" {
		return m.activationOutput, m.activationError
	}
	return nil, fmt.Errorf("unexpected command: %s %v", name, args)
}

func (m *mockDeviceManagementServer) ReportAMTStatus(ctx context.Context, req *pb.AMTStatusRequest) (*pb.AMTStatusResponse, error) {
	log.Logger.Infof("Received ReportAMTStatus request: HostID=%s, Status=%v, Version=%s", req.HostId, req.Status, req.Version)
	return &pb.AMTStatusResponse{}, nil
}

func (m *mockDeviceManagementServer) RetrieveActivationDetails(ctx context.Context, req *pb.ActivationRequest) (*pb.ActivationDetailsResponse, error) {
	log.Logger.Infof("Received RetrieveActivationDetails request: %v", req)
	return &pb.ActivationDetailsResponse{
		HostId:      req.HostId,
		Operation:   m.operationType,
		ProfileName: "mock-profile",
	}, nil
}

func (m *mockDeviceManagementServer) ReportActivationResults(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
	if m.onReportActivationResults != nil {
		return m.onReportActivationResults(ctx, req)
	}
	log.Logger.Infof("Received ReportActivationResults request: %v", req)
	return &pb.ActivationResultResponse{}, nil
}

func runMockServer(server pb.DeviceManagementServer) (*bufconn.Listener, *grpc.Server) {
	lis := bufconn.Listen(1024 * 1024)
	creds, err := credentials.NewServerTLSFromFile("../../test/_dummy-cert.pem", "../../test/_dummy-key.pem")
	if err != nil {
		log.Logger.Fatalf("Failed to load TLS credentials: %v", err)
	}
	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterDeviceManagementServer(s, server)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Logger.Infof("Mock server stopped: %v", err)
		}
	}()
	return lis, s
}

func WithBufconnDialer(lis *bufconn.Listener) func(*comms.Client) {
	return func(cli *comms.Client) {
		cli.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		})
	}
}

func WithMockExecutor(executor utils.CommandExecutor) func(*comms.Client) {
	return func(c *comms.Client) {
		c.Executor = executor
	}
}

func TestRetrieveActivationDetails_DeactivateOperation(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{
		operationType: pb.OperationType_DEACTIVATE,
	})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig, WithBufconnDialer(lis))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	err = client.RetrieveActivationDetails(context.Background(), "host-id", &config.Config{
		RPSAddress: "mock-service",
	})
	assert.NoError(t, err, "RetrieveActivationDetails for deactivate should not process")
}

func TestRetrieveActivationDetails_Success(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req // Capture the request to verify later.
			log.Logger.Infof("Received ReportActivationResults request: %v", req)
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Create mock executor with successful activation output.
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`msg="CIRA: Configured"`),
		activationError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	err = client.RetrieveActivationDetails(context.Background(), "host-id", &config.Config{
		RPSAddress: "mock-service",
	})
	assert.NoError(t, err, "RetrieveActivationDetails should succeed")

	// Verify that the activation result was reported with PROVISIONED status.
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, "host-id", capturedRequest.HostId, "Host ID should match")
	assert.Equal(t, pb.ActivationStatus_PROVISIONED, capturedRequest.ActivationStatus,
		"Activation status should be PROVISIONED when CIRA is configured")
}

func TestRetrieveActivationDetails_Failed(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req // Capture the request to verify later.
			log.Logger.Infof("Received ReportActivationResults request: %v", req)
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Create mock executor with failed activation output (no CIRA: Configured).
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`msg="Activation failed"`),
		activationError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	err = client.RetrieveActivationDetails(context.Background(), "host-id", &config.Config{
		RPSAddress: "mock-service",
	})
	assert.NoError(t, err, "RetrieveActivationDetails should succeed")

	// Verify that the activation result was reported with FAILED status.
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, "host-id", capturedRequest.HostId, "Host ID should match")
	assert.Equal(t, pb.ActivationStatus_FAILED, capturedRequest.ActivationStatus,
		"Activation status should be FAILED when CIRA is not configured")
}

func TestReportAMTStatus_Success(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	_, err = client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should succeed")
}

func TestReportAMTStatus_CommandFailure(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: nil,
		amtInfoError:  fmt.Errorf("command failed"),
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.Error(t, err, "ReportAMTStatus should fail when command fails")
	assert.Equal(t, pb.AMTStatus_DISABLED, status, "AMT should be disabled on failure")
}
