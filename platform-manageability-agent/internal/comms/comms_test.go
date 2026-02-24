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
	amtInfoOutput      []byte
	amtInfoError       error
	activationOutput   []byte
	activationError    error
	deactivationOutput []byte
	deactivationError  error
	amtInfoFunc        func() ([]byte, error) // Custom function for dynamic behavior
}

func (m *mockCommandExecutor) ExecuteAMTInfo() ([]byte, error) {
	if m.amtInfoFunc != nil {
		return m.amtInfoFunc()
	}
	return m.amtInfoOutput, m.amtInfoError
}

func (m *mockCommandExecutor) ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error) {
	return m.activationOutput, m.activationError
}

func (m *mockCommandExecutor) ExecuteAMTDeactivate(rpsAddress, password string) ([]byte, error) {
	return m.deactivationOutput, m.deactivationError
}

func (m *mockDeviceManagementServer) ReportAMTStatus(ctx context.Context, req *pb.AMTStatusRequest) (*pb.AMTStatusResponse, error) {
	log.Logger.Infof("Received ReportAMTStatus request: HostID=%s, Status=%v, Feature=%s", req.HostId, req.Status, req.Feature)
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

	// Create mock executor with connecting status that triggers immediate deactivation.
	// This tests the new behavior where direct transition to "connecting" triggers deactivation.
	mockExecutor := &mockCommandExecutor{
		activationOutput:   []byte(`msg="CIRA: Configured"`),
		activationError:    nil,
		amtInfoOutput:      []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: connecting"),
		amtInfoError:       nil,
		deactivationOutput: []byte("Deactivated successfully"),
		deactivationError:  nil,
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

	// Should succeed with the new async deactivation logic
	assert.NoError(t, err, "RetrieveActivationDetails should succeed with new async deactivation logic")

	// Verify that at least one activation result was reported
	assert.NotEmpty(t, capturedRequests, "At least one activation result should have been reported")

	// With new logic, device going directly to "connecting" should trigger ACTIVATION_FAILED (for deactivation)
	hasActivationFailedStatus := false
	for _, req := range capturedRequests {
		if req.ActivationStatus == pb.ActivationStatus_ACTIVATION_FAILED {
			hasActivationFailedStatus = true
			break
		}
	}
	assert.True(t, hasActivationFailedStatus, "Should have received ACTIVATION_FAILED status due to immediate deactivation")
}

func TestReportAMTStatus_Success(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nFeatures: "),
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

func TestReportAMTStatus_AMT_Feature(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nFeatures: AMT Pro Corporate"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should succeed")
	assert.Equal(t, pb.AMTStatus_ENABLED, status, "AMT should be enabled for AMT Pro features")
}

func TestReportAMTStatus_AMT_Pro_Corporate_Exact(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nFeatures: AMT Pro Corporate"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should succeed")
	assert.Equal(t, pb.AMTStatus_ENABLED, status, "AMT should be enabled for exact 'AMT Pro Corporate' features")
}

func TestReportAMTStatus_ISM_Feature(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nFeatures: Intel Standard Manageability Corporate SKU"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should succeed")
	assert.Equal(t, pb.AMTStatus_ENABLED, status, "AMT should be enabled for ISM features")
}

func TestReportAMTStatus_Unknown_Feature(t *testing.T) {
	lis, server := runMockServer(&mockDeviceManagementServer{})
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("Version: 16.1.25.1424\nBuild Number: 3425\nFeatures: Some Unknown Feature"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor),
	)

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	status, err := client.ReportAMTStatus(context.Background(), "host-id")
	assert.NoError(t, err, "ReportAMTStatus should succeed")
	assert.Equal(t, pb.AMTStatus_ENABLED, status, "AMT should be enabled but with empty feature string for unknown features")
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

