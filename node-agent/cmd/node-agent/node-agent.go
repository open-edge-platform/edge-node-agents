// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	statusService "github.com/open-edge-platform/edge-node-agents/node-agent/cmd/status-service"
	"github.com/open-edge-platform/edge-node-agents/node-agent/info"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/auth"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/cluster"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/hostmgr_client"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/instrument"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Initialize logger
var log = logger.Logger
var initTimestamp = time.Now().Unix()

const REFRESH_CHECK_INTERVAL = 600 * time.Second
const TOKEN_REFRESH_CHECK_INTERVAL = 300 * time.Second
const COMPONENTS_INIT_WAIT_INTERVAL = 300 * time.Second
const CLUSTER_DETECTION_INTERVAL = 120 * time.Second

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}

	log.Infof("Starting %s - %s\n", info.Component, info.Version)
	ctx, cancel := context.WithCancelCause(context.Background())
	sigs := make(chan os.Signal, 1)
	defer close(sigs)
	errSigterm := errors.New("SIGTERM")
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %v; shutting down...", sig)
		if sig == syscall.SIGTERM {
			cancel(errSigterm)
		} else {
			cancel(errors.New(sig.String()))
		}
	}()

	// Initialize configuration
	configPath := flag.String("config", "", "Config file path")

	flag.Parse()
	confs, err := config.New(*configPath)

	if err != nil {
		log.Errorf("unable to initialize configuration. Node agent will terminate %v", err)
		flag.Usage()
		os.Exit(1)
	}

	// metrics -> initialize metrics collection if enabled
	if confs.Metrics.Enabled {
		shutdown, err := metrics.Init(ctx, confs.Metrics.Endpoint, confs.Metrics.Interval, info.Component, info.Version)
		if err != nil {
			log.Errorf("Initialization of metrics failed: %v", err)
		} else {
			log.Info("Metrics collection started")
			defer func() {
				err = shutdown(ctx)
				if err != nil && !errors.Is(err, context.Canceled) {
					log.Errorf("Shutting down metrics failed! Error: %v", err)
				}
			}()
		}
	}

	// Set log level as per configuration
	logLevel := confs.LogLevel

	switch logLevel {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	case "error":
		log.Logger.SetLevel(logrus.ErrorLevel)
	default:
		log.Logger.SetLevel(logrus.InfoLevel)
	}

	// StatusMap in statusService is read in heartbeat go-routine
	server, statusService := statusService.InitStatusService(confs)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	// Go-routine to manage JWT token lifecycle for NA and other Agents
	go func() {
		defer wg.Done()
		authCli, err := comms.GetAuthCli(confs.Auth.AccessTokenURL, confs.GUID, nil)
		if err != nil {
			log.Errorf("failed to create IDP client: %v", err)
			cancel(errors.New("failed to create IDP client. Will terminate"))
		}

		releaseAuthCli, err := comms.GetAuthCli(confs.Auth.RsTokenURL, confs.GUID, nil)
		if err != nil {
			log.Errorf("failed to create IDP client for release service: %v", err)
			cancel(errors.New("failed to create IDP client for release service. Will terminate"))
		}

		tokMgr := auth.NewTokenManager(confs.Auth)
		// Populate all clients with JWT if already provisioned
		tokMgr.PopulateTokenClients(confs.Auth)
		// Add release-service client
		tokMgr.TokenClients = append(tokMgr.TokenClients, auth.ClientAuthToken{ClientName: "release-service"})
		// Initialize token for all configured clients
		createRefreshTokens(ctx, tokMgr, releaseAuthCli, confs, authCli)
		ticker := time.NewTicker(TOKEN_REFRESH_CHECK_INTERVAL)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("terminating JWT lifecycle management")
				return
			case <-ticker.C:
				// Renew token for all configured clients
				createRefreshTokens(ctx, tokMgr, releaseAuthCli, confs, authCli)
			}
		}
	}()

	wg.Add(1)
	// Go-routine to poll network endpoints for status
	go func() {
		defer wg.Done()
		// Need not poll outbound endpoints if heartbeat is not enabled
		if !confs.Onboarding.Enabled {
			return
		}

		ticker := time.NewTicker((1 * time.Nanosecond))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("terminating outbound endpoints polling")
				return
			case <-ticker.C:
				// Poll outbound endpoints for status
				statusService.PollNetworkEndpoints(ctx, confs.Status.NetworkEndpoints)
			}
			ticker.Reset(confs.Status.NetworkStatusInterval)
		}
	}()

	wg.Add(1)
	// Go-routine to collect status from EN components
	go func() {
		defer wg.Done()
		// Need not check status if heartbeat not enabled
		if !confs.Onboarding.Enabled {
			return
		}
		// Create a new listener
		lis, err := createListener(confs)
		if err != nil {
			cancel(err)
			return
		}
		// Start the status server in a goroutine
		go func() {
			if err := server.Serve(lis); err != nil {
				log.Errorf("Failed to serve: %v", err)
				cancel(errors.New("failed to create status server. Will terminate"))
			}
		}()
		// Wait for context to be done
		<-ctx.Done()
		log.Infoln("Shutting down status server")
		server.GracefulStop()
	}()

	wg.Add(1)
	// Go-routine to send heartbeats to Host Manager
	go func() {
		defer wg.Done()
		// Need not send heartbeats if onboarding is not enabled
		if !confs.Onboarding.Enabled {
			return
		}

		tlsConfig, err := utils.GetAuthConfig(ctx, nil)
		if err != nil {
			log.Errorf("failed to create TLS config for Host manager client : %v", err)
			cancel(errors.New("cannot create TLS config. Will terminate"))
		}

		hostmgrCli, err := hostmgr_client.ConnectToHostMgr(ctx, confs.GUID, confs.Onboarding.ServiceURL, tlsConfig)
		if err != nil {
			log.Errorf("failed to create Host Manager client : %v", err)
			cancel(errors.New("cannot create Host manager client. Will terminate"))
		}

		ticker := time.NewTicker(1 * time.Nanosecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("terminating heartbeats")
				return
			case <-ticker.C:
				updateInstanceStatus(ctx, hostmgrCli, statusService, confs)
			}
			ticker.Reset(confs.Onboarding.HeartbeatInterval)
		}
	}()

	wg.Add(1)
	// Go-routine to detect clusters and manage kubeconfig
	go func() {
		defer wg.Done()
		// Do not detect clusters if onboarding is not enabled or cluster detection is disabled
		if !confs.Onboarding.Enabled || !confs.Cluster.DetectionEnabled {
			if !confs.Cluster.DetectionEnabled {
				log.Info("Cluster detection disabled in configuration")
			}
			return
		}

		tlsConfig, err := utils.GetAuthConfig(ctx, nil)
		if err != nil {
			log.Errorf("failed to create TLS config for cluster detection : %v", err)
			return
		}

		hostmgrCli, err := hostmgr_client.ConnectToHostMgr(ctx, confs.GUID, confs.Onboarding.ServiceURL, tlsConfig)
		if err != nil {
			log.Errorf("failed to create Host Manager client for cluster detection updates : %v", err)
			return
		}

		clusterDetector := cluster.NewClusterDetector(confs.GUID)
		kubeconfigMgr := cluster.NewKubeconfigManager(hostmgrCli, confs.GUID)

		ticker := time.NewTicker(1 * time.Nanosecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("terminating cluster detection")
				return
			case <-ticker.C:
				detectAndManageCluster(ctx, hostmgrCli, clusterDetector, kubeconfigMgr, confs)
				fmt.Println("Cluster detection tick - implement cluster detection logic here")
			}
			ticker.Reset(confs.Cluster.DetectionInterval)
		}
	}()

	// once booted (connected to orchestrator, report boot stats)
	instrument.ReportBootStats()

	wg.Wait()
	log.Infoln("Exiting")
	if err := context.Cause(ctx); errors.Is(err, errSigterm) {
		os.Exit(0)
	}
	os.Exit(1)
}

