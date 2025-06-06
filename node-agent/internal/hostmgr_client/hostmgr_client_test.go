// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package hostmgr_client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"testing"

	test_util "github.com/open-edge-platform/edge-node-agents/common/pkg/testutils"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"github.com/stretchr/testify/require"

	hostmgr_client "github.com/open-edge-platform/edge-node-agents/node-agent/internal/hostmgr_client"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"
)

const testGUID = "TEST-GUID-TEST-GUID"

type mockServer struct {
	mock.Mock
	proto.UnimplementedHostmgrServer
}

type failedServer struct {
	mock.Mock
	proto.UnimplementedHostmgrServer
}

func (srv *mockServer) UpdateInstanceStateStatusByHostGUID(ctx context.Context, req *proto.UpdateInstanceStateStatusByHostGUIDRequest) (*proto.UpdateInstanceStateStatusByHostGUIDResponse, error) {
	insStatusResp := proto.UpdateInstanceStateStatusByHostGUIDResponse{}
	return &insStatusResp, nil
}

func (srv *failedServer) UpdateInstanceStateStatusByHostGUID(ctx context.Context, req *proto.UpdateInstanceStateStatusByHostGUIDRequest) (*proto.UpdateInstanceStateStatusByHostGUIDResponse, error) {
	return nil, fmt.Errorf("error in status update")
}

// Helper function for dailing to a server using the bufconn package
func WithBufconnDialer(ctx context.Context, lis *bufconn.Listener) func(*hostmgr_client.Client) {
	return func(s *hostmgr_client.Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		})
	}
}

func runMockServer(cFile string, kFile string) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	creds, _ := credentials.NewServerTLSFromFile(cFile, kFile)
	s := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterHostmgrServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("error serving: %v", err)
		}
	}()
	return lis
}

func runFailedServer(cFile string, kFile string) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	creds, _ := credentials.NewServerTLSFromFile(cFile, kFile)
	s := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterHostmgrServer(s, &failedServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("error serving: %v", err)
		}
	}()
	return lis
}

// Testing success of UpdateServer function.
func TestUpdateServer(t *testing.T) {
	ctx := context.Background()
	cFile, kFile, cPem, err := test_util.CreateTestCrt()
	require.NoError(t, err)

	lis := runMockServer(cFile, kFile)
	hostmgr := getClient(t, cPem, ctx, lis)

	err = hostmgr.UpdateInstanceStatus(ctx, proto.InstanceState_INSTANCE_STATE_RUNNING, proto.InstanceStatus_INSTANCE_STATUS_RUNNING, "edge node running")
	require.NoError(t, err)
}

func TestUpdateServerFailed(t *testing.T) {
	ctx := context.Background()
	cFile, kFile, cPem, err := test_util.CreateTestCrt()
	require.NoError(t, err)

	lis := runFailedServer(cFile, kFile)
	hostmgr := getClient(t, cPem, ctx, lis)

	err = hostmgr.UpdateInstanceStatus(ctx, proto.InstanceState_INSTANCE_STATE_INSTALLED, proto.InstanceStatus_INSTANCE_STATUS_BOOTING, "edge node booting")
	require.Error(t, err)
}

func getClient(t *testing.T, cPem []byte, ctx context.Context, lis *bufconn.Listener) *hostmgr_client.Client {
	caCertPool, err := x509.SystemCertPool()
	require.Nil(t, err)

	caCertPool.AppendCertsFromPEM(cPem)

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
		ServerName:         "test.dummy.unit",
	}

	hostmgr := hostmgr_client.NewClient(testGUID, "", tlsConfig, WithBufconnDialer(ctx, lis))
	require.NoError(t, hostmgr.Connect(ctx))
	return hostmgr
}
