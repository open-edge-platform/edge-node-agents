// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package hostmgr_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

const NUM_RETRIES = 3
const CONN_TIMEOUT = 5 * time.Second

type Client struct {
	HostGUID              string
	ServerAddr            string
	Dialer                grpc.DialOption
	Transport             grpc.DialOption
	GrpcConn              *grpc.ClientConn
	InfraSouthboundClient proto.HostmgrClient
}

// Initialize logger
var log = logger.Logger

// Helper function for dailing to the server.
func WithNetworkDialer(serverAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", serverAddr)
		})
	}
}

// NewClient creates grpc client to Edge Infrastructure Manager (Hostmgr) southbound API
// by default it uses tcp network dialer
func NewClient(guid string, serverAddr string, tlsConfig *tls.Config, options ...func(*Client)) *Client {
	cli := &Client{}
	cli.ServerAddr = serverAddr
	cli.HostGUID = guid
	cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	WithNetworkDialer(serverAddr)(cli)

	// options can be used to override default values, e.g. from unit tests
	for _, o := range options {
		o(cli)
	}
	return cli
}

func codeToLevel(_ codes.Code) logrus.Level {
	return logrus.DebugLevel
}

// FIXME: SA1019: grpc.DialContext is deprecated: use NewClient instead.
func (cli *Client) Connect(ctx context.Context) (err error) {
	cli.GrpcConn, err = grpc.DialContext(ctx, cli.ServerAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(log, grpc_logrus.WithLevels(codeToLevel))),
		grpc.WithStreamInterceptor(grpc_logrus.StreamClientInterceptor(log, grpc_logrus.WithLevels(codeToLevel))),
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(CONN_TIMEOUT)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connection to %v failed: %v", cli.ServerAddr, err)
	}
	cli.InfraSouthboundClient = proto.NewHostmgrClient(cli.GrpcConn)
	return nil
}

func ConnectToHostMgr(ctx context.Context, guid string, serverAddr string, tlsConfig *tls.Config) (*Client, error) {
	infraMgr := NewClient(guid, serverAddr, tlsConfig)
	op := func() error {
		return infraMgr.Connect(ctx)
	}
	err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
	if err != nil {
		log.Errorf("cannot connect to Host Manager: %v", err)
		return nil, err
	}
	return infraMgr, nil

}

// UpdateInstanceStatus client method sends UpdateInstanceStateStatusByHostGUIDRequest message to the server & receives UpdateInstanceStateStatusByHostGUIDResponse message
func (cli *Client) UpdateInstanceStatus(ctx context.Context, insState proto.InstanceState, insStatus proto.InstanceStatus, insDetails string) error {
	updateInstanceStatusRequest := proto.UpdateInstanceStateStatusByHostGUIDRequest{
		HostGuid:             cli.HostGUID,
		InstanceState:        insState,
		InstanceStatus:       insStatus,
		ProviderStatusDetail: insDetails,
	}

	op := func() error {
		_, err := cli.InfraSouthboundClient.UpdateInstanceStateStatusByHostGUID(ctx, &updateInstanceStatusRequest)
		if err != nil {
			log.Errorf("UpdateInstanceStatus failed with error: %v", err)
			return err
		}
		return nil
	}
	err := backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), NUM_RETRIES))
	if err != nil {
		log.Errorf("will try to reconnect because of failure: %v", err)
		conn_err := cli.GrpcConn.Close()
		if conn_err != nil {
			return conn_err
		}
		conn_err = cli.Connect(ctx)
		if conn_err != nil {
			return conn_err
		}
		return err
	}

	log.Infof("UpdateInstanceStatus sent successfully: %s", insStatus.String())

	return nil
}
