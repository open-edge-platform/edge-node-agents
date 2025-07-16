// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package config contains Platform Manageability Agent configuration management
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

const HEARTBEAT_DEFAULT = 10

type ConfigManageability struct {
	Enabled           bool          `yaml:"enabled"`
	ServiceURL        string        `yaml:"serviceURL"`
	HeartbeatInterval time.Duration `yaml:"heartbeatInterval"`
}

type Config struct {
	Version         string              `yaml:"version"`
	LogLevel        string              `yaml:"logLevel"`
	GUID            string              `yaml:"GUID"`
	Manageability   ConfigManageability `yaml:"manageability"`
	StatusEndpoint  string              `yaml:"statusEndpoint"`
	MetricsEndpoint string              `yaml:"metricsEndpoint"`
	MetricsInterval time.Duration       `yaml:"metricsInterval"`
	AccessTokenPath string              `yaml:"accessTokenPath"`
}

func New(configPath string, log *logrus.Entry) (*Config, error) {
	log.Infof("Loading configuration from: %s", configPath)

	// Read configuration file
	configBytes, err := utils.ReadFileNoLinks(configPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML configuration
	var config Config
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values if not specified
	if config.Manageability.HeartbeatInterval == 0 {
		config.Manageability.HeartbeatInterval = HEARTBEAT_DEFAULT * time.Second
	}

	if config.MetricsInterval == 0 {
		config.MetricsInterval = HEARTBEAT_DEFAULT * time.Second
	}

	if config.Manageability.ServiceURL == "" {
		return nil, fmt.Errorf("URL for Device Manageability Resource Manager not provided by config file")
	}

	if config.AccessTokenPath == "" {
		return nil, fmt.Errorf("JWT not provided by config file")
	}

	if config.StatusEndpoint == "" || !strings.HasPrefix(config.StatusEndpoint, "unix://") {
		return nil, fmt.Errorf("Agent status reporting address not provided by config file")
	}

	if config.GUID == "" {
		return nil, fmt.Errorf("Edge Node GUID not provided by config file")
	}

	log.Infof("Configuration loaded successfully")
	return &config, nil
}
