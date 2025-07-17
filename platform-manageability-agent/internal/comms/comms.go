package comms

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	dm "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

const retryInterval = 10 * time.Second
const tickerInterval = 500 * time.Millisecond
const connTimeout = 5 * time.Second

type Client struct {
	DMMgrServiceAddr string
	Dialer           grpc.DialOption
	Transport        grpc.DialOption
	GrpcConn         *grpc.ClientConn
	DMMgrClient      dm.DeviceManagementClient
	RetryInterval    time.Duration
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
	cli.DMMgrServiceAddr = serviceURL
	cli.RetryInterval = retryInterval
	cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	WithNetworkDialer(cli.DMMgrServiceAddr)(cli)
	return cli
}

func (cli *Client) Connect(ctx context.Context) (err error) {
	cli.GrpcConn, err = grpc.DialContext(ctx, cli.DMMgrServiceAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(connTimeout)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connection to %v failed: %v", cli.DMMgrServiceAddr, err)
	}
	cli.DMMgrClient = dm.NewDeviceManagementClient(cli.GrpcConn)
	return nil
}

func ConnectToDMManager(ctx context.Context, serviceAddr string, tlsConfig *tls.Config) *Client {
	dmMgr := NewClient(serviceAddr, tlsConfig)

	cyclicalTicker := time.NewTicker(tickerInterval)
	defer cyclicalTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Logger.Info("Connecting to DM Manager has been cancelled")
			return nil
		case <-cyclicalTicker.C:
			err := dmMgr.Connect(ctx)
			if err != nil {
				log.Logger.Warnf("Can't connect to DM Manager, retrying: %v", err)
				time.Sleep(dmMgr.RetryInterval)
				continue
			}
			log.Logger.Info("Successfully connected to DM Manager")
			return dmMgr
		}
	}
}
