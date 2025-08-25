/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"errors"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestHandleConfigLoadCmd_Success(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	var hashAlgorithm string
	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "LoadConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigLoadCmd_MissingURI(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := ""
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	var hashAlgorithm string
	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uri is required")
}

func TestHandleConfigGetCmd_MissingPath(t *testing.T) {
	socket := "/tmp/test.sock"
	path := ""

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigGetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleConfigSetCmd_InvalidPathFormat(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "invalidpathformat"

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigSetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path must be in the format 'key:value'")
}

func TestHandleConfigAppendCmd_MissingPath(t *testing.T) {
	socket := "/tmp/test.sock"
	path := ""

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigAppendCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleConfigRemoveCmd_MissingPath(t *testing.T) {
	socket := "/tmp/test.sock"
	path := ""

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigRemoveCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleConfigSetCmd_MissingPath(t *testing.T) {
	socket := "/tmp/test.sock"
	path := ""

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigSetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleConfigAppendCmd_InvalidPathFormat(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "invalidpathformat"

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigAppendCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path must be in the format 'key:value'")
}

func TestHandleConfigRemoveCmd_InvalidPathFormat(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "invalidpathformat"

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigRemoveCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path must be in the format 'key:value'")
}

func TestHandleConfigLoadCmd_DialError(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return nil, nil, errors.New("mock dial error")
	}

	var hashAlgorithm string
	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock dial error")
}

func TestHandleConfigLoadCmd_GrpcError(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{}, errors.New("grpc error"))

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	var hashAlgorithm string
	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc error")
}

func TestHandleConfigGetCmd_Success(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "publishIntervalSeconds"

	mockClient := &MockInbServiceClient{}
	mockClient.On("GetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.GetConfigResponse{StatusCode: 200, Error: "", Success: true, Value: "30", Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigGetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigGetCmd_GrpcError(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "publishIntervalSeconds"

	mockClient := &MockInbServiceClient{}
	mockClient.On("GetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.GetConfigResponse{}, errors.New("grpc error"))

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigGetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc error")
}

func TestHandleConfigSetCmd_Success(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "maxCacheSize:100"

	mockClient := &MockInbServiceClient{}
	mockClient.On("SetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigSetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "SetConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigSetCmd_GrpcError(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "maxCacheSize:100"

	mockClient := &MockInbServiceClient{}
	mockClient.On("SetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{}, errors.New("grpc error"))

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigSetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc error")
}

func TestHandleConfigAppendCmd_Success(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("AppendConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigAppendCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "AppendConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigAppendCmd_GrpcError(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("AppendConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{}, errors.New("grpc error"))

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigAppendCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc error")
}

func TestHandleConfigRemoveCmd_Success(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("RemoveConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigRemoveCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "RemoveConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigRemoveCmd_GrpcError(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("RemoveConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{}, errors.New("grpc error"))

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigRemoveCmd(&socket, &path, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc error")
}

func TestHandleConfigSetCmd_MultiKeyValue(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "maxCacheSize:100;publishIntervalSeconds:10"

	mockClient := &MockInbServiceClient{}
	mockClient.On("SetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigSetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "SetConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigAppendCmd_MultiKeyValue(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/;trustedRepositories:https://def.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("AppendConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigAppendCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "AppendConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigRemoveCmd_MultiKeyValue(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "trustedRepositories:https://abc.com/;trustedRepositories:https://def.com/"

	mockClient := &MockInbServiceClient{}
	mockClient.On("RemoveConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigRemoveCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "RemoveConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigLoadCmd_EmptySignature(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := ""

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	var hashAlgorithm string
	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "LoadConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigGetCmd_MultiKey(t *testing.T) {
	socket := "/tmp/test.sock"
	path := "maxCacheSize;publishIntervalSeconds"

	mockClient := &MockInbServiceClient{}
	mockClient.On("GetConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.GetConfigResponse{StatusCode: 200, Error: "", Success: true, Value: "100;10", Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigGetCmd(&socket, &path, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleConfigLoadCmd_HashAlgorithm_PassedAndUsed(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hashAlgorithm := "sha512"

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.MatchedBy(func(req *pb.LoadConfigRequest) bool {
		return req.HashAlgorithm == "sha512"
	}), mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "LoadConfig", mock.Anything, mock.MatchedBy(func(req *pb.LoadConfigRequest) bool {
		return req.HashAlgorithm == "sha512"
	}), mock.Anything)
}

func TestHandleConfigLoadCmd_HashAlgorithm_Invalid(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hashAlgorithm := "invalidalgo"

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 400, Error: "invalid hash algorithm", Success: false, Message: ""}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hash algorithm")
}

func TestHandleConfigLoadCmd_DefaultHashAlgorithm(t *testing.T) {
	socket := "/tmp/test.sock"
	uri := "file:///tmp/intel_manageability.conf"
	signature := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	var hashAlgorithm string // not set, should default to sha384

	mockClient := &MockInbServiceClient{}
	mockClient.On("LoadConfig", mock.Anything, mock.MatchedBy(func(req *pb.LoadConfigRequest) bool {
		return req.HashAlgorithm == "" || req.HashAlgorithm == "sha384"
	}), mock.Anything).
		Return(&pb.ConfigResponse{StatusCode: 200, Error: "", Success: true, Message: "OK"}, nil)

	dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
		return mockClient, &MockClientConn{}, nil
	}

	cmd := &cobra.Command{}
	err := handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, dialer)(cmd, []string{})
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "LoadConfig", mock.Anything, mock.MatchedBy(func(req *pb.LoadConfigRequest) bool {
		return req.HashAlgorithm == "" || req.HashAlgorithm == "sha384"
	}), mock.Anything)
}
