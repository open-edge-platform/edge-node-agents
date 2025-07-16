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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/sirupsen/logrus"
)

const AGENT_NAME = "platform-manageability-agent"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}

	var log = logger.Logger
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

	// Main agent loop using context-aware ticker
	var wg sync.WaitGroup
	var lastUpdateTimestamp int64
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
				atomic.StoreInt64(&lastUpdateTimestamp, time.Now().Unix())
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

				lastCheck := atomic.LoadInt64(&lastUpdateTimestamp)
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

	log.Infof("Platform Manageability Agent finished")
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
