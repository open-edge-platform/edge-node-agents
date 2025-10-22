// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/test/bufconn"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

// mockDMManagerForErrorHandling represents a mock DM Manager server for error handling tests
type mockDMManagerForErrorHandling struct {
	pb.UnimplementedDeviceManagementServer
	activationResponse *pb.ActivationDetailsResponse
}

func (m *mockDMManagerForErrorHandling) RetrieveActivationDetails(ctx context.Context, req *pb.ActivationRequest) (*pb.ActivationDetailsResponse, error) {
	log.Logger.Infof("Mock: Received RetrieveActivationDetails request: %v", req)
	if m.activationResponse != nil {
		return m.activationResponse, nil
	}
	return &pb.ActivationDetailsResponse{
		HostId:         req.HostId,
		Operation:      pb.OperationType_ACTIVATE,
		ProfileName:    "test-profile",
		ActionPassword: "test-password",
	}, nil
}

func (m *mockDMManagerForErrorHandling) ReportActivationResults(ctx context.Context, req *pb.ActivationResultRequest) (*pb.ActivationResultResponse, error) {
	log.Logger.Infof("Mock: Received ReportActivationResults request: %v", req)
	return &pb.ActivationResultResponse{}, nil
}

// mockCommandExecutorForErrorHandling represents a mock command executor for error handling tests
type mockCommandExecutorForErrorHandling struct {
	amtInfoCallCount      int
	amtInfoFunc           func(callCount int) ([]byte, error)
	amtActivateOutput     []byte
	amtActivateError      error
	amtDeactivateOutput   []byte
	amtDeactivateError    error
	deactivationTriggered bool
	deactivateCallCount   int
}

func (m *mockCommandExecutorForErrorHandling) ExecuteAMTInfo() ([]byte, error) {
	m.amtInfoCallCount++
	if m.amtInfoFunc != nil {
		return m.amtInfoFunc(m.amtInfoCallCount - 1)
	}
	output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: not connected"
	return []byte(output), nil
}

func (m *mockCommandExecutorForErrorHandling) ExecuteAMTActivate(rpsAddress, profileName, password string) ([]byte, error) {
	return m.amtActivateOutput, m.amtActivateError
}

func (m *mockCommandExecutorForErrorHandling) ExecuteAMTDeactivate() ([]byte, error) {
	m.deactivationTriggered = true
	m.deactivateCallCount++
	return m.amtDeactivateOutput, m.amtDeactivateError
}

// setupMockDMManagerForErrorHandling sets up a mock DM Manager server for error handling tests
func setupMockDMManagerForErrorHandling(t *testing.T) (*bufconn.Listener, *mockDMManagerForErrorHandling) {
	mockServer := &mockDMManagerForErrorHandling{}
	lis, server := runMockServer(mockServer)
	t.Cleanup(func() {
		server.GracefulStop()
		lis.Close()
	})
	return lis, mockServer
}

