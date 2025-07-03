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

func TestFOTACmd(t *testing.T) {
	cmd := FOTACmd()

	assert.Equal(t, "fota", cmd.Use, "command use should be 'fota'")
	assert.Equal(t, "Performs Firmware Update", cmd.Short, "command short description should match")
	assert.Equal(t, `Updates the firmware on the device.`, cmd.Long, "command long description should match")

	flags := cmd.Flags()

	socket, err := flags.GetString("socket")
	assert.NoError(t, err)
	assert.Equal(t, "/var/run/inbd.sock", socket, "default socket should be '/var/run/inbd.sock'")

	url, err := flags.GetString("uri")
	assert.NoError(t, err)
	assert.Equal(t, "", url, "default uri should be empty")

	releaseDate, err := flags.GetString("releasedate")
	assert.NoError(t, err)
	assert.Equal(t, "", releaseDate, "default release date should be empty")

	toolOptions, err := flags.GetString("tooloptions")
	assert.NoError(t, err)
	assert.Equal(t, "", toolOptions, "default tool options should be empty")

	reboot, err := flags.GetBool("reboot")
	assert.NoError(t, err)
	assert.Equal(t, true, reboot, "default reboot should be true")

	userName, err := flags.GetString("username")
	assert.NoError(t, err)
	assert.Empty(t, userName, "default username should be an empty string")
}

func TestHandleFOTA(t *testing.T) {
	socket := "/var/run/inbd.sock"
	url := "http://example.com/package"
	releaseDate := "2025-03-13T00:00:00Z"
	toolOptions := "tooloptions"
	reboot := true
	userName := ""
	signature := ""
	cmd := &cobra.Command{}
	args := []string{}

	t.Run("successful FOTA", func(t *testing.T) {
		mockClient := new(MockInbServiceClient)
		mockClient.On("UpdateFirmware", mock.Anything, mock.Anything, mock.Anything).Return(&pb.UpdateResponse{
			StatusCode: 200,
			Error:      "",
		}, nil)

		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, false)
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &signature, dialer)(cmd, args)
		assert.NoError(t, err, "handleFOTA should not return an error")

		mockClient.AssertExpectations(t)
	})
	t.Run("invalid signature format", func(t *testing.T) {
		badSignature := "notAHexSignature"
		mockClient := new(MockInbServiceClient)
		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, false)
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &badSignature, dialer)(cmd, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature does not match expected format")
	})

	t.Run("valid signature format", func(t *testing.T) {
		validSignature := "0daaeaf170bf62c2d5c764505ff9693620b71476f41e38851601fb5a7b812b3d"
		mockClient := new(MockInbServiceClient)
		mockClient.On("UpdateFirmware", mock.Anything, mock.Anything, mock.Anything).Return(&pb.UpdateResponse{
			StatusCode: 200,
			Error:      "",
		}, nil)

		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, false)
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &validSignature, dialer)(cmd, args)
		assert.NoError(t, err, "handleFOTA should not return an error for valid signature")
		mockClient.AssertExpectations(t)
	})
	t.Run("invalid release date", func(t *testing.T) {
		invalidReleaseDate := "invalid-date"
		mockClient := new(MockInbServiceClient)
		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, false)
		}

		err := handleFOTA(&socket, &url, &invalidReleaseDate,
			&toolOptions, &reboot, &userName, &signature, dialer)(cmd, args)
		assert.Error(t, err, "error parsing release date: parsing time")
	})

	t.Run("gRPC client setup error", func(t *testing.T) {
		mockClient := new(MockInbServiceClient)
		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, true)
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions,
			&reboot, &userName, &signature, dialer)(cmd, args)
		assert.Error(t, err, "error setting up new gRPC client")
	})

	t.Run("gRPC UpdateFirmware error", func(t *testing.T) {
		mockClient := new(MockInbServiceClient)
		mockClient.On("UpdateFirmware", mock.Anything, mock.Anything, mock.Anything).Return(&pb.UpdateResponse{}, errors.New("error updating firmware"))

		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return MockDialer(ctx, socket, mockClient, false)
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &signature, dialer)(cmd, args)
		assert.Error(t, err, "error updating firmware")
	})

	t.Run("gRPC connection close error is handled", func(t *testing.T) {
		mockClient := new(MockInbServiceClient)
		mockClient.On("UpdateFirmware", mock.Anything, mock.Anything, mock.Anything).Return(&pb.UpdateResponse{
			StatusCode: 200,
			Error:      "",
		}, nil)

		dialer := func(ctx context.Context, socket string) (pb.InbServiceClient, grpc.ClientConnInterface, error) {
			return mockClient, &mockConnWithCloseError{}, nil
		}

		err := handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &signature, dialer)(cmd, args)
		assert.NoError(t, err, "handleFOTA should not return an error even if Close fails")
	})
}
