// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// main package implements functionality of the Cluster Agent
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
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/info"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/k8sbootstrap"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/state"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"

	"github.com/sirupsen/logrus"

	proto "github.com/open-edge-platform/cluster-api-provider-intel/pkg/api/proto"
)

var log = logger.Logger

const AGENT_NAME = "cluster-agent"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("%v v%v\n", info.Component, info.Version)
		os.Exit(0)
	}
	log.Infof("Test Starting Cluster Agent. Args: %v\n", os.Args[1:])

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %v; shutting down...", sig)
		cancel()
	}()

	// configuration
	cfgPath := flag.String("config", "", "Path to cluster agent config")
	flag.Parse()

	cfg, err := config.New(*cfgPath)
	if err != nil {
		log.Errorf("Failed to create configuration! Error: %s", err)
		flag.Usage()
		os.Exit(1)
	}

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warnf("parse log level: %v", err)
	} else {
		log.Logger.SetLevel(logLevel)
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

	log.Infof("Loaded configuration: %+v", cfg)

	tlsConfig, err := utils.GetAuthConfig(ctx, nil)
	if err != nil {
		log.Fatalf("TLS configuration creation failed! Error: %v", err)
	}

	clusterOrch, err := comms.ConnectToClusterOrch(cfg.ServerAddr, tlsConfig)
	if err != nil {
		log.Errorf("Connecting to Cluster Orchestrator failed! Error: %v", err)
	}

	stateMachine := state.New(ctx, clusterOrch, cfg.GUID, cfg.JWT.AccessTokenPath, k8sbootstrap.Execute)
	var lastUpdateTimestamp int64 // atomically accessed to store last cluster orch successful response timestamp

	wg := &sync.WaitGroup{}
	wg.Add(1)
	actionRequests := make(chan proto.UpdateClusterStatusResponse_ActionRequest)
	var resp *proto.UpdateClusterStatusResponse
	go func() {
		defer wg.Done()
		op := func() error {
			res, updateErr := clusterOrch.UpdateClusterStatus(utils.GetAuthContext(ctx, cfg.JWT.AccessTokenPath), stateMachine.State(), cfg.GUID)
			if updateErr != nil {
				return updateErr
			}
			atomic.StoreInt64(&lastUpdateTimestamp, time.Now().Unix())
			resp = res
			return nil
		}

		cyclicalTicker := time.NewTicker(1 * time.Nanosecond)
		for {
			select {
			case <-ctx.Done():
				return

			case <-cyclicalTicker.C:
				cyclicalTicker.Stop()
				err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
				if err != nil {
					log.Errorf("Retry error: %v", err)
				} else {
					// Non-blocking send. Select used with unbufferd channel.
					select {
					// If receiver is reading from channel then this goroutine will write to it.
					case actionRequests <- resp.GetActionRequest():
					// Receiver not reading from channel, skipping.
					default:
					}
				}
			}
			cyclicalTicker.Reset(cfg.Heartbeat)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		statusClient, statusInterval := initStatusClientAndTicker(ctx, cancel, cfg.StatusEndpoint)
		compareInterval := max(int64(statusInterval.Seconds()), int64(cfg.Heartbeat.Seconds()))
		ticker := time.NewTicker(1 * time.Nanosecond)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticker.Stop()

				lastCheck := atomic.LoadInt64(&lastUpdateTimestamp)
				now := time.Now().Unix()

				// To ensure that the agent is not marked as not ready due to a functional delay, a
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

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case actionRequest := <-actionRequests:
				switch actionRequest {

				case proto.UpdateClusterStatusResponse_NONE:
					// do nothing

				case proto.UpdateClusterStatusResponse_REGISTER:
					err = stateMachine.Register()
					if err != nil {
						log.Error(err)
					}

				case proto.UpdateClusterStatusResponse_DEREGISTER:
					err = stateMachine.Deregister()
					if err != nil {
						log.Error(err)
					}

				default:
					log.Errorf("Unknown ActionRequest: %s", actionRequest.String())
				}
			}
		}
	}()

	wg.Wait()
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
