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

// parseAMTInfo parses the output of the `rpc amtinfo` command and populates the AMTStatusRequest.
// func parseAMTInfo(uuid string, output []byte, parse_string string) *pb.AMTStatusRequest {
// 	var (
// 		status  = pb.AMTStatus_DISABLED
// 		version string
// 	)

// 	scanner := bufio.NewScanner(strings.NewReader(string(output)))
// 	for scanner.Scan() {
// 		line := scanner.Text()

// 		if strings.HasPrefix(line, "Version") {
// 			parts := strings.Split(line, ":")
// 			if len(parts) > 1 {
// 				version = strings.TrimSpace(parts[1])
// 				status = pb.AMTStatus_ENABLED
// 			}
// 		}
// 	}

// 	req := &pb.AMTStatusRequest{
// 		HostId:  uuid,
// 		Status:  status,
// 		Version: version,
// 	}
// 	return req
// }

// ReportAMTStatus executes the `rpc amtinfo` command, parses the output, and sends the AMT status to the server.
func (cli *Client) ReportAMTStatus(ctx context.Context, hostID string) (pb.AMTStatus, error) {
	defaultStatus := pb.AMTStatus_DISABLED
	var req *pb.AMTStatusRequest
	output, err := cli.Executor.ExecuteAMTInfo()
	if err != nil {
		req = &pb.AMTStatusRequest{
			HostId:  hostID,
			Status:  defaultStatus,
			Version: "",
		}
		_, reportErr := cli.DMMgrClient.ReportAMTStatus(ctx, req)
		if reportErr != nil {
			return defaultStatus, fmt.Errorf("failed to report AMTStatus to DM Manager: %w", reportErr)
		}
		return defaultStatus, fmt.Errorf("failed to execute `rpc amtinfo` command: %w", err)
	}

	value, ok := parseAMTInfoField(output, "Version")
	if ok {
		req = &pb.AMTStatusRequest{
			HostId:  hostID,
			Status:  pb.AMTStatus_ENABLED,
			Version: value,
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

	log.Logger.Infof("Reported AMT status: HostID=%s, Status=%v, Version=%s",
		req.HostId, req.Status, req.Version)
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

	if resp.Operation == pb.OperationType_ACTIVATE {
		rpsAddress := fmt.Sprintf("wss://%s/activate", conf.RPSAddress)
		// TODO:
		// This is a placeholder, replace with actual logic to fetch the password.
		// Need to check how to fetch the password from dm-manager, hardcoded for now.
		password := resp.ActionPassword
		output, err := cli.Executor.ExecuteAMTActivate(rpsAddress, resp.ProfileName, password)
		if err != nil {
			return fmt.Errorf("failed to execute activation command for host %s: %w, Output: %s",
				hostID, err, string(output))
		}
		log.Logger.Debugf("Activation command output for host %s: %s", hostID, string(output))

		// Monitor RAS Remote Status with 3-minute timeout
		activationStatus, err := cli.monitorActivationStatus(ctx, hostID)
		if err != nil {
			return fmt.Errorf("failed to monitor activation status for host %s: %w", hostID, err)
		}

		req := &pb.ActivationResultRequest{
			HostId:           hostID,
			ActivationStatus: activationStatus,
		}

		_, err = cli.DMMgrClient.ReportActivationResults(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to report activation results for host %s: %w", hostID, err)
		}
		log.Logger.Debugf("Reported activation results: HostID=%s, ActivationStatus=%v",
			hostID, req.ActivationStatus)
	}
	return nil
}

// monitorActivationStatus monitors the RAS Remote Status for up to 3 minutes
func (cli *Client) monitorActivationStatus(ctx context.Context, hostID string) (pb.ActivationStatus, error) {
	const (
		monitorTimeout = 3 * time.Minute
		checkInterval  = 2 * time.Second
	)

	// Create a timeout context for monitoring
	monitorCtx, cancel := context.WithTimeout(ctx, monitorTimeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Logger.Infof("Starting activation status monitoring for host: %s", hostID)

	for {
		select {
		case <-monitorCtx.Done():
			// Timeout reached, perform final check
			output, err := cli.Executor.ExecuteAMTInfo()
			if err != nil {
				log.Logger.Warnf("Failed to execute final AMT info check for host %s: %v", hostID, err)
				return pb.ActivationStatus_ACTIVATION_FAILED, nil
			}

			rasStatus, ok := parseAMTInfoField(output, "RAS Remote Status")
			if !ok {
				log.Logger.Warnf("RAS Remote Status not found in AMT info output for host: %s", hostID)
				return pb.ActivationStatus_ACTIVATION_FAILED, nil
			}

			log.Logger.Infof("Activation monitoring timeout reached for host: %s, final status: %s", hostID, rasStatus)

			switch strings.ToLower(strings.TrimSpace(rasStatus)) {
			case "connected":
				log.Logger.Infof("Activation successful for host: %s", hostID)
				return pb.ActivationStatus_ACTIVATED, nil
			case "connecting":
				log.Logger.Infof("Activation timeout - still connecting for host: %s", hostID)
				return pb.ActivationStatus_ACTIVATION_FAILED, nil
			default:
				log.Logger.Infof("Activation failed for host: %s, status: %s", hostID, rasStatus)
				return pb.ActivationStatus_ACTIVATION_FAILED, nil
			}

		case <-ticker.C:
			output, err := cli.Executor.ExecuteAMTInfo()
			if err != nil {
				log.Logger.Warnf("Failed to execute AMT info during monitoring for host %s: %v", hostID, err)
				continue
			}

			rasStatus, ok := parseAMTInfoField(output, "RAS Remote Status")
			if !ok {
				log.Logger.Warnf("RAS Remote Status not found in AMT info output for host: %s", hostID)
				continue
			}

			normalizedStatus := strings.ToLower(strings.TrimSpace(rasStatus))
			log.Logger.Debugf("Current RAS Remote Status for host %s: %s", hostID, rasStatus)

			switch normalizedStatus {
			case "connected":
				log.Logger.Infof("Activation successful for host: %s", hostID)
				return pb.ActivationStatus_ACTIVATED, nil
			case "connecting":
				log.Logger.Debugf("Host %s is still connecting, continuing to monitor...", hostID)
				req := &pb.ActivationResultRequest{
					HostId:           hostID,
					ActivationStatus: pb.ActivationStatus_ACTIVATING,
				}
				_, err = cli.DMMgrClient.ReportActivationResults(ctx, req)
				if err != nil {
					return pb.ActivationStatus_ACTIVATION_FAILED, err
				}
			case "not connected":
				log.Logger.Infof("Activation failed for host: %s - not connected", hostID)
				return pb.ActivationStatus_ACTIVATION_FAILED, nil
			default:
				log.Logger.Debugf("Unknown RAS Remote Status for host %s: %s, continuing to monitor...", hostID, rasStatus)
				// Continue monitoring for unknown states
			}
		}
	}
}

// // isProvisioned checks if the output contains the line indicating provisioning success.
// func isProvisioned(output string) bool {
// 	scanner := bufio.NewScanner(strings.NewReader(output))
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.Contains(line, `msg="CIRA: Configured"`) {
// 			return true
// 		}
// 	}
// 	return false
// }
