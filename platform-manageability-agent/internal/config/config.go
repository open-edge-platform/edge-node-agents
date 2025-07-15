// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package config contains Platform Manageability Agent configuration management
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

const HEARTBEAT_DEFAULT = 10

type ConfigManageability struct {
	Enabled           bool          `yaml:"enabled"`
	ServiceURL        string        `yaml:"serviceURL"`
	HeartbeatInterval time.Duration `yaml:"heartbeatInterval"`
}

type ConfigAuth struct {
	AccessTokenURL  string   `yaml:"accessTokenURL"`
	RsTokenURL      string   `yaml:"rsTokenURL"`
	AccessTokenPath string   `yaml:"accessTokenPath"`
	ClientCredsPath string   `yaml:"clientCredsPath"`
	TokenClients    []string `yaml:"tokenClients"`
}

type NetworkEndpoint struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type ConfigStatus struct {
	Endpoint         string            `yaml:"endpoint"`
	ServiceClients   []string          `yaml:"serviceClients"`
	OutboundClients  []string          `yaml:"outboundClients"`
	NetworkEndpoints []NetworkEndpoint `yaml:"networkEndpoints"`
}

type Config struct {
	Version         string              `yaml:"version"`
	LogLevel        string              `yaml:"logLevel"`
	GUID            string              `yaml:"GUID"`
	Manageability   ConfigManageability `yaml:"manageability"`
	Status          ConfigStatus        `yaml:"status"`
	MetricsEndpoint string              `yaml:"metricsEndpoint"`
	MetricsInterval time.Duration       `yaml:"metricsInterval"`
	Auth            ConfigAuth          `yaml:"auth"`
}

func New(configPath string, log *logrus.Entry) (*Config, error) {
	// Set default config path if not provided
	if configPath == "" {
		configPath = "/etc/edge-node/platform-manageability/confs/platform-manageability-agent.yaml"
	}

	log.Infof("Loading configuration from: %s", configPath)

	// Read configuration file
	configBytes, err := os.ReadFile(configPath)
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

	log.Infof("Configuration loaded successfully")
	return &config, nil
}