func createListener(confs *config.NodeAgentConfig) (net.Listener, error) {

	// Remove the socket file if it already exists
	if err := os.RemoveAll(confs.Status.Endpoint); err != nil {
		log.Error("error removing socket file - ", err)
		return nil, errors.New("error removing socket file. Will terminate")
	}

	lis, err := net.Listen("unix", confs.Status.Endpoint)
	if err != nil {
		log.Error("error creating listener for status server - ", err)
		return nil, errors.New("failed to create listener for status server. Will terminate")
	}

	return lis, nil
}

// getSystemUptime returns the system uptime in seconds by reading /proc/uptime
func getSystemUptime() (float64, error) {
	data, err := utils.ReadFileNoLinks("/proc/uptime")
	if err != nil {
		return 0, err
	}

	line := strings.TrimSpace(string(data))
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, errors.New("unable to parse uptime from /proc/uptime")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uptime, nil
}

func createRefreshTokens(ctx context.Context, tokMgr *auth.TokenManager, releaseAuthCli *comms.Client, confs *config.NodeAgentConfig, authCli *comms.Client) {
	for i, tokenClient := range tokMgr.TokenClients {
		var err error
		var token oauth2.Token
		// If token is already provisioned, manage lifecycle
		if len(tokenClient.AccessToken) != 0 {
			if auth.IsTokenRefreshRequired(tokenClient.Expiry) {
				// provision release service token
				if tokenClient.ClientName == "release-service" {
					token, err = releaseAuthCli.ProvisionToken(ctx, confs.Auth, tokenClient)
				} else {
					token, err = authCli.ProvisionToken(ctx, confs.Auth, tokenClient)
				}
				if err != nil {
					log.Errorf("failed to manage token: %v", err)
					continue
				}
				tokMgr.TokenClients[i].AccessToken = token.AccessToken
				tokMgr.TokenClients[i].Expiry = token.Expiry
				log.Infof("JWT token refreshed for client %s successfully", tokenClient.ClientName)
			}
		} else {
			// provision release service token
			if tokenClient.ClientName == "release-service" {
				token, err = releaseAuthCli.ProvisionToken(ctx, confs.Auth, tokenClient)
			} else {
				token, err = authCli.ProvisionToken(ctx, confs.Auth, tokenClient)
			}

			if err != nil {
				log.Errorf("Failed to manage token: %v", err)
				continue
			}
			tokMgr.TokenClients[i].AccessToken = token.AccessToken
			tokMgr.TokenClients[i].Expiry = token.Expiry
			log.Infof("JWT token freshly provisioned for client %s successfully", tokenClient.ClientName)
		}
	}
}