// TestErrorHandlingScenarios validates deactivation
func TestErrorHandlingScenarios(t *testing.T) {
	tests := []struct {
		name                     string
		description              string
		initialPreviousState     string
		amtStatusSequence        []string
		expectedActivationStatus pb.ActivationStatus
		expectDeactivation       bool
		expectStateTransition    bool
		manualRecoveryAutomated  bool
	}{
		{
			name:                     "NotConnected_To_Connecting_3MinTimeout",
			description:              "RetrieveActivationDetails -> 'not connected' -> 'connecting' for more than 3 minutes -> deactivation -> activation will be triggered again by main timer",
			initialPreviousState:     "",
			amtStatusSequence:        []string{"not connected", "connecting"}, // First call returns "not connected", second call (after 3+ min) returns "connecting"
			expectedActivationStatus: pb.ActivationStatus_ACTIVATION_FAILED,   // deactivation triggered
			expectDeactivation:       true,
			expectStateTransition:    true,
			manualRecoveryAutomated:  true,
		},
		{
			name:                     "Immediate_Connecting_Direct_Deactivation",
			description:              "RetrieveActivationDetails -> immediate 'connecting' -> immediate deactivation will be triggered -> immediate deactivation -> activation will be triggered again by main timer",
			initialPreviousState:     "",                                    // Empty previous state simulates startup
			amtStatusSequence:        []string{"connecting"},                // Directly goes to connecting without "not connected"
			expectedActivationStatus: pb.ActivationStatus_ACTIVATION_FAILED, // Immediate deactivation triggered
			expectDeactivation:       true,
			expectStateTransition:    true,
			manualRecoveryAutomated:  true,
		},
		{
			name:                     "Success_Flow_NotConnected_To_Connected",
			description:              "Normal success flow: 'not connected' -> activation -> 'connected' should not be affected",
			initialPreviousState:     "",
			amtStatusSequence:        []string{"not connected", "connected"}, // Normal successful activation flow
			expectedActivationStatus: pb.ActivationStatus_ACTIVATED,          // Success case
			expectDeactivation:       false,
			expectStateTransition:    false,
			manualRecoveryAutomated:  false,
		},
		{
			name:                     "Success_Flow_Already_Connected",
			description:              "Already connected case should not be affected",
			initialPreviousState:     "connected",
			amtStatusSequence:        []string{"connected"},         // Already connected
			expectedActivationStatus: pb.ActivationStatus_ACTIVATED, // Success case
			expectDeactivation:       false,
			expectStateTransition:    false,
			manualRecoveryAutomated:  false,
		},
		{
			name:                     "Previous_Connected_To_Connecting",
			description:              "Device was previously connected but now in connecting - should trigger deactivation",
			initialPreviousState:     "connected",
			amtStatusSequence:        []string{"connecting"},                // Previously connected, now connecting
			expectedActivationStatus: pb.ActivationStatus_ACTIVATION_FAILED, // Async triggered
			expectDeactivation:       true,
			expectStateTransition:    true,
			manualRecoveryAutomated:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", tt.description)

			// Setup mock DM Manager
			lis, mockServer := setupMockDMManagerForErrorHandling(t)

			// Configure mock for activation request
			mockServer.activationResponse = &pb.ActivationDetailsResponse{
				HostId:         "test-host",
				Operation:      pb.OperationType_ACTIVATE,
				ProfileName:    "test-profile",
				ActionPassword: "test-password",
			}

			// Setup mock executor with controlled AMT info responses
			mockExecutor := &mockCommandExecutorForErrorHandling{
				amtInfoCallCount: 0,
				amtInfoFunc: func(callCount int) ([]byte, error) {
					// Return different responses based on call sequence
					if callCount < len(tt.amtStatusSequence) {
						status := tt.amtStatusSequence[callCount]
						output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: " + status
						return []byte(output), nil
					}
					// Default to "not connected" after deactivation for subsequent calls
					output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: not connected"
					return []byte(output), nil
				},
				amtActivateOutput:   []byte("Activated Successfully"),
				amtActivateError:    nil,
				amtDeactivateOutput: []byte("Deactivated Successfully"),
				amtDeactivateError:  nil,
			}

			// Create client with mock dependencies
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client := comms.NewClient("mock-service", tlsConfig,
				WithBufconnDialer(lis),
				WithMockExecutor(mockExecutor))

			// Set initial previous state
			client.SetPreviousState(tt.initialPreviousState)

			err := client.Connect(context.Background())
			require.NoError(t, err, "Client should connect successfully")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// For 3-minute timeout scenario, simulate time passing
			if tt.name == "NotConnected_To_Connecting_3MinTimeout" {
				// First call - should see "not connected"
				err = client.RetrieveActivationDetails(ctx, "test-host", &config.Config{
					RPSAddress: "mock-service",
				})

				// Should be activating at this point
				assert.NoError(t, err, "First activation call should succeed")
				assert.Equal(t, "not connected", client.GetPreviousState(), "Previous state should be 'not connected'")
				// Connecting state start time is only set when transition to "connecting", not during "not connected"
				assert.Nil(t, client.GetConnectingStateStartTime(), "Connecting state start time should be nil for 'not connected'")

				// Simulate being in connecting state for more than 3 minutes by setting the past time manually
				// This simulates the scenario where the device was in connecting state for > 3 minutes
				pastTime := time.Now().Add(-4 * time.Minute) // 4 minutes ago
				client.SetConnectingStateStartTime(pastTime)

				// Second call - should see "connecting" and trigger deactivation due to timeout
				_ = client.RetrieveActivationDetails(ctx, "test-host", &config.Config{
					RPSAddress: "mock-service",
				})
			} else {
				// For other scenarios, only single call
				_ = client.RetrieveActivationDetails(ctx, "test-host", &config.Config{
					RPSAddress: "mock-service",
				})
			}

			// Verify the response based on scenario
			if tt.expectDeactivation {
				// Should have triggered deactivation
				assert.True(t, mockExecutor.deactivationTriggered, "Deactivation should have been triggered for scenario: %s", tt.description)

				// Wait a moment for async deactivation to complete
				time.Sleep(100 * time.Millisecond)

				// Verify that deactivation was executed
				assert.True(t, mockExecutor.deactivateCallCount > 0, "Deactivate command should have been called for scenario: %s", tt.description)

				// Verify that after deactivation, state is reset properly for next activation cycle
				assert.False(t, client.GetDeactivationInProgress(), "Deactivation should not be in progress after completion")

			} else {
				// Should not have triggered deactivation for success flows
				assert.False(t, mockExecutor.deactivationTriggered, "Deactivation should not be triggered for success scenario: %s", tt.description)
			}

			// Verify state transitions
			if tt.expectStateTransition {
				assert.NotEmpty(t, client.GetPreviousState(), "Previous state should be updated")
			}

			// Verify that main timer will retry activation
			if tt.manualRecoveryAutomated {
				// After deactivation completes, the state should be ready for main timer retry
				assert.False(t, client.GetDeactivationInProgress(), "Should be ready for main timer to retry activation")
				t.Logf("Ready for main timer to trigger activation retry")
			}

			t.Logf("Scenario completed: %s", tt.description)
		})
	}
}

