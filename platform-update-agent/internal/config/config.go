// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"

	yaml "gopkg.in/yaml.v3"
)

var log = logger.Logger()

type JWT struct {
	AccessTokenPath string `yaml:"accessTokenPath"`
}

type Service struct {
	ServiceUrl string `yaml:"serviceURL"`
}

type Config struct {
	Version              string        `yaml:"version"`
	GUID                 string        `yaml:"GUID"`
	LogLevel             string        `yaml:"logLevel"`
	UpdateServiceURL     string        `yaml:"updateServiceURL"`
	TickerInterval       time.Duration `yaml:"tickerInterval"`
	MetadataPath         string        `yaml:"metadataPath"`
	INBCLogsPath         string        `yaml:"INBCLogsPath"`
	INBCGranularLogsPath string        `yaml:"INBCGranularLogsPath"`
	MetricsEndpoint      string        `yaml:"metricsEndpoint"`
	MetricsInterval      time.Duration `yaml:"metricsInterval"`
	JWT                  JWT           `yaml:"jwt"`
	ReleaseServiceFQDN   string        `yaml:"releaseServiceFQDN"`

	// When pre-downloading updates, if we are less than this time away from the
	// scheduled update start, start downloading immediately. Generally this should
	// be enough time to complete the download comfortably so the update can actually run.
	ImmediateDownloadWindow time.Duration `yaml:"immediateDownloadWindow"`

	// Length of time before the immediate download window in which we normally
	// randomize download start time. When not in immediate download window, we
	// pick a random sample inside whatever part of this download window is remaining
	// to start the download.
	DownloadWindow time.Duration `yaml:"downloadWindow"`

	// The endpoint to send the health status (Ready/NotReady) of the agent
	StatusEndpoint string `yaml:"statusEndpoint"`
}

func New(cfgPath string) (*Config, error) {
	log.Infoln("Config path", cfgPath)

	err := utils.IsSymlink(cfgPath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Errorf("Loading config failed: %v", err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Errorf("Unmarshaling failed: %v", err)
		return nil, err
	}

	// Set default values for new fields if they are not set in the config file
	config.setDefaults()

	err = config.validate()
	if err != nil {
		log.Errorf("Config validation failed: %v", err)
		return nil, err
	}

	log.Debugf("Loaded configuration: %+v", config)
	return &config, nil
}

func (cfg *Config) setDefaults() {
	if cfg.ImmediateDownloadWindow == 0 {
		cfg.ImmediateDownloadWindow = 10 * time.Minute
	}

	if cfg.DownloadWindow == 0 {
		cfg.DownloadWindow = 6 * time.Hour
	}
}

func (cfg *Config) validate() error {
	if cfg.UpdateServiceURL == "" {
		return fmt.Errorf("updateServiceURL is required")
	}

	if cfg.GUID == "" {
		return fmt.Errorf("GUID is required")
	}
	if cfg.JWT.AccessTokenPath == "" {
		return fmt.Errorf("JWT is required")
	}

	if cfg.ImmediateDownloadWindow < 0 {
		return fmt.Errorf("immediateDownloadWindow cannot be negative")
	}
	if cfg.DownloadWindow < 0 {
		return fmt.Errorf("downloadWindow cannot be negative")
	}

	return nil
}
