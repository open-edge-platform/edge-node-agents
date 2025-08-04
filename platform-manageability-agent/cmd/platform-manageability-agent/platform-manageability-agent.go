// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main package implements functionality of the Platform Manageability Agent
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	auth "github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

const (
	AGENT_NAME  = "platform-manageability-agent"
	MAX_RETRIES = 3

	// AMTStatus constants representing the state of AMT.
	AMTStatusDisabled int32 = 0
	AMTStatusEnabled  int32 = 1
)

var (
	log = logger.Logger

	isAMTEnabled                  int32
	isModuleAndServiceInitialized int32
)

func isAMTCurrentlyEnabled() bool {
	return atomic.LoadInt32(&isAMTEnabled) == AMTStatusEnabled
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}

	log.Infof("Starting Platform Manageability Agent")

	// Initialize configuration
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()

	if configPath == nil || *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required and must not be empty\n")
		flag.Usage()
		os.Exit(1)
	}

	confs, err := config.New(*configPath, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to initialize configuration. Platform Manageability Agent will terminate %v\n", err)
		os.Exit(1)
	}

	// Set the log level as per the configuration
	logLevel := confs.LogLevel

	switch logLevel {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	case "error":
		log.Logger.SetLevel(logrus.ErrorLevel)
	case "info":
		log.Logger.SetLevel(logrus.InfoLevel)
	default:
		log.Warnf("Unknown log level '%s', defaulting to 'info'. Supported values: debug, info, error", logLevel)
		log.Logger.SetLevel(logrus.InfoLevel)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling before starting the agent
	sigs := make(chan os.Signal, 1)
	defer close(sigs) // Close the signal channel
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start signal handler goroutine with proper cleanup
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %v; shutting down...", sig)
		cancel()
	}()

	// Enable agent metrics
	shutdown, err := metrics.Init(ctx, confs.MetricsEndpoint, confs.MetricsInterval, info.Component, info.Version)
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

	log.Info("Platform Manageability Agent started successfully")

	tlsConfig, err := auth.GetAuthConfig(ctx, nil)
	if err != nil {
		log.Fatalf("TLS configuration creation failed! Error: %v", err)
	}

	log.Infof("Connecting to Device Management Manager at %s", confs.Manageability.ServiceURL)
	dmMgrClient := comms.ConnectToDMManager(ctx, confs.Manageability.ServiceURL, tlsConfig)

	hostID, err := utils.GetSystemUUID()
	if err != nil {
		log.Fatalf("Failed to retrieve system UUID with an error: %v", err)
	}

	var wg sync.WaitGroup
	var lastCheckTimestamp int64
	wg.Add(1)
	go func() {
		defer wg.Done()

		op := func() error {
			log.Infof("Reporting AMT status for host %s", hostID)
			status, err := dmMgrClient.ReportAMTStatus(auth.GetAuthContext(ctx, confs.AccessTokenPath), hostID)
			if err != nil {
				log.Errorf("Failed to report AMT status for host %s: %v", hostID, err)
				// Deliberately induce unhealthy state by increasing elapsed time.
				// This is done to ensure the other goroutine does not report false positive.
				// Can be consolidated when the goroutines are merged.
				atomic.StoreInt64(&lastCheckTimestamp, time.Now().Unix()-3600)
				return fmt.Errorf("failed to report AMT status: %w", err)
			}

			switch status {
			case pb.AMTStatus_DISABLED:
				atomic.StoreInt32(&isAMTEnabled, AMTStatusDisabled)
			case pb.AMTStatus_ENABLED:
				atomic.StoreInt32(&isAMTEnabled, AMTStatusEnabled)
				if atomic.CompareAndSwapInt32(&isModuleAndServiceInitialized, 0, 1) {
					if err := loadModule(); err != nil {
						log.Errorf("Error while loading module: %v", err)
					} else {
						log.Info("Module mei_me loaded successfully")
					}
					for _, action := range []string{"unmask", "enable", "start"} {
						log.Infof("%sing %s...\n", action, "lms.service")
						if err := enableService(action); err != nil {
							log.Errorf("Error while enabling service: %v", err)
						}
					}
				}
			default:
				log.Warnf("Unknown AMT status: %v, treating as disabled", status)
				atomic.StoreInt32(&isAMTEnabled, AMTStatusDisabled)
			}
			return nil
		}
		err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		if err != nil {
			if ctx.Err() != nil {
				log.Info("AMT status reporting canceled due to context cancellation")
			} else {
				log.Errorf("Failed to report AMT status for host %s after retries: %v", hostID, err)
			}
			return
		}
		log.Infof("Successfully reported AMT status for host %s", hostID)
	}()

	var (
		activationCheckInterval      = confs.Manageability.HeartbeatInterval
		lastActivationCheckTimestamp int64
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		activationTicker := time.NewTicker(activationCheckInterval)
		defer activationTicker.Stop()

		op := func() error {
			if !isAMTCurrentlyEnabled() {
				log.Info("Skipping activation check because AMT is not enabled")
				atomic.StoreInt64(&lastCheckTimestamp, time.Now().Unix()) // To report Agent healthy if AMT disabled
				return nil
			}

			log.Infof("AMT is enabled, checking activation details for host %s", hostID)
			// FIXME: https://github.com/open-edge-platform/edge-node-agents/pull/170#discussion_r2236433075
			// The suggestion is to combine the activation check and retrieval of activation details into a single call
			// to reduce the number of RPC calls.
			err = dmMgrClient.RetrieveActivationDetails(auth.GetAuthContext(ctx, confs.AccessTokenPath), hostID, confs)
			if err != nil {
				if errors.Is(err, comms.ErrActivationSkipped) {
					log.Logger.Debugf("AMT activation skipped for host %s - reason: %v", hostID, err)
					return nil
				}
				return fmt.Errorf("failed to retrieve activation details: %w", err)
			}
			atomic.StoreInt64(&lastCheckTimestamp, time.Now().Unix()) // To report Agent healthy if AMT enabled
			log.Infof("Successfully retrieved activation details for host %s", hostID)
			return nil
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-activationTicker.C:
				activationTicker.Stop()
				updateWithRetry(ctx, log, op, &lastActivationCheckTimestamp)
			}
			activationTicker.Reset(activationCheckInterval)
		}
	}()

	// Main agent loop using context-aware ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Info("Platform Manageability Agent shutting down")
				return
			case <-ticker.C:
				log.Debug("Platform Manageability Agent heartbeat")
				// TODO: Add main agent functionality here (e.g., health checks, work scheduling, etc.)
			}
		}
	}()

	// Add agent status reporting
	wg.Add(1)
	go func() {
		defer wg.Done()
		statusClient, statusInterval := initStatusClientAndTicker(ctx, cancel, log, confs.StatusEndpoint)
		compareInterval := max(int64(statusInterval.Seconds()), int64(confs.Manageability.HeartbeatInterval.Seconds()))
		statusTicker := time.NewTicker(1 * time.Nanosecond)
		defer statusTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-statusTicker.C:
				statusTicker.Stop()

				lastCheck := atomic.LoadInt64(&lastCheckTimestamp)
				now := time.Now().Unix()
				// To ensure the agent is not marked as not ready due to a functional delay, a
				// check of 2x of interval is considered. This should prevent false negatives.
				if now-lastCheck <= 2*compareInterval {
					if err := statusClient.SendStatusReady(ctx, AGENT_NAME); err != nil {
						log.Errorf("Failed to send status ready: %v", err)
					}
					log.Infoln("Status Ready")
				} else {
					if err := statusClient.SendStatusNotReady(ctx, AGENT_NAME); err != nil {
						log.Errorf("Failed to send status not ready: %v", err)
					}
					log.Infoln("Status Not Ready")
				}
			}
			statusTicker.Reset(statusInterval)
		}
	}()

	wg.Wait()
	log.Infof("Platform Manageability Agent finished")
}

