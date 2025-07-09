/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockDialerForQuery creates a mock dialer for testing query command
func mockDialerForQuery(client *MockInbServiceClient, err error) func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
	return func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		if err != nil {
			return nil, nil, err
		}
		return client, &MockClientConn{}, nil
	}
}

func TestQueryCmd(t *testing.T) {
	cmd := QueryCmd()

	// Test command properties
	assert.Equal(t, "query", cmd.Use)
	assert.Equal(t, "Query system information", cmd.Short)
	assert.Contains(t, cmd.Long, "Query system information including hardware")
	assert.Contains(t, cmd.Example, "inbc query")

	// Test flags
	socketFlag := cmd.Flag("socket")
	require.NotNil(t, socketFlag)
	assert.Equal(t, "/var/run/inbd.sock", socketFlag.DefValue)

	optionFlag := cmd.Flag("option")
	require.NotNil(t, optionFlag)
	assert.Equal(t, "o", optionFlag.Shorthand)
	assert.Equal(t, "all", optionFlag.DefValue) // Default should be "all"

	// Test that command works without option flag (should use default "all")
	cmd.SetArgs([]string{})
	// This should not error now since we have a default value
	// We'll test the actual execution in integration tests
}

func TestParseQueryOption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected pb.QueryOption
		wantErr  bool
	}{
		{
			name:     "hardware option",
			input:    "hw",
			expected: pb.QueryOption_QUERY_OPTION_HARDWARE,
			wantErr:  false,
		},
		{
			name:     "hardware option long",
			input:    "hardware",
			expected: pb.QueryOption_QUERY_OPTION_HARDWARE,
			wantErr:  false,
		},
		{
			name:     "firmware option",
			input:    "fw",
			expected: pb.QueryOption_QUERY_OPTION_FIRMWARE,
			wantErr:  false,
		},
		{
			name:     "firmware option long",
			input:    "firmware",
			expected: pb.QueryOption_QUERY_OPTION_FIRMWARE,
			wantErr:  false,
		},
		{
			name:     "os option",
			input:    "os",
			expected: pb.QueryOption_QUERY_OPTION_OS,
			wantErr:  false,
		},
		{
			name:     "os option long",
			input:    "operating-system",
			expected: pb.QueryOption_QUERY_OPTION_OS,
			wantErr:  false,
		},
		{
			name:     "swbom option",
			input:    "swbom",
			expected: pb.QueryOption_QUERY_OPTION_SWBOM,
			wantErr:  false,
		},
		{
			name:     "swbom option long",
			input:    "software-bom",
			expected: pb.QueryOption_QUERY_OPTION_SWBOM,
			wantErr:  false,
		},
		{
			name:     "version option",
			input:    "version",
			expected: pb.QueryOption_QUERY_OPTION_VERSION,
			wantErr:  false,
		},
		{
			name:     "version option short",
			input:    "ver",
			expected: pb.QueryOption_QUERY_OPTION_VERSION,
			wantErr:  false,
		},
		{
			name:     "all option",
			input:    "all",
			expected: pb.QueryOption_QUERY_OPTION_ALL,
			wantErr:  false,
		},
		{
			name:     "case insensitive",
			input:    "HW",
			expected: pb.QueryOption_QUERY_OPTION_HARDWARE,
			wantErr:  false,
		},
		{
			name:     "invalid option",
			input:    "invalid",
			expected: pb.QueryOption_QUERY_OPTION_UNSPECIFIED,
			wantErr:  true,
		},
		{
			name:     "empty option",
			input:    "",
			expected: pb.QueryOption_QUERY_OPTION_UNSPECIFIED,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseQueryOption(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleQueryCmd_Success(t *testing.T) {
	socket := "/test/socket"
	option := "hw"

	// Create mock response
	mockResponse := &pb.QueryResponse{
		StatusCode: 200,
		Success:    true,
		Error:      "",
		Data: &pb.QueryData{
			Timestamp: timestamppb.New(time.Now()),
			Type:      "static_telemetry",
			Values: &pb.QueryData_Hardware{
				Hardware: &pb.HardwareInfo{
					Manufacturer: "Intel Corporation",
					Product:      "Test Product",
					CpuId:        "GenuineIntel",
				},
			},
		},
	}

	// Setup mock client
	mockClient := &MockInbServiceClient{}
	mockClient.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockResponse, nil)

	// Test successful query
	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestHandleQueryCmd_DefaultOption(t *testing.T) {
	socket := "/test/socket"
	option := "" // Empty option should default to "all"

	// Create mock response for "all" option
	mockResponse := &pb.QueryResponse{
		StatusCode: 200,
		Success:    true,
		Error:      "",
		Data: &pb.QueryData{
			Timestamp: timestamppb.New(time.Now()),
			Type:      "static_telemetry",
			Values: &pb.QueryData_AllInfo{
				AllInfo: &pb.AllInfo{
					Hardware: &pb.HardwareInfo{
						Manufacturer: "Intel Corporation",
					},
				},
			},
		},
	}

	// Setup mock client - it should be called with QUERY_OPTION_ALL
	mockClient := &MockInbServiceClient{}
	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(req *pb.QueryRequest) bool {
		return req.Option == pb.QueryOption_QUERY_OPTION_ALL
	}), mock.Anything).Return(mockResponse, nil)

	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestHandleQueryCmd_InvalidOption(t *testing.T) {
	socket := "/test/socket"
	option := "invalid"

	mockClient := &MockInbServiceClient{}
	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid query option")
}

func TestHandleQueryCmd_DialError(t *testing.T) {
	socket := "/test/socket"
	option := "hw"

	dialError := errors.New("connection failed")
	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(nil, dialError))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error setting up new gRPC client")
}

func TestHandleQueryCmd_QueryError(t *testing.T) {
	socket := "/test/socket"
	option := "hw"

	// Setup mock client with error
	mockClient := &MockInbServiceClient{}
	queryError := errors.New("query failed")
	mockClient.On("Query", mock.Anything, mock.Anything, mock.Anything).Return((*pb.QueryResponse)(nil), queryError)

	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error performing query")
	mockClient.AssertExpectations(t)
}

func TestHandleQueryCmd_NotImplementedResponse(t *testing.T) {
	socket := "/test/socket"
	option := "hw"

	// Mock 501 Not Implemented response
	mockResponse := &pb.QueryResponse{
		StatusCode: 501,
		Success:    false,
		Error:      "Query method not implemented yet",
		Data:       nil,
	}

	mockClient := &MockInbServiceClient{}
	mockClient.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockResponse, nil)

	handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

	cmd := QueryCmd()
	err := handler(cmd, []string{})

	assert.NoError(t, err) // Should not error, just display the 501 response
	mockClient.AssertExpectations(t)
}

