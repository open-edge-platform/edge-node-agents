// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/logger"
)

func main() {
	log := logger.Get()
	// flushes buffer, if any
	defer log.Sync() //nolint:errcheck // Ignoring error as it doesn't make sense to handle it during shutdown

	var shortMode bool
	var rootCmd = &cobra.Command{
		Use:   "agent",
		Short: "Reporting Service Agent",
		Run: func(cmd *cobra.Command, _ []string) {
			cfg := config.LoadConfig(cmd)
			shortMode, _ = cmd.Flags().GetBool("short") //nolint:errcheck // Ignoring error, if something goes wrong, full data will be collected anyway
			collector := internal.NewCollector()
			var dataCollected model.Root
			start := time.Now()
			if shortMode {
				log.Info("Agent started in short mode.")
				dataCollected = collector.CollectDataShort(cfg)
			} else {
				log.Info("Agent started in full mode.")
				dataCollected = collector.CollectData(cfg)
			}

			log.Infow("Agent finished collecting data", "duration", time.Since(start).String())

			jsonDataCollected, err := json.MarshalIndent(dataCollected, "", "  ")
			if err != nil {
				log.Errorf("Error occurred while marshalling data: %v", err)
				return
			}

			// TODO: for now, to track progress, will be removed later
			println("Collected data:\n" + string(jsonDataCollected))
		},
	}
	rootCmd.Flags().StringP("config", "c", "", "path to config file")
	rootCmd.Flags().BoolP("short", "s", false, "collect only identity, uptime and kubernetes data")
	_ = rootCmd.Execute() //nolint:errcheck // Ignoring error as it will be handled in the command execution
}
