// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Unit Tests for package comms: Testing the GRPC client implementation code for the communication with
// a GRPC server.
package comms_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"testing"
	"time"

	proto "github.com/open-edge-platform/cluster-api-provider-intel/pkg/api/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/comms"
)

type mockServer struct {
	proto.ClusterOrchestratorSouthboundServer
}

type errCodeUnknownServer struct {
	proto.ClusterOrchestratorSouthboundServer
}

type errCodeNotFoundServer struct {
	proto.ClusterOrchestratorSouthboundServer
}

type mockServerErr struct {
	proto.ClusterOrchestratorSouthboundServer
}

var expectedInstallCmd = "install kubernetes engine"
var expectedUninstallCmd = "uninstall kubernetes engine"

func (srv *mockServer) RegisterCluster(_ context.Context, _ *proto.RegisterClusterRequest) (*proto.RegisterClusterResponse, error) {
	registerClusterResponse := proto.RegisterClusterResponse{
		InstallCmd:   &proto.ShellScriptCommand{Command: expectedInstallCmd},
		UninstallCmd: &proto.ShellScriptCommand{Command: expectedUninstallCmd},
	}
	return &registerClusterResponse, nil
}

func (srv *mockServer) UpdateClusterStatus(_ context.Context, _ *proto.UpdateClusterStatusRequest) (*proto.UpdateClusterStatusResponse, error) {
	updateClusterStatusResponse := proto.UpdateClusterStatusResponse{}
	return &updateClusterStatusResponse, nil
}

func (srv *errCodeUnknownServer) RegisterCluster(_ context.Context, _ *proto.RegisterClusterRequest) (*proto.RegisterClusterResponse, error) {
	return nil, status.Error(codes.Unknown, "failed to update status")
}

func (srv *errCodeUnknownServer) UpdateClusterStatus(_ context.Context, _ *proto.UpdateClusterStatusRequest) (*proto.UpdateClusterStatusResponse, error) {
	return nil, status.Error(codes.Unknown, "failed to update status")
}

func (srv *errCodeNotFoundServer) RegisterCluster(_ context.Context, _ *proto.RegisterClusterRequest) (*proto.RegisterClusterResponse, error) {
	return nil, status.Error(codes.NotFound, "")
}

func (srv *errCodeNotFoundServer) UpdateClusterStatus(_ context.Context, _ *proto.UpdateClusterStatusRequest) (*proto.UpdateClusterStatusResponse, error) {
	return nil, status.Error(codes.NotFound, "")
}

func (srv *mockServerErr) RegisterCluster(_ context.Context, _ *proto.RegisterClusterRequest) (*proto.RegisterClusterResponse, error) {
	registerClusterResponse := proto.RegisterClusterResponse{
		Res: proto.RegisterClusterResponse_ERROR,
	}
	return &registerClusterResponse, nil
}

func runMockServer(server proto.ClusterOrchestratorSouthboundServer) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	creds, _ := credentials.NewServerTLSFromFile("../../test/_dummy.crt", "../../test/_dummy.key")
	s := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterClusterOrchestratorSouthboundServer(s, server)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()
	return lis
}

// Helper function for dailing to a server using the bufconn package
func WithBufconnDialer(_ context.Context, lis *bufconn.Listener) func(*comms.Client) {
	return func(s *comms.Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		})
	}
}

func TestInvalidProtocol(t *testing.T) {
	ctx := context.Background()
	tlsConfig := &tls.Config{RootCAs: x509.NewCertPool()}
	clusterOrch := comms.NewClient("INVALID", tlsConfig)

	assert.NoError(t, clusterOrch.Connect())

	cmd, err := clusterOrch.Register(ctx, "dummy_token")
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

// Testing success of Register function using the bufconn package for simulating network.
func TestRegistration(t *testing.T) {
	ctx := context.Background()
	lis := runMockServer(&mockServer{})
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	clusterOrch := comms.NewClient("", tlsConfig, WithBufconnDialer(ctx, lis))
	assert.NoError(t, clusterOrch.Connect())

	installCmd, uninstallCmd := clusterOrch.RegisterToClusterOrch(ctx, "dummy_guid")

	assert.Equal(t, expectedInstallCmd, installCmd)
	assert.Equal(t, expectedUninstallCmd, uninstallCmd)
}

func TestFailedRegister(t *testing.T) {
	lis := runMockServer(&errCodeUnknownServer{})
	ctx := context.Background()
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err := testClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server!")
	}
	cmd, err := testClient.Register(ctx, "dummy-token")
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestRegisterErrorResponse(t *testing.T) {
	lis := runMockServer(&mockServerErr{})
	ctx := context.Background()
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err := testClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server!")
	}
	cmd, err := testClient.Register(ctx, "dummy-token")
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestFailedRegisterToClusterOrch(t *testing.T) {
	lis := runMockServer(&errCodeUnknownServer{})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err := testClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server!")
	}
	installCmd, uninstallCmd := testClient.RegisterToClusterOrch(ctx, "dummy-token")
	assert.Empty(t, installCmd)
	assert.Empty(t, uninstallCmd)
}

// Testing success of UpdateServer function.
func TestUpdateServer(t *testing.T) {
	ctx := context.Background()
	lis := runMockServer(&mockServer{})
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	clusterOrch := comms.NewClient("", tlsConfig, WithBufconnDialer(ctx, lis))
	assert.NoError(t, clusterOrch.Connect())

	cmd, err := clusterOrch.UpdateClusterStatus(ctx, "REGISTERING", "dummy_token")
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
}

func TestErrUnknownUpdate(t *testing.T) {
	lis := runMockServer(&errCodeUnknownServer{})
	ctx := context.Background()
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err := testClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server!")
	}
	cmd, err := testClient.UpdateClusterStatus(ctx, "REGISTERING", "dummy-token")
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestErrNotFoundUpdate(t *testing.T) {
	lis := runMockServer(&errCodeNotFoundServer{})
	ctx := context.Background()
	tlsConfig := &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true,
	}

	testClient := comms.NewClient("dummy-addr", tlsConfig, WithBufconnDialer(ctx, lis))
	err := testClient.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server!")
	}
	cmd, err := testClient.UpdateClusterStatus(ctx, "REGISTERING", "dummy-token")
	assert.Error(t, err)
	assert.Nil(t, cmd)
}
