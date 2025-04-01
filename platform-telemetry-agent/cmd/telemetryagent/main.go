// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"github.com/cenkalti/backoff/v4"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/clientapi"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/helper"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logcfg"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/metriccfg"
)

var log = logger.Logger

const AGENT_NAME = "platform-telemetry-agent"

const MIN_ARGS = 3
const MAX_ARGS = 100

func getNodeIDString() string {
	return viper.GetString("global.nodeid")
}

func getConString() string {

	serverHost := viper.GetString("server.address")
	serverPort := strconv.Itoa(viper.GetInt("server.port"))
	connectionString := fmt.Sprintf("%s:%s", serverHost, serverPort)
	log.Printf("Connecting to telemetrymgr: %s; \n", connectionString)
	return connectionString
}

func setViperConfig(cfgFilePath string) {

	viper.SetConfigFile(cfgFilePath)

	// Read the configuration file
	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("Error reading config file:%v\n", err)
	}
}

func main() {

	var configFilePath = flag.String("config", "", "the platform-telemetry-agent configuration file location")
	flag.Parse()

	if *configFilePath == "" {
		log.Errorf("-config must be provided")
		flag.Usage()
		os.Exit(1)
	}

	setViperConfig(*configFilePath)
	helper.AgentId = getNodeIDString()

	log.Printf("Starting Telemetry Agent on Node ID: %s\n", helper.AgentId)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println(sig)
		//cancel() //enable this will cause TA forever running even ctrl+c, only can using "kill" to kill the process
		os.Exit(0)
	}()

	telegrafFilePath := viper.GetString("ConfigPath.telegraf")
	fluentbitFilePath := viper.GetString("ConfigPath.fluentbit")

	telegrafHostGoldPath := viper.GetString("ConfigPath.configroot") + viper.GetString("ConfigPath.telegrafgoldhost")
	telegrafClusterGoldPath := viper.GetString("ConfigPath.configroot") + viper.GetString("ConfigPath.telegrafgoldcluster")
	fluentbitHostGoldPath := viper.GetString("ConfigPath.configroot") + viper.GetString("ConfigPath.fluentbitgoldhost")
	fluentbitClusterGoldPath := viper.GetString("ConfigPath.configroot") + viper.GetString("ConfigPath.fluentbitgoldcluster")
	metriccfg.TmpFileDir = viper.GetString("ConfigPath.tmpdir")
	logcfg.TmpFileDir = viper.GetString("ConfigPath.tmpdir")
	helper.Kubeconfig = viper.GetString("misc.kubeconfig")
	helper.Kubectl = viper.GetString("misc.kubectl")
	metriccfg.ConfigMapCommand = "sudo " + helper.Kubectl + " " + viper.GetString("misc.telegrafConfigMap")
	logcfg.ConfigMapCommand = "sudo " + helper.Kubectl + " " + viper.GetString("misc.fluentbitConfigMap")
	logcfg.FileOwner = viper.GetString("misc.fileOwner")

	refreshIntervalStr := viper.GetString("global.updateinterval")
	refreshInterval, err := strconv.Atoi(refreshIntervalStr)
	if err != nil {
		log.Printf("Error converting refreshInterval to integer, set to default 60 seconds: %v", err)
		refreshInterval = 60
	}

	address := getConString()
	tokenPath := viper.GetString("server.token")
	devMode := viper.GetBool("global.developerMode")

	cli, err := clientapi.ConnectToTelemetryManager(ctx, address, devMode)
	if cli == nil || err != nil {
		log.Fatalf("service client creation failed : %v", err)
	}

	var lastUpdateTimestamp int64 // atomically accessed to store last successful response timestamp

	var mainWg sync.WaitGroup
	mainWg.Add(1)
	go func() {
		defer mainWg.Done()
		for {
			//for i := 0; i < 1; i++ {
			time.Sleep(time.Duration(refreshInterval) * time.Second)

			resp, err := clientapi.GetConfig(ctx, cli.SouthboundClient, getNodeIDString(), tokenPath)
			if err != nil {
				log.Printf("Error of calling Telemetry Manager via GetConfig: %v", err)
				continue
			}

			atomic.StoreInt64(&lastUpdateTimestamp, time.Now().Unix())

			// to get isInit flag as well
			isDirtyMask, isInitMask := clientapi.CheckIfChanged(resp)
			if !isDirtyMask[clientapi.MetricHost] && !isDirtyMask[clientapi.MetricCluster] &&
				!isDirtyMask[clientapi.LogHost] && !isDirtyMask[clientapi.LogCluster] {
				log.Printf("No Changed Detected from Telemetry Manager on this cycle")
				continue
			} else {
				log.Printf("Changed Detected from Telemetry Manager on this cycle")
			}

			var wg sync.WaitGroup

			if isDirtyMask[clientapi.MetricHost] {
				wg.Add(1)
				go func() {
					defer wg.Done()
					// pass isInit flag to updates
					_, err := metriccfg.UpdateHostMetricConfig(ctx, resp, telegrafFilePath, telegrafHostGoldPath, isInitMask[clientapi.MetricHost])
					if err != nil {
						fmt.Println("Update Host Metric Config Error:", err)
					}
				}()
			}

			if isDirtyMask[clientapi.MetricCluster] {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := metriccfg.UpdateClusterMetricConfig(ctx, resp, telegrafClusterGoldPath, isInitMask[clientapi.MetricCluster])
					if err != nil {
						fmt.Println("Update Cluster Metric Config Error:", err)
					}
				}()
			}

			if isDirtyMask[clientapi.LogHost] {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := logcfg.UpdateHostLogConfig(ctx, resp, fluentbitFilePath, fluentbitHostGoldPath, isInitMask[clientapi.LogHost])
					if err != nil {
						fmt.Println("Update Host Log Config Error:", err)
					}
				}()
			}

			if isDirtyMask[clientapi.LogCluster] {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := logcfg.UpdateClusterLogConfig(ctx, resp, fluentbitClusterGoldPath, isInitMask[clientapi.LogCluster])
					if err != nil {
						fmt.Println("Update Cluster Log Config Error:", err)
					}
				}()
			}

			wg.Wait()

		}

	}()

	// Add the ticker functionality
	mainWg.Add(1)
	go func() {
		defer mainWg.Done()

		statusEndpoint := viper.GetString("global.statusEndpoint")
		statusClient, statusInterval := initStatusClientAndTicker(ctx, cancel, statusEndpoint)
		compareInterval := max(int64(statusInterval.Seconds()), int64(refreshInterval))
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
	// wait for program termination
	mainWg.Wait()

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
	err = backoff.Retry(op, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 30), ctx))
	if err != nil {
		log.Warnf("Defaulting to 10 seconds")
		interval = 10 * time.Second
	}

	return statusClient, interval
}
