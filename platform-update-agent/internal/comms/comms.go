// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const RETRY_INTERVAL = 10 * time.Second
const TICKER_INTERVAL = 500 * time.Millisecond

var log = logger.Logger()

type Client struct {
	MMServiceAddr string
	Dialer        grpc.DialOption
	Transport     grpc.DialOption
	GrpcConn      *grpc.ClientConn
	MMClient      pb.MaintmgrServiceClient
	RetryInterval time.Duration
}

func WithNetworkDialer(serviceAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", serviceAddr)
		})
	}
}

func NewClient(serviceURL string, tlsConfig *tls.Config) *Client {
	cli := &Client{}
	cli.MMServiceAddr = serviceURL
	cli.RetryInterval = RETRY_INTERVAL
	cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	WithNetworkDialer(cli.MMServiceAddr)(cli)

	return cli
}

// FIXME: SA1019: grpc.DialContext is deprecated: use NewClient instead.
func (cli *Client) Connect(ctx context.Context) (err error) {
	cli.GrpcConn, err = grpc.DialContext(ctx, cli.MMServiceAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(log)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connection to %v failed: %v", cli.MMServiceAddr, err)
	}
	cli.MMClient = pb.NewMaintmgrServiceClient(cli.GrpcConn)
	return nil
}

func ConnectToEdgeInfrastructureManager(ctx context.Context, serviceAddr string, tlsConfig *tls.Config) *Client {
	maintnMngr := NewClient(serviceAddr, tlsConfig)

	cyclicalTicker := time.NewTicker(TICKER_INTERVAL)
	for {
		select {
		case <-ctx.Done():
			log.Info("Connecting to Maintenance Manager has been canceled")
			return nil
		case <-cyclicalTicker.C:
			err := maintnMngr.Connect(ctx)
			if err != nil {
				log.Infof("Can't connect to Maintenance Manager: %v", err)
				time.Sleep(maintnMngr.RetryInterval)
				continue
			}
			return maintnMngr
		}

	}
}

func (cli *Client) PlatformUpdateStatus(ctx context.Context, status *pb.UpdateStatus, hostGUID string) (*pb.PlatformUpdateStatusResponse, error) {
	// TODO handle new downloading/downloaded statuses?

	request := pb.PlatformUpdateStatusRequest{HostGuid: hostGUID, UpdateStatus: status}

	response, err := cli.MMClient.PlatformUpdateStatus(ctx, &request)
	if err != nil {
		log.Errorf("The protobuf client PlatformUpdateStatus function failed! Error: %v", err)
		return nil, err
	}
	log.Infof("PlatformUpdateStatusRequest sent successfully. Update Status sent: %v.", request.UpdateStatus.StatusType)
	log.Infof("Update Log sent: %v.", request.UpdateStatus.StatusDetail)

	log.Debug("The PlatformUpdateStatus response received:\n")
	log.Debugf("Received Schedule: %s\n", response.GetUpdateSchedule().String())
	log.Debugf("Received Sources: %s\n", response.GetUpdateSource().String())
	return response, nil
}
