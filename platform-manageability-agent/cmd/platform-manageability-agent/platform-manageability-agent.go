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
	"github.com/open-edge-platform/edge-node-agents/platform-manageability-agent/internal/logger"
	"github.com/sirupsen/logrus"
)

var log = logger.Logger

const AGENT_NAME = "platform-manageability-agent"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}
	log.Infof("Starting Platform Manageability Agent. Args: %v\n", os.Args[1:])

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %v; shutting down...", sig)
		cancel()
	}()

	// Initialize configuration
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()

	confs, err := config.New(*configPath)
	if err != nil {
		log.Errorf("unable to initialize configuration. Platform Manageability Agent will terminate %v", err)
		flag.Usage()
		os.Exit(1)
	}

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

	// Set log level as per configuration
	logLevel := confs.LogLevel

	switch logLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	log.Info("Platform Manageability Agent started successfully")

	// Main agent loop
	for {
		select {
		case <-ctx.Done():
			log.Info("Platform Manageability Agent shutting down")
			return
		case <-time.After(30 * time.Second):
			log.Debug("Platform Manageability Agent heartbeat")
		}
	}
}