func TestAutomaticRecoveryE2E(t *testing.T) {

	// Setup mock DM Manager
	lis, mockServer := setupMockDMManagerForErrorHandling(t)

	mockServer.activationResponse = &pb.ActivationDetailsResponse{
		HostId:         "test-host",
		Operation:      pb.OperationType_ACTIVATE,
		ProfileName:    "test-profile",
		ActionPassword: "test-password",
	}

	// Setup mock executor that simulates the problematic "connecting" state
	mockExecutor := &mockCommandExecutorForErrorHandling{
		amtInfoCallCount: 0,
		amtInfoFunc: func(callCount int) ([]byte, error) {
			if callCount == 0 {
				// First call shows "connecting"
				output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: connecting"
				return []byte(output), nil
			}
			// After deactivation, should return "not connected"
			output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: not connected"
			return []byte(output), nil
		},
		amtDeactivateOutput: []byte("Deactivated Successfully"),
		amtDeactivateError:  nil,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := comms.NewClient("mock-service", tlsConfig,
		WithBufconnDialer(lis),
		WithMockExecutor(mockExecutor))

	err := client.Connect(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// device goes directly to "connecting"
	t.Log("Simulate problematic 'connecting' state")
	_ = client.RetrieveActivationDetails(ctx, "test-host", &config.Config{
		RPSAddress: "mock-service",
	})

	// Verify automatic deactivation was triggered
	t.Log("Verify automatic deactivation was triggered")
	assert.True(t, mockExecutor.deactivationTriggered, "Should automatically trigger deactivation")

	// Wait for deactivation to complete
	time.Sleep(200 * time.Millisecond)

	assert.True(t, mockExecutor.deactivateCallCount > 0, "Should execute deactivate command")
	t.Log("Automatic deactivation executed")

	assert.False(t, client.GetDeactivationInProgress(), "Deactivation should be complete")
	assert.Equal(t, "not connected", client.GetPreviousState(), "State should be reset to 'not connected'")

	// Verify main timer will retry activation
	t.Log("Verify main timer can retry activation")
	// Since deactivation is complete and state is "not connected", main timer will retry
	assert.False(t, client.GetDeactivationInProgress(), "Should be ready for main timer retry")
	t.Log("Ready for main timer to retry activation")
}

// TestSuccessFlowNotAffected verifies that normal success flow
func TestSuccessFlowNotAffected(t *testing.T) {
	successScenarios := []struct {
		name           string
		initialState   string
		amtStatus      string
		expectedStatus pb.ActivationStatus
	}{
		{
			name:           "Normal_NotConnected_To_Activation",
			initialState:   "",
			amtStatus:      "not connected",
			expectedStatus: pb.ActivationStatus_ACTIVATING,
		},
		{
			name:           "Already_Connected",
			initialState:   "connected",
			amtStatus:      "connected",
			expectedStatus: pb.ActivationStatus_ACTIVATED,
		},
		{
			name:           "Transition_NotConnected_To_Connected",
			initialState:   "not connected",
			amtStatus:      "connected",
			expectedStatus: pb.ActivationStatus_ACTIVATED,
		},
	}

	for _, scenario := range successScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			lis, mockServer := setupMockDMManagerForErrorHandling(t)

			mockServer.activationResponse = &pb.ActivationDetailsResponse{
				HostId:         "test-host",
				Operation:      pb.OperationType_ACTIVATE,
				ProfileName:    "test-profile",
				ActionPassword: "test-password",
			}

			mockExecutor := &mockCommandExecutorForErrorHandling{
				amtInfoFunc: func(callCount int) ([]byte, error) {
					output := "Version: 16.1.25.1424\nBuild Number: 3425\nRecovery Version: 16.1.25.1424\nRAS Remote Status: " + scenario.amtStatus
					return []byte(output), nil
				},
				amtActivateOutput: []byte("Activated Successfully"),
				amtActivateError:  nil,
			}

			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client := comms.NewClient("mock-service", tlsConfig,
				WithBufconnDialer(lis),
				WithMockExecutor(mockExecutor))

			client.SetPreviousState(scenario.initialState)

			err := client.Connect(context.Background())
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = client.RetrieveActivationDetails(ctx, "test-host", &config.Config{
				RPSAddress: "mock-service",
			})

			// Verify success flows
			assert.False(t, mockExecutor.deactivationTriggered, "Deactivation should not be triggered for success flow")
			assert.Equal(t, 0, mockExecutor.deactivateCallCount, "Deactivate should not be called for success flow")

			t.Logf("Success activation flow: %s", scenario.name)
		})
	}
}
