// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/collector"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/sender"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "agent",
		Short: "Reporting Service Agent",
		Run:   runAgent,
	}
	rootCmd.Flags().StringP("config", "c", "", "path to config file")
	rootCmd.Flags().BoolP("short", "s", false, "collect only identity, uptime and kubernetes data")
	_ = rootCmd.Execute() //nolint:errcheck // Ignoring error as it will be handled in the command execution
}

func runAgent(cmd *cobra.Command, _ []string) {
	start := time.Now()

	log := createLogger()
	// flushes buffer, if any
	defer log.Sync() //nolint:errcheck // Ignoring error as it doesn't make sense to handle it during shutdown

	configLoader := config.NewConfigLoader(log)
	cfg := configLoader.Load(cmd)

	shortMode, _ := cmd.Flags().GetBool("short") //nolint:errcheck // Ignoring error, if something goes wrong, full data will be collected anyway
	coll := collector.NewCollector(log)
	var dataCollected model.Root
	if shortMode {
		log.Info("Test Agent started in short mode.")
		dataCollected = coll.CollectDataShort(cfg)
	} else {
		log.Info("Agent started in full mode.")
		dataCollected = coll.CollectData(cfg)
	}

	log.Infow("Agent finished collecting data.", "duration", time.Since(start).String())

	// Send to backend
	sndr := sender.NewBackendSender("/etc/edge-node/metrics/endpoint", "/etc/edge-node/metrics/token")
	if err := sndr.Send(cfg, &dataCollected); err != nil {
		log.Errorf("Failed to send data to backend: %v", err)
	} else {
		log.Info("Data successfully sent to backend.")
	}
}

// createLogger initializes a new logger with a lumberjack writer for log rotation.
func createLogger() *zap.SugaredLogger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.StacktraceKey = ""                      // disable stacktrace key
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // time in ISO8601 format (e.g. "2006-01-02T15:04:05.000Z0700")

	logWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/var/log/edge-node/reporting.log",
		MaxAge:     90, // days
		MaxBackups: 5,
		Compress:   false,
	})

	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), logWriter, zapcore.InfoLevel)
	logger := zap.New(core)

	return logger.Sugar()
}
