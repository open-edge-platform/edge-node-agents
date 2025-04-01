// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package clientapi_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/clientapi"
	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// TelemetryManager is a placeholder for your actual implementation
type TelemetryManager struct {
	Client pb.TelemetryMgrClient
}

// Create a mock TelemetryMgrClient for testing
type mockTelemetryMgrClient struct {
	pb.TelemetryMgrClient
	response *pb.GetTelemetryConfigResponse
	err      error
}

func createTmpJWTFile(filename string) error {
	content := `testing-access-token-jwt`

	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return err
	}

	return nil
}

func (m *mockTelemetryMgrClient) GetTelemetryConfigByGUID(ctx context.Context, in *pb.GetTelemetryConfigByGuidRequest, opts ...grpc.CallOption) (*pb.GetTelemetryConfigResponse, error) {
	return m.response, m.err
}

func TestGetConfig(t *testing.T) {
	// Create a mock client with the expected response
	mockClient := &mockTelemetryMgrClient{
		response: &pb.GetTelemetryConfigResponse{
			HostGuid: "mock-host-guid",
			Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
				{
					Input:    "mock-input",
					Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
					Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
					Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
					Interval: 60,
				},
			},
		},
	}

	// Create an instance of the TelemetryManager
	telemetryMgr := &TelemetryManager{
		Client: mockClient,
	}

	err := createTmpJWTFile("access_token")
	require.Nil(t, err)

	mockTokenPath := "access_token"

	// Call the GetConfig method
	response, err := clientapi.GetConfig(context.Background(), telemetryMgr.Client, "mock-guid", mockTokenPath)

	// Add your assertions or checks here based on the expected response and error
	// For example:
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, "mock-host-guid", response.HostGuid)

	os.Remove(mockTokenPath)
}

func TestGetConfigError(t *testing.T) {
	// Create a mock client with the expected response
	mockClient := &mockTelemetryMgrClient{
		response: &pb.GetTelemetryConfigResponse{
			HostGuid: "mock-host-guid",
			Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
				{
					Input:    "mock-input",
					Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
					Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
					Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
					Interval: 60,
				},
			},
		},
	}

	// Create an instance of the TelemetryManager
	telemetryMgr := &TelemetryManager{
		Client: mockClient,
	}

	mockTokenPath := ""

	// Call the GetConfig method
	response, err := clientapi.GetConfig(context.Background(), telemetryMgr.Client, "mock-guid", mockTokenPath)

	// Add your assertions or checks here based on the expected response and error
	// For example:
	require.NotNil(t, err)
	require.Nil(t, response)
}

func TestCheckIfChanged(t *testing.T) {

	// Create a sample GetTelemetryConfigResponse for testing
	latestCfg := &pb.GetTelemetryConfigResponse{
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "input1",
				Interval: 10,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
			},
			// Add more TelemetryCfg as needed for your test cases
		},
	}

	// First call to CheckIfChanged, it should not indicate a change
	isDirtyMask, _ := clientapi.CheckIfChanged(latestCfg)
	if !isDirtyMask[0] && !isDirtyMask[1] && !isDirtyMask[2] && !isDirtyMask[3] {
		t.Error("Expected change on the first call, but got no change.")
	}

	// Second call with the same configuration, it should not indicate a change
	isDirtyMask, _ = clientapi.CheckIfChanged(latestCfg)
	if isDirtyMask[0] || isDirtyMask[1] || isDirtyMask[2] || isDirtyMask[3] {
		t.Error("Expected no change on the second call with the same configuration, but got a change.")
	}

	// Third call with the updated configuration, it should indicate a change
	// Create another sample GetTelemetryConfigResponse for testing
	latestCfg = &pb.GetTelemetryConfigResponse{
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "inputUpdate02",
				Interval: 10,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
			},
			// Add more TelemetryCfg as needed for your test cases
		},
	}
	isDirtyMask, _ = clientapi.CheckIfChanged(latestCfg)
	if !isDirtyMask[0] && !isDirtyMask[1] && !isDirtyMask[2] && !isDirtyMask[3] {
		t.Error("Expected a change on the third call with the updated configuration, but got no change.")
	}

	latestCfg = &pb.GetTelemetryConfigResponse{
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "inputUpdate03",
				Interval: 10,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
			},
			// Add more TelemetryCfg as needed for your test cases
		},
	}
	isDirtyMask, _ = clientapi.CheckIfChanged(latestCfg)
	if !isDirtyMask[0] && !isDirtyMask[1] && !isDirtyMask[2] && !isDirtyMask[3] {
		t.Error("Expected a change on the third call with the updated configuration, but got no change.")
	}

	latestCfg = &pb.GetTelemetryConfigResponse{
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "inputUpdate04",
				Interval: 10,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
			},
			// Add more TelemetryCfg as needed for your test cases
		},
	}
	isDirtyMask, _ = clientapi.CheckIfChanged(latestCfg)
	if !isDirtyMask[0] && !isDirtyMask[1] && !isDirtyMask[2] && !isDirtyMask[3] {
		t.Error("Expected a change on the third call with the updated configuration, but got no change.")
	}

	// Test the case where latestCfg is nil
	isDirtyMask, _ = clientapi.CheckIfChanged(nil)
	if isDirtyMask[0] || isDirtyMask[1] || isDirtyMask[2] || isDirtyMask[3] {
		t.Error("Expected no change on the second call with the same configuration, but got a change.")
	}

}

