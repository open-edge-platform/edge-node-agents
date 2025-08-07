// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/system"
	"github.com/sirupsen/logrus"
)

var log = logger.Logger

const AGENT_NAME = "hardware-discovery-agent"
const MAX_RETRIES = 3

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}

	log.Infof("Test Starting Hardware Discovery Agent.")
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %v; shutting down...", sig)
		cancel()
	}()

	// Initial the command arguments
	configPath := flag.String("config", "", "the hd-agent configuration file location")
	flag.Parse()

	if *configPath == "" {
		log.Errorf("-config must be provided")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := config.New(*configPath)
	if err != nil {
		log.Errorf("loading configuration failed : %v", err)
		os.Exit(1)
	}

	// metrics
	shutdown, err := metrics.Init(ctx, cfg.MetricsEndpoint, cfg.MetricsInterval, info.Component, info.Version)
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
	logLevel := cfg.LogLevel

	switch logLevel {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	case "error":
		log.Logger.SetLevel(logrus.ErrorLevel)
	default:
		log.Logger.SetLevel(logrus.InfoLevel)
	}

	tlsConfig, err := utils.GetAuthConfig(ctx, nil)
	if err != nil {
		log.Errorf("TLS configuration creation failed : %v", err)
		os.Exit(1)
	}

	guid, err := system.GetSystemUUID(exec.Command)
	if err != nil {
		log.Errorf("system uuid fetch failed : %v", err)
		os.Exit(1)
	}

	cli, err := comms.ConnectToEdgeInfrastructureManager(cfg.Onboarding.ServiceURL, tlsConfig)
	if cli == nil || err != nil {
		log.Errorf("onboarding service client creation failed : %v", err)
		os.Exit(1)
	}
	log.Info("Hardware Discovery Agent has successfully connected to orchestrator")

	var wg sync.WaitGroup
	update := make(chan string)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startmonitoringUdev(ctx, update); err != nil {
			if strings.Contains(err.Error(), "signal") {
				log.Infof("udev monitoring finished: %v", err)
				return
			}
			log.Errorf("udev monitoring failure : %v", err)
			os.Exit(1)
		}
	}()

	wg.Add(1)
	var lastUpdateTimestamp int64
	cyclicalTicker := time.NewTicker(cfg.UpdateInterval)
	go func() {
		defer wg.Done()
		op := func() error {
			return sendStatusUpdate(ctx, cli, guid, cfg.JWT.AccessTokenPath)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-update:
				cyclicalTicker.Stop()
				log.Infof("%s", msg)
				updateWithRetry(ctx, op, &lastUpdateTimestamp)
			case <-cyclicalTicker.C:
				cyclicalTicker.Stop()
				updateWithRetry(ctx, op, &lastUpdateTimestamp)
			}
			cyclicalTicker.Reset(cfg.UpdateInterval)
		}
	}()

	// Add the ticker functionality
	wg.Add(1)
	go func() {
		defer wg.Done()
		statusClient, statusInterval := initStatusClientAndTicker(ctx, cancel, cfg.StatusEndpoint)
		compareInterval := max(int64(statusInterval.Seconds()), int64(cfg.UpdateInterval.Seconds()))
		ticker := time.NewTicker(1 * time.Nanosecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticker.Stop()

				lastCheck := atomic.LoadInt64(&lastUpdateTimestamp)
				now := time.Now().Unix()
				// The agent consumes 15-20 seconds to populate the system information and does so with the
				// ticker stopped. To ensure that the agent is not marked as not ready due to this delay, a
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
			ticker.Reset(statusInterval)
		}
	}()

	update <- "Sending initial update"
	wg.Wait()
	log.Infof("Hardware Discovery Agent finished.")
}

func updateWithRetry(ctx context.Context, op func() error, lastUpdateTimestamp *int64) {
	err := backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), MAX_RETRIES))
	if err != nil {
		log.Errorf("Retry error : %v", err)
	} else {
		atomic.StoreInt64(lastUpdateTimestamp, time.Now().Unix())
	}
}

func startmonitoringUdev(ctx context.Context, update chan string) error {
	cmd := exec.CommandContext(ctx, "udevadm", "monitor", "--udev", "--subsystem-match=block", "--subsystem-match=net")

	// Starting to record output of udevadm monitor
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("failed creating StdoutPipe : %v", err)
		return err
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "devices") {
				fields := strings.Fields(scanner.Text())
				update <- fmt.Sprintf("hardware %s detected, name=%s", fields[2], fields[3])
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed starting command: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("finished waiting for command: %w", err)
	}

	return nil
}

func sendStatusUpdate(ctx context.Context, cli *comms.Client, guid string, tokenFile string) error {
	updateDeviceRequest := comms.GenerateSystemInfoRequest(exec.Command)
	_, err := cli.UpdateHostSystemInfoByGUID(utils.GetAuthContext(ctx, tokenFile), guid, updateDeviceRequest)
	if err != nil {
		log.Errorf("update device failure : %v", err)
	}
	return err
}

func initStatusClientAndTicker(ctx context.Context, cancel context.CancelFunc, statusServer string) (*status.StatusClient, time.Duration) {
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