// TestRetrieveActivationDetails_InterruptedSystemCall tests the error handling during activation
func TestRetrieveActivationDetails_InterruptedSystemCall(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor with interrupted system call in activation output
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`time="2025-09-29T10:40:34Z" level=error msg="interrupted system call"`),
		activationError:  fmt.Errorf("activation failed with interrupted system call"),
		amtInfoOutput:    []byte("RAS Remote Status: not connected"),
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

	// Verify activation failed due to interrupted system call
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, capturedRequest.ActivationStatus,
		"Activation should fail when interrupted system call occurs")
}

// TestRetrieveActivationDetails_InterruptedSystemCallWithExitCode tests exit code 10 detection
func TestRetrieveActivationDetails_InterruptedSystemCallWithExitCode(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor with exit code 10 in activation output
	mockExecutor := &mockCommandExecutor{
		activationOutput: []byte(`Activation failed with exit code: 10`),
		activationError:  nil,
		amtInfoOutput:    []byte("RAS Remote Status: not connected"),
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

	// Verify activation failed due to exit code 10
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, capturedRequest.ActivationStatus,
		"Activation should fail when exit code 10 occurs")
}

// TestRetrieveActivationDetails_ConnectingStateTimeout tests the 3-minute timeout for connecting state
func TestRetrieveActivationDetails_ConnectingStateTimeout(t *testing.T) {
	var capturedRequests []*pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequests = append(capturedRequests, req)
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor that always returns connecting state
	mockExecutor := &mockCommandExecutor{
		amtInfoOutput:      []byte("RAS Remote Status: connecting"),
		amtInfoError:       nil,
		deactivationOutput: []byte("AMT deactivated successfully"),
		deactivationError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	// Set connecting state start time to 4 minutes ago to trigger timeout
	client.SetConnectingStateStartTime(time.Now().Add(-4 * time.Minute))

	err = client.RetrieveActivationDetails(context.Background(), "host-id", &config.Config{
		RPSAddress: "mock-service",
	})
	assert.NoError(t, err, "RetrieveActivationDetails should succeed")

	// Verify that timeout triggered deactivation and reported ACTIVATION_FAILED
	assert.NotEmpty(t, capturedRequests, "At least one activation result should have been reported")
	lastRequest := capturedRequests[len(capturedRequests)-1]
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, lastRequest.ActivationStatus,
		"Should report ACTIVATION_FAILED when connecting state times out")
}

// TestRetrieveActivationDetails_ConnectingStateWithinTimeout tests normal connecting state
func TestRetrieveActivationDetails_ConnectingStateWithinTimeout(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor that returns connecting state (immediate deactivation will be triggered)
	mockExecutor := &mockCommandExecutor{
		amtInfoOutput:      []byte("RAS Remote Status: connecting"),
		amtInfoError:       nil,
		deactivationOutput: []byte("Deactivated successfully"),
		deactivationError:  nil,
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

	// With new logic, direct transition to "connecting" triggers immediate deactivation (ACTIVATION_FAILED)
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, capturedRequest.ActivationStatus,
		"Should report ACTIVATION_FAILED when in connecting state triggers immediate deactivation")
}

