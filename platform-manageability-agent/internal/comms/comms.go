package comms

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

const retryInterval = 10 * time.Second
const tickerInterval = 500 * time.Millisecond
const connTimeout = 5 * time.Second

type Client struct {
	DMMgrServiceAddr string
	Dialer           grpc.DialOption
	Transport        grpc.DialOption
	GrpcConn         *grpc.ClientConn
	DMMgrClient      pb.DeviceManagementClient
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
	cli.DMMgrClient = pb.NewDeviceManagementClient(cli.GrpcConn)
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

// parseAMTInfo parses the output of the `rpc amtinfo` command and populates the AMTStatusRequest.
func parseAMTInfo(uuid string, output []byte) (*pb.AMTStatusRequest, error) {
	var (
		status  = pb.AMTStatus_DISABLED
		version string
	)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Version") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
				status = pb.AMTStatus_ENABLED
			}
		}
	}

	req := &pb.AMTStatusRequest{
		HostId:  strings.TrimSpace(string(uuid)),
		Status:  status,
		Version: version,
	}
	return req, nil
}

// ReportAMTStatus executes the `rpc amtinfo` command, parses the output, and sends the AMT status to the server.
func (cli *Client) ReportAMTStatus(ctx context.Context) error {
	uuid, err := utils.GetSystemUUID()
	if err != nil {
		return fmt.Errorf("failed to retrieve UUID: %w", err)
	}

	output, err := utils.ExecuteWithRetries("sudo", []string{"./rpc", "amtinfo"})
	if err != nil {
		req := &pb.AMTStatusRequest{
			HostId:  uuid,
			Status:  pb.AMTStatus_DISABLED,
			Version: "",
		}
		_, reportErr := cli.DMMgrClient.ReportAMTStatus(ctx, req)
		if reportErr != nil {
			return fmt.Errorf("failed to report AMTStatus to DM Manager: %w", reportErr)
		}
		return fmt.Errorf("failed to execute `rpc amtinfo` command: %w", err)
	}

	req, err := parseAMTInfo(uuid, output)
	if err != nil {
		return fmt.Errorf("failed to parse `rpc amtinfo` output: %w", err)
	}

	_, err = cli.DMMgrClient.ReportAMTStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to report AMT status: %w", err)
	}
	log.Logger.Info("Successfully reported AMT status")
	return nil
}

// RetrieveActivationDetails retrieves activation details and executes the activation command if required.
func (cli *Client) RetrieveActivationDetails(ctx context.Context, hostID string, conf *config.Config) error {
	req := &pb.ActivationRequest{
		HostId: hostID,
	}
	resp, err := cli.DMMgrClient.RetrieveActivationDetails(ctx, req)
	if err != nil {
		return fmt.Errorf("Failed to retrieve activation details: %w", err)
	}

	log.Logger.Infof("Retrieved activation details: HostID=%s, Operation=%v, ProfileName=%s",
		resp.HostId, resp.Operation, resp.ProfileName)

	if resp.Operation == pb.OperationType_ACTIVATE {
		rpsAddress := fmt.Sprintf("wss://%s/activate", conf.RPSAddress)
		cmd := exec.Command("sudo", "rpc", "activate", "-u", rpsAddress, "-n", "-profile", resp.ProfileName)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to execute activation command: %w, Output: %s", err, string(output))
		}
		// TODO: Parse the output and send the activation result back to DM Manager.
	}
	return nil
}