// updateInstanceStatus sends status report to orchestrator. Assumes that hostMgrCli is always initialized at this point.
func updateInstanceStatus(ctx context.Context, hostMgrCli *hostmgr_client.Client, statusService *statusService.StatusService, confs *config.NodeAgentConfig) {

	status := proto.InstanceStatus_INSTANCE_STATUS_ERROR
	humanReadableStatus, ok := statusService.GatherStatus(confs)

	// Check if system uptime is less than COMPONENTS_INIT_WAIT_INTERVAL
	systemUptime, uptimeErr := getSystemUptime()
	isSystemBootingUp := false
	if uptimeErr != nil {
		log.Warnf("Failed to get system uptime: %v", uptimeErr)
	} else {
		// Booting up can be time consuming especially on Ubuntu post-install as installer
		// carries out several updates/upgrades. Hence consider system to be booting up
		// if uptime is less than 3 times COMPONENTS_INIT_WAIT_INTERVAL
		isSystemBootingUp = systemUptime < 3*COMPONENTS_INIT_WAIT_INTERVAL.Seconds()
	}

	// If all components are healthy, send running status
	if ok {
		status = proto.InstanceStatus_INSTANCE_STATUS_RUNNING
	} else if time.Now().Unix() < (initTimestamp+int64(COMPONENTS_INIT_WAIT_INTERVAL)) && isSystemBootingUp {
		// Send initializing status if:
		// 1. The agent has started less than 5 minutes ago, AND
		// 2. The system has been up for less than 5 minutes
		status = proto.InstanceStatus_INSTANCE_STATUS_INITIALIZING
	}

	tokenFile := filepath.Join(confs.Auth.AccessTokenPath, "node-agent", config.AccessToken)
	err := hostMgrCli.UpdateInstanceStatus(utils.GetAuthContext(ctx, tokenFile), proto.InstanceState_INSTANCE_STATE_RUNNING, status, humanReadableStatus)
	if err != nil {
		log.Errorf("not able to update node status to running : %v", err)
	}
}

// detects clusters on the node and manages kubeconfig lifecycle
func detectAndManageCluster(ctx context.Context, hostMgrCli *hostmgr_client.Client, detector *cluster.ClusterDetector, kubeconfigMgr *cluster.KubeconfigManager, confs *config.NodeAgentConfig) {
	clusterInfo, err := detector.DetectCluster()
	if err != nil {
		// No cluster detected yet - clear any existing kubeconfig and update status
		log.Debugf("No cluster detected: %v", err)
		if kubeconfigMgr.HasKubeconfig() {
			log.Debug("No cluster detected, clearing kubeconfig")
			// Done in case cluster was removed after detection or kubeconfig was left orphaned due to some error
			// In both cases, kubeconfig needs to be cleared to avoid stale kubeconfig scenario
			if clearErr := kubeconfigMgr.ClearKubeconfig(ctx); clearErr != nil {
				log.Errorf("Failed to clear kubeconfig: %v", clearErr)
			}
		}
		return
	}

	// Cluster detected - check if it's running and has kubeconfig
	if clusterInfo.Status != "running" {
		log.Debugf("Cluster detected but not running: %s", clusterInfo.Status)
		return
	}

	if clusterInfo.KubeconfigPath == "" {
		log.Warnf("Cluster running but no kubeconfig found for %s", clusterInfo.Type)
		return
	}

	// Retrieve kubeconfig content
	kubeconfigData, err := detector.GetKubeconfig(clusterInfo)
	if err != nil {
		log.Errorf("Failed to retrieve kubeconfig: %v", err)
		return
	}

	// Validate kubeconfig
	if err := detector.ValidateKubeconfig(kubeconfigData); err != nil {
		log.Errorf("Invalid kubeconfig detected: %v", err)
		return
	}

	// // Notify host manager about kubeconfig
	// if err := kubeconfigMgr.NotifyKubeconfig(ctx, kubeconfigData, clusterInfo); err != nil {
	// 	log.Errorf("Failed to notify host manager about kubeconfig: %v", err)
	// 	return
	// }

	tokenFile := filepath.Join(confs.Auth.AccessTokenPath, "node-agent", config.AccessToken)
	err = hostMgrCli.UpdateClusterStatus(utils.GetAuthContext(ctx, tokenFile), "kubeconfig")
	if err != nil {
		log.Errorf("not able to update node status to running : %v", err)
	}

	log.Debugf("Cluster management completed for %s cluster", clusterInfo.Type)
}