// TestRetrieveActivationDetails_DeactivationFailure tests failed deactivation during timeout
func TestRetrieveActivationDetails_DeactivationFailure(t *testing.T) {
	var capturedRequest *pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequest = req
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor with failed deactivation
	mockExecutor := &mockCommandExecutor{
		amtInfoOutput:      []byte("RAS Remote Status: connecting"),
		amtInfoError:       nil,
		deactivationOutput: []byte("Deactivation failed"),
		deactivationError:  fmt.Errorf("deactivation command failed"),
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	// Set connecting state start time to 4 minutes ago to trigger timeout
	client.SetConnectingStateStartTime(time.Now().Add(-4 * time.Minute))

	err = client.RetrieveActivationDetails(context.Background(), "host-id", &config.Config{
		RPSAddress: "mock-service",
	})
	assert.NoError(t, err, "RetrieveActivationDetails should succeed")

	// Verify that failed deactivation still reports ACTIVATION_FAILED (timestamp reset for retry)
	assert.NotNil(t, capturedRequest, "Activation result should have been reported")
	assert.Equal(t, pb.ActivationStatus_ACTIVATION_FAILED, capturedRequest.ActivationStatus,
		"Should report ACTIVATION_FAILED when deactivation fails (retry in next cycle)")
}

// TestTriggerDeactivationAsync_AlreadyInProgress tests that concurrent deactivation attempts are blocked
func TestTriggerDeactivationAsync_AlreadyInProgress(t *testing.T) {
	mockExecutor := &mockCommandExecutor{
		amtInfoOutput: []byte("RAS Remote Status: connecting"),
		amtInfoError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig, WithMockExecutor(mockExecutor))

	// Manually set deactivation in progress
	client.SetDeactivationInProgress(true)

	// Trigger deactivation - should be blocked
	status := client.TriggerDeactivationAsync("host-id", "wss://mock-rps/activate", "test-password")
	assert.Equal(t, pb.ActivationStatus_ACTIVATING, status, "Should return ACTIVATING when deactivation already in progress")
}

// TestPerformDeactivationAsync_Success tests successful deactivation with polling
func TestPerformDeactivationAsync_Success(t *testing.T) {
	callCount := 0
	mockExecutor := &mockCommandExecutor{
		deactivationOutput: []byte("Deactivation successful"),
		deactivationError:  nil,
	}

	// Mock AMT info to return "connecting" first, then "not connected"
	mockExecutor.amtInfoFunc = func() ([]byte, error) {
		callCount++
		if callCount <= 2 {
			return []byte("RAS Remote Status: connecting"), nil
		}
		return []byte("RAS Remote Status: not connected"), nil
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig, WithMockExecutor(mockExecutor))

	// Set initial state
	client.SetPreviousState("connecting")
	client.SetDeactivationInProgress(false)

	// Perform deactivation
	done := make(chan bool, 1) // Buffered channel to prevent goroutine leak
	go func() {
		client.PerformDeactivationAsync("host-id", "wss://mock-rps/activate", "test-password")
		done <- true
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Success - deactivation completed
	case <-time.After(60 * time.Second):
		t.Fatal("Deactivation should have completed within 60 seconds")
	}

	// Verify final state
	assert.Equal(t, "not connected", client.GetPreviousState(), "Previous state should be updated to 'not connected'")
	assert.False(t, client.GetDeactivationInProgress(), "Deactivation in progress flag should be cleared")
	assert.True(t, callCount >= 3, "Should have polled multiple times before success")
}

// TestPerformDeactivationAsync_DeactivationCommandFails tests failed deactivation command
func TestPerformDeactivationAsync_DeactivationCommandFails(t *testing.T) {
	mockExecutor := &mockCommandExecutor{
		deactivationOutput: []byte("Command failed"),
		deactivationError:  fmt.Errorf("deactivation command failed"),
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig, WithMockExecutor(mockExecutor))

	// Set initial state
	client.SetPreviousState("connecting")
	client.SetDeactivationInProgress(false)

	// Perform async deactivation
	done := make(chan bool, 1) // Buffered channel to prevent goroutine leak
	go func() {
		client.PerformDeactivationAsync("host-id", "wss://mock-rps/activate", "test-password")
		done <- true
	}()

	// Wait for completion
	select {
	case <-done:
		// Expected - should exit quickly on command failure
	case <-time.After(2 * time.Second):
		t.Fatal("Deactivation should have failed quickly")
	}

	// Verify state unchanged (command failed, no polling)
	assert.Equal(t, "connecting", client.GetPreviousState(), "Previous state should remain unchanged on command failure")
	assert.False(t, client.GetDeactivationInProgress(), "Deactivation in progress flag should be cleared")
}

// TestPerformDeactivationAsync_AMTInfoFailures tests handling of AMT info command failures during polling
func TestPerformDeactivationAsync_AMTInfoFailures(t *testing.T) {
	callCount := 0
	mockExecutor := &mockCommandExecutor{
		deactivationOutput: []byte("Deactivation successful"),
		deactivationError:  nil,
	}

	// Mock AMT info to fail a few times, then succeed
	mockExecutor.amtInfoFunc = func() ([]byte, error) {
		callCount++
		if callCount <= 3 {
			return []byte(""), fmt.Errorf("AMT info failed")
		}
		return []byte("RAS Remote Status: not connected"), nil
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig, WithMockExecutor(mockExecutor))

	// Set initial state
	client.SetPreviousState("connecting")
	client.SetDeactivationInProgress(false)

	// Perform deactivation
	done := make(chan bool, 1) // Buffered channel to prevent goroutine leak
	go func() {
		client.PerformDeactivationAsync("host-id", "wss://mock-rps/activate", "test-password")
		done <- true
	}()

	// Wait for completion
	select {
	case <-done:
		// Success - should eventually succeed despite initial failures
	case <-time.After(60 * time.Second):
		t.Fatal("Deactivation should have completed within 60 seconds")
	}

	// Verify final state
	assert.Equal(t, "not connected", client.GetPreviousState(), "Previous state should be updated to 'not connected'")
	assert.False(t, client.GetDeactivationInProgress(), "Deactivation in progress flag should be cleared")
	assert.True(t, callCount >= 4, "Should have retried AMT info multiple times")
}

// TestRetrieveActivationDetails_UnableToAuthenticateWithAMT tests the "Unable to authenticate with AMT" scenario
func TestRetrieveActivationDetails_UnableToAuthenticateWithAMT(t *testing.T) {
	var capturedRequests []*pb.ActivationResultRequest

	mockServer := &mockDeviceManagementServer{
		operationType: pb.OperationType_ACTIVATE,
		onReportActivationResults: func(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
			capturedRequests = append(capturedRequests, req)
			return &pb.ActivationResultResponse{}, nil
		},
	}

	lis, server := runMockServer(mockServer)
	defer func() {
		server.GracefulStop()
		lis.Close()
	}()

	// Mock executor with "Unable to authenticate with AMT" error in activation output
	mockExecutor := &mockCommandExecutor{
		activationOutput:   []byte(`time="2025-11-05T10:30:15Z" level=error msg="Unable to authenticate with AMT"`),
		activationError:    fmt.Errorf("activation failed"),
		amtInfoOutput:      []byte("RAS Remote Status: not connected"),
		amtInfoError:       nil,
		deactivationOutput: []byte("Deactivation successful"),
		deactivationError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	assert.NoError(t, err, "Client should connect successfully")

	// Use a longer timeout context for testing to allow async deactivation to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.RetrieveActivationDetails(ctx, "host-id", &config.Config{
		RPSAddress: "mock-service",
	})

	// Should succeed despite the authentication failure (async deactivation triggered)
	assert.NoError(t, err, "RetrieveActivationDetails should succeed with async deactivation logic")

	// Verify that at least one activation result was reported
	assert.NotEmpty(t, capturedRequests, "At least one activation result should have been reported")

	// The "Unable to authenticate with AMT" error should trigger async deactivation
	// which returns ACTIVATION_FAILED status
	hasActivationFailedStatus := false
	for _, req := range capturedRequests {
		if req.ActivationStatus == pb.ActivationStatus_ACTIVATION_FAILED {
			hasActivationFailedStatus = true
			break
		}
	}
	assert.True(t, hasActivationFailedStatus, "Should have received ACTIVATION_FAILED status due to authentication failure triggering deactivation")

	// Give a brief moment for async deactivation to complete
	time.Sleep(100 * time.Millisecond)

	// Verify that deactivation is no longer in progress after completion
	assert.False(t, client.GetDeactivationInProgress(), "Deactivation should no longer be in progress after completion")
}
