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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	auth "github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

const (
	AGENT_NAME  = "platform-manageability-agent"
	MAX_RETRIES = 3
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}

	// Initialize configuration
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()

	if configPath == nil || *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required and must not be empty\n")
		flag.Usage()
		os.Exit(1)
	}

	confs, err := config.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to initialize configuration. Platform Manageability Agent will terminate %v\n", err)
		os.Exit(1)
	}

	// Create logger locally
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level as per configuration
	// Supported log levels: "debug", "info", "error"
	logLevel := confs.LogLevel
	switch logLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	default:
		log.Warnf("Unknown log level '%s', defaulting to 'info'. Supported values: debug, info, error", logLevel)
		log.SetLevel(logrus.InfoLevel)
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

	// Run the agent with dependency injection
	if err := runAgentWithDependencies(ctx, log, confs); err != nil {
		fmt.Fprintf(os.Stderr, "Agent failed: %v\n", err)
		os.Exit(1)
	}
}

// runAgentWithDependencies initializes the agent with proper dependency injection
func runAgentWithDependencies(ctx context.Context, log *logrus.Logger, confs *config.Config) error {
	log.Info("Starting Platform Manageability Agent")
	log.Debugf("Platform Manageability Agent arguments: %v", os.Args[1:])

	// Run the agent with injected dependencies
	return runAgent(ctx, log, confs)
}

// runAgent runs the main agent logic with injected dependencies
func runAgent(ctx context.Context, log *logrus.Logger, confs *config.Config) error {
	// metrics
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

	tlsConfig, err := auth.GetAuthConfig(auth.GetAuthContext(ctx, confs.Auth.AccessTokenPath), nil)
	if err != nil {
		log.Fatalf("TLS configuration creation failed! Error: %v", err)
	}

	dmMgrClient := comms.ConnectToDMManager(auth.GetAuthContext(ctx, confs.Auth.AccessTokenPath), confs.Manageability.ServiceURL, tlsConfig)

	var (
		isAMTEnabled                = false
		amtStatusCheckInterval      = 30 * time.Second // TODO: Make this configurable.
		lastAMTStatusCheckTimestamp int64
	)

	amtStatusTicker := time.NewTicker(amtStatusCheckInterval)
	go func() {
		op := func() error {
			status, err := dmMgrClient.ReportAMTStatus(ctx)
			if err != nil || status == pb.AMTStatus_DISABLED {
				log.Errorf("Failed to report AMT status: %v", err)
				isAMTEnabled = false
				return err
			}
			log.Info("Successfully reported AMT status")
			isAMTEnabled = true
			return nil
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-amtStatusTicker.C:
				amtStatusTicker.Stop()
				updateWithRetry(ctx, log, op, &lastAMTStatusCheckTimestamp)
			}
			amtStatusTicker.Reset(amtStatusCheckInterval)
		}
	}()

	var (
		lastActivationCheckTimestamp int64
		activationCheckInterval      = 30 * time.Second // TODO: Make this configurable.
	)

	activationTicker := time.NewTicker(activationCheckInterval)
	go func() {
		op := func() error {
			if !isAMTEnabled {
				log.Info("Skipping activation check because AMT is not enabled")
				return nil
			}
			uuid, err := utils.GetSystemUUID()
			if err != nil {
				log.Errorf("Failed to retrieve UUID: %v", err)
				return err
			}
			err = dmMgrClient.RetrieveActivationDetails(ctx, uuid, confs)
			if err != nil {
				log.Errorf("Failed to retrieve activation details: %v", err)
				return err
			}
			log.Info("Successfully retrieved activation details")
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

	for {
		select {
		case <-ctx.Done():
			log.Info("Platform Manageability Agent shutting down")
			return nil
		case <-ticker.C:
			log.Debug("Platform Manageability Agent heartbeat")
			// TODO: Add main agent functionality here (e.g., health checks, work scheduling, etc.)
		}
	}
}

func updateWithRetry(ctx context.Context, log *logrus.Logger, op func() error, lastUpdateTimestamp *int64) {
	err := backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), MAX_RETRIES))
	if err != nil {
		log.Errorf("Retry error: %v", err)
	} else {
		atomic.StoreInt64(lastUpdateTimestamp, time.Now().Unix())
	}
}