func TestWithNetworkDialer(t *testing.T) {
	serverAddr := "localhost:5566"
	client := clientapi.WithNetworkDialer(serverAddr)
	assert.NotNil(t, client, "Dialer should not be nil")
}

func TestNewClient(t *testing.T) {

	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	// Create a mock client with the expected response
	serverAddr := "localhost:5566"
	cli := clientapi.NewClient(serverAddr, tlsConfig, false)
	require.NotNil(t, cli)
	assert.Equal(t, serverAddr, cli.ServerAddr, tlsConfig)

	cli = clientapi.NewClient(serverAddr, tlsConfig, false)
	require.NotNil(t, cli)
	assert.Equal(t, serverAddr, cli.ServerAddr)

	cli = clientapi.NewClient(serverAddr, tlsConfig, true)
	if cli.GrpcConn != nil {
		t.Errorf("expected to be nil since tls not support in unit test mode")
	}

	option := func(cli *clientapi.Client) {
		cli.ServerAddr = "localhost:5566"
	}
	cli = clientapi.NewClient(serverAddr, tlsConfig, false, option)
	require.NotNil(t, cli)
	assert.Equal(t, serverAddr, cli.ServerAddr, tlsConfig)

}

// CustomBufDialer is used to create a connection to the in-memory gRPC server
func CustomBufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestConnect(t *testing.T) {
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}
	// Setup a mocked gRPC server
	server := grpc.NewServer()
	defer server.Stop()

	// Create a listener to be used for the in-memory connection
	listener := bufconn.Listen(1024 * 1024)
	go func() {
		// Your gRPC server logic here (if needed)
	}()

	// Use the mocked gRPC server in the client
	serverAddr := listener.Addr().String()
	cli := clientapi.NewClient(serverAddr, tlsConfig, false) // Assuming no TLS for simplicity

	// Dial the in-memory connection
	cli.Dialer = grpc.WithContextDialer(CustomBufDialer(listener))

	// Call the Connect method
	err := cli.Connect()

	// Assert that the connection was successful
	require.NoError(t, err)
	require.NotNil(t, cli.GrpcConn)
	assert.NotNil(t, cli.SouthboundClient)
}

func TestConnectFailed(t *testing.T) {
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}
	// Setup a mocked gRPC server
	server := grpc.NewServer()
	defer server.Stop()

	serverAddr := "invalid_address"
	cli := clientapi.NewClient(serverAddr, tlsConfig, false) // Assuming no TLS for simplicity

	// Call the Connect method
	err := cli.Connect()

	// Assert that the connection was successful
	require.Nil(t, err)
	require.NotNil(t, cli.GrpcConn)
	assert.NotNil(t, cli.SouthboundClient)

}

func TestConnectToServer(t *testing.T) {
	// Arrange
	mockServerAddr := "mockServerAddr"

	err := createTmpJWTFile("access_token")
	require.Nil(t, err)

	mockTokenPath := "access_token"

	// Act
	client, err := clientapi.ConnectToTelemetryManager(context.Background(), mockServerAddr, true)
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, client)

	os.Remove(mockTokenPath)
}

func TestConnectToServerFalsemTLS(t *testing.T) {
	// Arrange
	mockServerAddr := "mockServerAddr"

	// Act
	client, err := clientapi.ConnectToTelemetryManager(context.Background(), mockServerAddr, false)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately
	client, err = clientapi.ConnectToTelemetryManager(ctx, mockServerAddr, false)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

// Fuzz test
func FuzzGetConfigGuid(f *testing.F) {
	f.Add(uuid.New().String()) // Add seed input

	f.Fuzz(func(t *testing.T, fuzzData string) {
		// Sanitize fuzzData to avoid invalid characters in file operations
		sanitizedData := strings.Map(func(r rune) rune {
			if r == 0 || r == '\n' || r == '\r' {
				return -1 // Remove invalid characters
			}
			return r
		}, fuzzData)

		// Randomized content for the access token
		jwtContent := fmt.Sprintf("testing-access-token-jwt-%s", sanitizedData)

		// Create a unique temporary file
		tempFile, err := os.CreateTemp("", "access_token_*.tmp")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name()) // Ensure file cleanup after each run

		// Write content to the temporary file
		_, err = tempFile.Write([]byte(jwtContent))
		require.NoError(t, err)
		tempFile.Close() // Ensure file is fully written

		// Mock client with expected response
		mockClient := &mockTelemetryMgrClient{
			response: &pb.GetTelemetryConfigResponse{
				HostGuid: "mock-host-guid",
				Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
					{
						Input:    "mock-input",
						Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
						Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
						Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
						Interval: 60,
					},
				},
			},
		}

		// TelemetryManager instance
		telemetryMgr := &TelemetryManager{Client: mockClient}

		// Call GetConfig with fuzzed input
		response, err := clientapi.GetConfig(context.Background(), telemetryMgr.Client, sanitizedData, tempFile.Name())
		if err != nil {
			t.Logf("Error with fuzzData: %s, error: %v", sanitizedData, err)
		}

		// Assertions
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}