// Test all query options
func TestHandleQueryCmd_AllOptions(t *testing.T) {
	socket := "/test/socket"

	options := []string{"hw", "fw", "os", "swbom", "version", "all"}

	for _, opt := range options {
		t.Run(opt, func(t *testing.T) {
			option := opt

			mockResponse := &pb.QueryResponse{
				StatusCode: 200,
				Success:    true,
				Error:      "",
				Data: &pb.QueryData{
					Timestamp: timestamppb.New(time.Now()),
					Type:      "static_telemetry",
				},
			}

			mockClient := &MockInbServiceClient{}
			mockClient.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockResponse, nil)

			handler := handleQueryCmd(&socket, &option, mockDialerForQuery(mockClient, nil))

			cmd := QueryCmd()
			err := handler(cmd, []string{})

			assert.NoError(t, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDisplayQueryResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *pb.QueryResponse
		option   string
	}{
		{
			name: "successful hardware response",
			response: &pb.QueryResponse{
				StatusCode: 200,
				Success:    true,
				Error:      "",
				Data: &pb.QueryData{
					Timestamp: timestamppb.New(time.Now()),
					Type:      "static_telemetry",
					Values: &pb.QueryData_Hardware{
						Hardware: &pb.HardwareInfo{
							Manufacturer: "Intel Corporation",
							Product:      "Test Product",
						},
					},
				},
			},
			option: "hw",
		},
		{
			name: "error response",
			response: &pb.QueryResponse{
				StatusCode: 500,
				Success:    false,
				Error:      "Internal server error",
				Data:       nil,
			},
			option: "hw",
		},
		{
			name: "not implemented response",
			response: &pb.QueryResponse{
				StatusCode: 501,
				Success:    false,
				Error:      "Query method not implemented yet",
				Data:       nil,
			},
			option: "hw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test mainly checks that the function doesn't panic
			// In a real scenario, you might want to capture stdout to verify output
			displayQueryResponse(tt.response, tt.option)
		})
	}
}

func TestDisplayFunctions(t *testing.T) {
	// Test display functions with sample data
	t.Run("displayHardwareInfo", func(t *testing.T) {
		hw := &pb.HardwareInfo{
			Manufacturer: "Intel Corporation",
			Product:      "Test Product",
			CpuId:        "GenuineIntel",
		}
		displayHardwareInfo(hw)
		displayHardwareInfo(nil) // Test nil case
	})

	t.Run("displayFirmwareInfo", func(t *testing.T) {
		fw := &pb.FirmwareInfo{
			BiosVendor:  "Intel",
			BiosVersion: "2.0.0",
		}
		displayFirmwareInfo(fw)
		displayFirmwareInfo(nil) // Test nil case
	})

	t.Run("displayOSInfo", func(t *testing.T) {
		os := &pb.OSInfo{
			OsType:    "Linux",
			OsVersion: "Ubuntu 20.04",
		}
		displayOSInfo(os)
		displayOSInfo(nil) // Test nil case
	})

	t.Run("displaySWBOMInfo", func(t *testing.T) {
		swbom := &pb.SWBOMInfo{
			CollectionMethod: "dpkg",
			Packages: []*pb.SoftwarePackage{
				{
					Name:    "package1",
					Version: "1.0.0",
					Vendor:  "vendor1",
				},
			},
		}
		displaySWBOMInfo(swbom)
		displaySWBOMInfo(nil) // Test nil case
	})

	t.Run("displayVersionInfo", func(t *testing.T) {
		version := &pb.VersionInfo{
			Version:   "1.0.0",
			GitCommit: "1234567890abcdef",
		}
		displayVersionInfo(version)
		displayVersionInfo(nil) // Test nil case
	})

	t.Run("displayAllInfo", func(t *testing.T) {
		all := &pb.AllInfo{
			Hardware: &pb.HardwareInfo{
				Manufacturer: "Intel Corporation",
			},
			PowerCapabilities: "S0, S3, S5",
		}
		displayAllInfo(all)
		displayAllInfo(nil) // Test nil case
	})
}
