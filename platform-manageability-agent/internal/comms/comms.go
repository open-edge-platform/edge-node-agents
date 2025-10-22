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
	"sync"
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
	retryInterval          = 10 * time.Second
	tickerInterval         = 500 * time.Millisecond
	connTimeout            = 5 * time.Second
	connectingStateTimeout = 3 * time.Minute // Timeout for AMT stuck in "connecting" state

	// AMT and ISM feature constants.
	AMTFeature                = "AMT"
	ISMFeature                = "ISM"
	ISMFeatureDetectionString = "Intel Standard Manageability Corporate"
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
	DMMgrServiceAddr         string
	Dialer                   grpc.DialOption
	Transport                grpc.DialOption
	GrpcConn                 *grpc.ClientConn
	DMMgrClient              pb.DeviceManagementClient
	RetryInterval            time.Duration
	Executor                 utils.CommandExecutor
	connectingStateStartTime *time.Time   // Track when AMT entered "connecting" state
	previousState            string       // Track the previous AMT state to detect direct transitions
	deactivationInProgress   bool         // Track if deactivation is currently in progress
	mu                       sync.RWMutex // Protects concurrent access to the fields above
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
	// TODO : Use json parser to fetch fields directly from json output
	value, ok := parseAMTInfoField(output, "Features")
	if ok {
		log.Logger.Infof("Parsed Features field value: '%s' for host %s", value, hostID)
		if strings.Contains(strings.ToUpper(value), AMTFeature) {
			log.Logger.Debugf("AMT detected in features for host %s", hostID)
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: AMTFeature,
			}
		} else if strings.Contains(value, ISMFeatureDetectionString) {
			log.Logger.Debugf("ISM detected in features for host %s", hostID)
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: ISMFeature,
			}
		} else {
			log.Logger.Debugf("Unknown feature detected: '%s' for host %s", value, hostID)
			// If features field contains other values, send empty string
			req = &pb.AMTStatusRequest{
				HostId:  hostID,
				Status:  pb.AMTStatus_ENABLED,
				Feature: "",
			}
		}
	} else {
		log.Logger.Debugf("Features field not found or empty for host %s", hostID)
		// If we can't parse the Features field, send empty string
		req = &pb.AMTStatusRequest{
			HostId:  hostID,
			Status:  pb.AMTStatus_ENABLED,
			Feature: "",
		}
	}
	// Send the AMT status to the device manager
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
		// Update previous state and clear connecting state timer
		cli.mu.Lock()
		cli.previousState = normalizedStatus
		cli.connectingStateStartTime = nil
		cli.mu.Unlock()

		// Execute activation command
		rpsAddress := fmt.Sprintf("wss://%s/activate", conf.RPSAddress)
		password := resp.ActionPassword
		activationOutput, activationErr := cli.Executor.ExecuteAMTActivate(rpsAddress, resp.ProfileName, password)

		// handles intermittent activation failures
		// and allows the main periodic timer to retry activation in the next cycle
		outputStr := string(activationOutput)
		if strings.Contains(outputStr, `msg="interrupted system call"`) ||
			strings.Contains(outputStr, "exit code: 10") {
			log.Logger.Warnf("Interrupted system call detected for host %s - retrying next cycle", hostID)
		}
		if activationErr != nil {
			log.Logger.Errorf("Failed to execute activation command for host %s: %v", hostID, activationErr)
		}

		// Check provisioning status
		ok := cli.isProvisioned(ctx, outputStr, hostID)
		if !ok {
			log.Logger.Errorf("Failed to execute activation command for host %s: %v, Output: %s",
				hostID, activationErr, string(activationOutput))
			activationStatus = pb.ActivationStatus_ACTIVATION_FAILED
		} else {
			log.Logger.Debugf("Activation command output for host %s: %s", hostID, outputStr)
			activationStatus = pb.ActivationStatus_ACTIVATING
			log.Logger.Debugf("setting activation status to %s: %s", activationStatus, hostID)
		}
	case "connecting":
		activationStatus = cli.handleConnectingState(hostID, normalizedStatus)

	case "connected":
		// Update previous state and reset connecting state timestamp
		cli.mu.Lock()
		cli.previousState = normalizedStatus
		cli.connectingStateStartTime = nil
		cli.mu.Unlock()
		activationStatus = pb.ActivationStatus_ACTIVATED
		log.Logger.Debugf("setting activation status to %s: %s", activationStatus, hostID)

	default:
		log.Logger.Warnf("Unknown RAS Remote Status for host %s: %s", hostID, rasStatus)
		cli.mu.Lock()
		cli.previousState = normalizedStatus
		cli.mu.Unlock()
		activationStatus = pb.ActivationStatus_UNSPECIFIED
		log.Logger.Debugf("setting activation status to %s: %s", activationStatus, hostID)
	}

	return cli.reportActivationResult(ctx, hostID, activationStatus)
}

