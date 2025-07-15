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
	"syscall"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/info"
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/config"
	"github.com/sirupsen/logrus"
)

const AGENT_NAME = "platform-manageability-agent"

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
