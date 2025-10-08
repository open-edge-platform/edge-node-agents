// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	log "github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

const (
	retryInterval  = 10 * time.Second
	tickerInterval = 500 * time.Millisecond
	connTimeout    = 5 * time.Second
)

var ErrActivationSkipped = errors.New("activation skipped")

// TO DO: Implement proper parsing of AMT info json output
// AMTInfo represents the JSON structure returned by "rpc amtinfo -json"
/*
type AMTInfo struct {
    Version     string `json:"version"`
    BuildNumber string `json:"buildNumber"`
    ControlMode string `json:"controlMode"`
    DNSSuffix   string `json:"dnsSuffix"`
    Features    string `json:"features"`
    RAS         struct {
        NetworkStatus string `json:"networkStatus"`
        RemoteStatus  string `json:"remoteStatus"`
        RemoteTrigger string `json:"remoteTrigger"`
        MPSHostname   string `json:"mpsHostname"`
    } `json:"ras"`
    SKU  string `json:"sku"`
    UUID string `json:"uuid"`
}
*/

type Client struct {
	DMMgrServiceAddr string
	Dialer           grpc.DialOption
	Transport        grpc.DialOption
	GrpcConn         *grpc.ClientConn
	DMMgrClient      pb.DeviceManagementClient
	RetryInterval    time.Duration
	Executor         utils.CommandExecutor
}

func WithNetworkDialer(serviceAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", serviceAddr)
		})
	}
}

func NewClient(serviceURL string, tlsConfig *tls.Config, options ...func(*Client)) *Client {
	cli := &Client{}
	cli.DMMgrServiceAddr = serviceURL
	cli.RetryInterval = retryInterval
	cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	cli.Executor = &utils.RealCommandExecutor{}

	WithNetworkDialer(cli.DMMgrServiceAddr)(cli)

	// options can be used to override default values, e.g. from unit tests
	for _, o := range options {
		o(cli)
	}
	return cli
}

func (cli *Client) Connect(ctx context.Context) (err error) {
	cli.GrpcConn, err = grpc.DialContext(ctx, cli.DMMgrServiceAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(connTimeout)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connection to %v failed: %w", cli.DMMgrServiceAddr, err)
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
			log.Logger.Info("Connecting to DM Manager has been canceled")
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

func parseAMTInfoField(output []byte, parseKey string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, parseKey) {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), true
			}
		}
	}
	return "", false
}

// ReportAMTStatus executes the `rpc amtinfo` command, parses the output, and sends the AMT status to the server.
func (cli *Client) ReportAMTStatus(ctx context.Context, hostID string) (pb.AMTStatus, error) {
	defaultStatus := pb.AMTStatus_DISABLED
	var req *pb.AMTStatusRequest

	// TODO: Implement proper parsing of AMT info json output
	output, err := cli.Executor.ExecuteAMTInfo()
	if err != nil {
		req = &pb.AMTStatusRequest{
			HostId:  hostID,
			Status:  defaultStatus,
			Feature: "",
		}
		_, reportErr := cli.DMMgrClient.ReportAMTStatus(ctx, req)
		if reportErr != nil {
			if strings.Contains(string(output), "HECIDriverNotDetected") {
				log.Logger.Warnf("HECIDriver not detected. vPRO is not enabled on host")
				return defaultStatus, nil
			}
			return defaultStatus, fmt.Errorf("failed to report AMTStatus to DM Manager: %w", reportErr)
		}
		return defaultStatus, fmt.Errorf("failed to execute `rpc amtinfo` command: %w", err)
	}

	// Parse Features field to determine if AMT or ISM is enabled
	// TODO : Use json parser to fetch filed directly from json output
	value, ok := parseAMTInfoField(output, "Features")
	if ok {
		if strings.Contains(value, "AMT Pro Corporate") {
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: "AMT",
			}
		} else if strings.Contains(value, "Intel Standard Manageability Corporate") {
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: "ISM",
			}
		} else {
			// If features field contains other values, send empty string
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: "",
			}
		}
	} else {
		// If we can't parse the Features field, send empty string
		req = &pb.AMTStatusRequest{
			HostId:  hostID,
			Status:  pb.AMTStatus_ENABLED,
			Feature: "",
		}
	}

	_, err = cli.DMMgrClient.ReportAMTStatus(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.FailedPrecondition:
				log.Logger.Debugf("Received %v, %v", st.Message(), err.Error())
				return req.Status, nil
			}
		}
		return defaultStatus, fmt.Errorf("failed to report AMT status: %w", err)
	}

	log.Logger.Infof("Reported AMT status: HostID=%s, Status=%v, Feature=%s",
		req.HostId, req.Status, req.Feature)
	return req.Status, nil
}

