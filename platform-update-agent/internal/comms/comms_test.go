// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"testing"

	mock_server "github.com/open-edge-platform/edge-node-agents/platform-update-agent/cmd/mock-server/mock-server"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/comms"

	"github.com/stretchr/testify/assert"
)

// constants and global vars
const serviceAddr = "127.0.0.1:8080"

var tlsConfig = &tls.Config{
	RootCAs:            x509.NewCertPool(),
	InsecureSkipVerify: false,
}

func init() {

	server, lis := mock_server.NewGrpcServer(serviceAddr, "../../mocks", mock_server.UBUNTU)

	go func() {
		if err := mock_server.RunGrpcServer(server, lis); err != nil {
			log.Println("Failed to run maintenance gRPC server")
		}
	}()

}

func Test_ConnectToEdgeInfrastructureManager_WhenContextIsClosedThenNoClientShouldBeReturned(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	cf()

	client := comms.ConnectToEdgeInfrastructureManager(ctx, serviceAddr, tlsConfig)
	assert.Nil(t, client)

	tlsConfig.InsecureSkipVerify = true

	client = comms.ConnectToEdgeInfrastructureManager(ctx, serviceAddr, tlsConfig)
	assert.Nil(t, client)
	tlsConfig.InsecureSkipVerify = false
}

func Test_ConnectToEdgeInfrastructureManager_ShouldReturnClientWhenContextIsntClosed(t *testing.T) {
	ctx := context.Background()

	client := comms.ConnectToEdgeInfrastructureManager(ctx, serviceAddr, tlsConfig)
	assert.NotNil(t, client)
}

func Test_PlatformUpdateStatus_ShouldReturnNoErrorWhileGrpcServerIsRunning(t *testing.T) {
	ctx := context.Background()

	tlsConfig.InsecureSkipVerify = true
	client := comms.ConnectToEdgeInfrastructureManager(ctx, "localhost:8080", tlsConfig)

	assert.NotNil(t, client)

	status := &pb.UpdateStatus{}

	// Iterate over all defined enum values
	for enumValue := range pb.UpdateStatus_StatusType_name {
		status.StatusType = pb.UpdateStatus_StatusType(enumValue)
		_, err := client.PlatformUpdateStatus(ctx, status, "")
		assert.NoError(t, err)
	}
}