func enableService(action string) error {
	allowedActions := map[string]bool{"unmask": true, "enable": true, "start": true}

	if !allowedActions[action] {
		return fmt.Errorf("invalid service details")
	}

	// Check if the service is already running
	output, err := utils.ExecuteCommand("systemctl", []string{"is-active", "lms.service"})
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		log.Infof("Service %s is already running, skipping %s operation", "lms.service", action)
		return nil
	}

	output, err = utils.ExecuteCommand("sudo", []string{"systemctl", action, "lms.service"})
	if err != nil {
		return fmt.Errorf("failed to %s %s: %v", action, "lms.service", err)
	}
	log.Infof("Service %s %s successfully: %s", "lms.service", action, string(output))
	return nil
}

func loadModule() error {
	// Check if the module is already loaded using lsmod | grep
	output, err := utils.ExecuteCommand("sh", []string{"-c", "lsmod | grep mei_me"})
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		log.Info("Module mei_me is already loaded, skipping load operation")
		return nil
	}

	// Load the module if not already loaded
	output, err = utils.ExecuteCommand("sudo", []string{"modprobe", "mei_me"})
	if err != nil {
		return fmt.Errorf("failed to load module %s: %v", "mei_me", err)
	}
	log.Infof("Module %s loaded successfully: %s", "mei_me", string(output))
	return nil
}

func initStatusClientAndTicker(ctx context.Context, cancel context.CancelFunc, log *logrus.Entry, statusServer string) (*status.StatusClient, time.Duration) {
	statusClient, err := status.InitClient(statusServer)
	if err != nil {
		log.Errorf("Failed to initialize status client: %v", err)
		cancel()
	}

	var interval time.Duration
	op := func() error {
		interval, err = statusClient.GetStatusInterval(ctx, AGENT_NAME)
		if err != nil {
			log.Errorf("Failed to get status interval: %v", err)
		}
		return err
	}

	// High number of retries as retries would mostly indicate a problem with the status server
	err = backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), 30))
	if err != nil {
		log.Warnf("Defaulting to 10 seconds")
		interval = 10 * time.Second
	}

	return statusClient, interval
}

func updateWithRetry(ctx context.Context, log *logrus.Entry, op func() error, lastUpdateTimestamp *int64) {
	err := backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), MAX_RETRIES))
	if err != nil {
		log.Errorf("Retry error: %v", err)
	} else {
		atomic.StoreInt64(lastUpdateTimestamp, time.Now().Unix())
	}
}