// RetrieveActivationDetails retrieves activation details and executes the activation command if required.
func (cli *Client) RetrieveActivationDetails(ctx context.Context, hostID string, conf *config.Config) error {
	req := &pb.ActivationRequest{
		HostId: hostID,
	}
	resp, err := cli.DMMgrClient.RetrieveActivationDetails(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.FailedPrecondition:
				return fmt.Errorf("%w: host %s precondition failed - %v", ErrActivationSkipped, hostID, st.Message())
			}
		}
		return fmt.Errorf("failed to retrieve activation details for host %s: %w", hostID, err)
	}

	log.Logger.Debugf("Retrieved activation details: HostID=%s, Operation=%v, ProfileName=%s",
		resp.HostId, resp.Operation, resp.ProfileName)

	// Skip activation if operation is not ACTIVATE
	if resp.Operation != pb.OperationType_ACTIVATE {
		return fmt.Errorf("activation not requested for host %s: %w", hostID, err)
	}

	// Get AMT info to check RAS Remote Status
	output, err := cli.Executor.ExecuteAMTInfo()
	if err != nil {
		log.Logger.Warnf("Failed to execute AMT info check for host %s: %v", hostID, err)
		return cli.reportActivationResult(ctx, hostID, pb.ActivationStatus_ACTIVATION_FAILED)
	}

	rasStatus, ok := parseAMTInfoField(output, "RAS Remote Status")
	if !ok {
		log.Logger.Warnf("RAS Remote Status not found in AMT info output for host: %s", hostID)
		return cli.reportActivationResult(ctx, hostID, pb.ActivationStatus_ACTIVATION_FAILED)
	}

	normalizedStatus := strings.ToLower(strings.TrimSpace(rasStatus))
	log.Logger.Debugf("Current RAS Remote Status for host %s: %s", hostID, rasStatus)

	// Determine activation status based on RAS Remote Status
	var activationStatus pb.ActivationStatus
	switch normalizedStatus {
	case "not connected":
		// Execute activation command
		rpsAddress := fmt.Sprintf("wss://%s/activate", conf.RPSAddress)
		password := resp.ActionPassword
		activationOutput, activationErr := cli.Executor.ExecuteAMTActivate(rpsAddress, resp.ProfileName, password)
		ok := cli.isProvisioned(ctx, string(activationOutput), hostID)
		if !ok {
			log.Logger.Errorf("Failed to execute activation command for host %s: %v, Output: %s",
				hostID, activationErr, string(activationOutput))
			activationStatus = pb.ActivationStatus_ACTIVATION_FAILED
		} else {
			log.Logger.Debugf("Activation command output for host %s: %s", hostID, string(activationOutput))
			// Check if activation was successful by looking for success indicators in output
			activationStatus = pb.ActivationStatus_ACTIVATING
			log.Logger.Debugf("setting activation status to %s: %s", activationStatus, hostID)
		}
	case "connecting":
		activationStatus = pb.ActivationStatus_ACTIVATING
		log.Logger.Debugf("setting activation status to %s: %s", activationStatus, hostID)
	case "connected":
		activationStatus = pb.ActivationStatus_ACTIVATED
		log.Logger.Debugf("setting activation status to  %s: %s", activationStatus, hostID)
	default:
		log.Logger.Warnf("Unknown RAS Remote Status for host %s: %s", hostID, rasStatus)
		activationStatus = pb.ActivationStatus_UNSPECIFIED
		log.Logger.Debugf("setting activation status to  %s: %s", activationStatus, hostID)
	}

	return cli.reportActivationResult(ctx, hostID, activationStatus)
}

// reportActivationResult reports the activation result to the DM Manager
func (cli *Client) reportActivationResult(ctx context.Context, hostID string, status pb.ActivationStatus) error {
	req := &pb.ActivationResultRequest{
		HostId:           hostID,
		ActivationStatus: status,
	}

	_, err := cli.DMMgrClient.ReportActivationResults(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to report activation results for host %s: %w", hostID, err)
	}

	log.Logger.Infof("Reported activation results: HostID=%s, ActivationStatus=%v",
		hostID, req.ActivationStatus)
	return nil
}

// isProvisioned checks if the output contains the line indicating provisioning success.
func (cli *Client) isProvisioned(ctx context.Context, output string, hostID string) bool {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `msg="CIRA: Configured"`) {
			op := func() error {
				output_info, err := cli.Executor.ExecuteAMTInfo()
				if err != nil {
					log.Logger.Warnf("Failed to execute AMT info check for host %s: %v", hostID, err)
					return err
				}

				rasStatus, _ := parseAMTInfoField(output_info, "RAS Remote Status")
				normalizedStatus := strings.ToLower(strings.TrimSpace(rasStatus))
				log.Logger.Debugf("Current RAS Remote Status for host %s: %s", hostID, normalizedStatus)
				if (normalizedStatus != "connecting") && (normalizedStatus != "connected") {
					log.Logger.Warnf("RAS Remote Status not found in AMT info output for host: %s", hostID)
					return fmt.Errorf("RAS Remote Status not found retry AMTInfo: %s", hostID)
				}
				return nil
			}
			err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
			if err != nil {
				log.Logger.Warnf("Failed to execute AMT info check for host %s: %v", hostID, err)
				return false
			}
			log.Logger.Debugf("is Provisioning passed %s", hostID)
			return true
		}
	}
	log.Logger.Debugf("is Provisioning failed %s", hostID)
	return false
}
