// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

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
	reportAMTStatusError      error
}

type mockCommandExecutor struct {
	amtInfoOutput    []byte
	amtInfoError     error
	activationOutput []byte
	activationError  error
}

func (m *mockCommandExecutor) ExecuteAMTInfo() ([]byte, error) {
	return m.amtInfoOutput, m.amtInfoError
}

func (m *mockCommandExecutor) ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error) {
	return m.activationOutput, m.activationError
}

func (m *mockDeviceManagementServer) ReportAMTStatus(ctx context.Context, req *pb.AMTStatusRequest) (*pb.AMTStatusResponse, error) {
	log.Logger.Infof("Received ReportAMTStatus request: HostID=%s, Status=%v, Version=%s", req.HostId, req.Status, req.Version)
	if m.reportAMTStatusError != nil {
		return nil, m.reportAMTStatusError
	}
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
	assert.Error(t, err, "RetrieveActivationDetails should return error for deactivate operation")
	assert.Contains(t, err.Error(), "activation not requested", "Error should indicate activation was not requested")
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

	// Create mock executor with successful activation output and AMT info with connected RAS status.
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`msg="CIRA: Configured"`),
		activationError:  nil,
		amtInfoOutput:    []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: connected"),
		amtInfoError:     nil,
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
	assert.Equal(t, pb.ActivationStatus_ACTIVATED, capturedRequest.ActivationStatus,
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

	// Create mock executor with failed activation output (no CIRA: Configured) and disconnected RAS status.
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`msg="Activation failed"`),
		activationError:  nil,
		amtInfoOutput:    []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: not connected"),
		amtInfoError:     nil,
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

	// Verify that the activation result was reported with ACTIVATION_FAILED status since CIRA is not configured.
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, "host-id", capturedRequest.HostId, "Host ID should match")
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, capturedRequest.ActivationStatus,
		"Activation status should be ACTIVATION_FAILED when CIRA is not configured")
}

func TestRetrieveActivationDetails_Connecting_Timeout(t *testing.T) {
	var capturedRequests []*pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequests = append(capturedRequests, req) // Capture all requests
			log.Logger.Infof("Received ReportActivationResults request: %v", req)
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Create mock executor with connecting status that never changes to connected.
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`msg="CIRA: Configured"`),
		activationError:  nil,
		amtInfoOutput:    []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: connecting"),
		amtInfoError:     nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	// Use a longer timeout context for testing to avoid premature cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.RetrieveActivationDetails(ctx, "host-id", &config.Config{
		RPSAddress: "mock-service",
	})

	// The function should return an error due to the test context timeout, but that's expected
	// since we're testing the timeout scenario
	if err != nil {
		t.Logf("Expected error due to context timeout: %v", err)
	}

	// Verify that at least one ACTIVATING status was reported during monitoring
	assert.NotEmpty(t, capturedRequests, "At least one activation result should have been reported")

	// Check that we received ACTIVATING status reports
	hasActivatingStatus := false
	for _, req := range capturedRequests {
		if req.ActivationStatus == pb.ActivationStatus_ACTIVATING {
			hasActivatingStatus = true
			break
		}
	}
	assert.True(t, hasActivatingStatus, "Should have received at least one ACTIVATING status report")
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

func TestReportAMTStatus_ErrorMessageInOutput(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{
		reportAMTStatusError: fmt.Errorf("simulated gRPC error"),
	})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Simulate amtinfo output containing an error message
	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte(`ERRO[0000] AMT not found: MEI/driver is missing or the call to the HECI driver failed
ERRO[0000] Failed to execute due to access issues. Please ensure that Intel ME is present, the MEI driver is installed, and the runtime has administrator or root privileges.
ERRO[0000] Error 2: HECIDriverNotDetected
`),
		amtInfoError: fmt.Errorf("command failed"),
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should not return error for error message in output")
	assert.Equal(t, pb.AMTStatus_DISABLED, status, "AMT should be disabled if output contains error message")
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