// handleConnectingState manages the connecting state
// If device went directly to "connecting" (bypassing "not connected"), trigger immediate deactivation
// Otherwise, wait for 3 minutes before triggering deactivation
func (cli *Client) handleConnectingState(hostID string, currentState string) pb.ActivationStatus {
	now := time.Now()

	// Track when first entered connecting state
	cli.mu.Lock()
	if cli.connectingStateStartTime == nil {
		cli.connectingStateStartTime = &now
		log.Logger.Infof("Detected 'connecting' state for host %s at %v", hostID, now)

		// Check if device went directly to "connecting" without being "not connected" first
		// This issue observed in EKS/On-prem environment
		// Trigger immediate deactivation if
		// 1. First startup (previousState is empty)/"connecting"
		// 2. Previous state was anything other than "not connected"
		previousState := cli.previousState
		cli.mu.Unlock()

		if previousState == "" || previousState != "not connected" {
			log.Logger.Infof("Host %s: direct transition to 'connecting' from '%s', triggering deactivation",
				hostID, previousState)
			// trigger the deactivation
			return cli.triggerDeactivationAsync(hostID)
		}
	} else {
		cli.mu.Unlock()
	}

	// Update previous state to current state
	cli.mu.Lock()
	cli.previousState = currentState
	connectingStartTime := cli.connectingStateStartTime
	cli.mu.Unlock()

	// Check if been in connecting state too long (normal 3-minute timeout)
	timeInConnecting := now.Sub(*connectingStartTime)
	log.Logger.Debugf("Host %s has been in 'connecting' state for %v", hostID, timeInConnecting)

	if timeInConnecting > connectingStateTimeout {
		log.Logger.Warnf("Host %s stuck in 'connecting' state for %v (>%v), triggering deactivation",
			hostID, timeInConnecting, connectingStateTimeout)
		// trigger the deactivation
		return cli.triggerDeactivationAsync(hostID)
	}

	// Still in connecting state within timeout
	return pb.ActivationStatus_ACTIVATING
}

// triggerDeactivationAsync launches deactivation in background goroutine and returns immediately
func (cli *Client) triggerDeactivationAsync(hostID string) pb.ActivationStatus {
	cli.mu.Lock()
	// Only trigger deactivation if not already in progress
	if cli.deactivationInProgress {
		cli.mu.Unlock()
		log.Logger.Infof("Deactivation already in progress for host %s, skipping", hostID)
		return pb.ActivationStatus_ACTIVATING
	}
	// Mark deactivation as in progress
	cli.deactivationInProgress = true
	// Reset connecting state timer but DON'T change previousState yet
	cli.connectingStateStartTime = nil
	cli.mu.Unlock()

	// Launch goroutine for deactivation
	go cli.performDeactivationAsync(hostID)
	// Return ACTIVATION_FAILED to stop current activation attempts
	return pb.ActivationStatus_ACTIVATION_FAILED
}

// performDeactivationAsync executes deactivation
// and polls RAS status until "not connected" or timeout
func (cli *Client) performDeactivationAsync(hostID string) {
	log.Logger.Infof("Starting async deactivation for host %s", hostID)

	// Always reset deactivation in progress flag when done
	defer func() {
		cli.mu.Lock()
		cli.deactivationInProgress = false
		cli.mu.Unlock()
	}()

	// Execute deactivation command
	deactivateOutput, deactivateErr := cli.Executor.ExecuteAMTDeactivate()
	if deactivateErr != nil {
		log.Logger.Errorf("Deactivation command failed for host %s: %v, Output: %s",
			hostID, deactivateErr, string(deactivateOutput))
		return
	}

	log.Logger.Infof("Deactivation command executed for host %s, now polling RAS status...", hostID)

	// Poll RAS status for up to 1 minute to confirm deactivation
	const pollTimeout = 1 * time.Minute
	const pollInterval = 2 * time.Second
	startTime := time.Now()

	for {
		// Check if exceeded the timeout
		if time.Since(startTime) > pollTimeout {
			log.Logger.Errorf("Deactivation polling timeout for host %s after %v", hostID, pollTimeout)
			return
		}
		// Get current AMT info to check RAS status
		output, err := cli.Executor.ExecuteAMTInfo()
		if err != nil {
			log.Logger.Warnf("Failed to get AMT info during polling for host %s: %v", hostID, err)
			time.Sleep(pollInterval)
			continue
		}
		rasStatus, ok := parseAMTInfoField(output, "RAS Remote Status")
		if !ok {
			log.Logger.Warnf("RAS Remote Status not found during polling for host %s", hostID)
			time.Sleep(pollInterval)
			continue
		}
		normalizedStatus := strings.ToLower(strings.TrimSpace(rasStatus))
		log.Logger.Infof("Polling RAS status for host %s: %s (elapsed: %v)",
			hostID, normalizedStatus, time.Since(startTime))
		// Check if deactivation succeeded (status is "not connected")
		if normalizedStatus == "not connected" {
			log.Logger.Infof("Deactivation successful for host %s - RAS status: %s (elapsed: %v)",
				hostID, normalizedStatus, time.Since(startTime))
			// Update previousState to "not connected" for next activation cycle
			cli.mu.Lock()
			cli.previousState = "not connected"
			cli.mu.Unlock()
			return
		}
		// Continue polling - sleep before next check
		time.Sleep(pollInterval)
	}
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

// SetConnectingStateStartTime sets the connecting state start
// time for testing purposes.
func (cli *Client) SetConnectingStateStartTime(t time.Time) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	cli.connectingStateStartTime = &t
}

// GetConnectingStateStartTime gets the connecting state start time for testing
func (cli *Client) GetConnectingStateStartTime() *time.Time {
	cli.mu.RLock()
	defer cli.mu.RUnlock()
	return cli.connectingStateStartTime
}

// Test helper methods - only for testing
func (cli *Client) SetPreviousState(state string) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	cli.previousState = state
}

func (cli *Client) GetPreviousState() string {
	cli.mu.RLock()
	defer cli.mu.RUnlock()
	return cli.previousState
}

func (cli *Client) SetDeactivationInProgress(inProgress bool) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	cli.deactivationInProgress = inProgress
}

func (cli *Client) GetDeactivationInProgress() bool {
	cli.mu.RLock()
	defer cli.mu.RUnlock()
	return cli.deactivationInProgress
}

func (cli *Client) TriggerDeactivationAsync(hostID string) pb.ActivationStatus {
	return cli.triggerDeactivationAsync(hostID)
}

func (cli *Client) PerformDeactivationAsync(hostID string) {
	cli.performDeactivationAsync(hostID)
}
