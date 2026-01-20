// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package config contains Node Agent configuration management
package config

import (
	"fmt"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
	yaml "gopkg.in/yaml.v3"
)

const HostCert = "host-cert.pem"
const HostKey = "host-key.pem"

const NodeAgentCert = "node-agent.pem"
const NodeAgentKey = "node-agent-key.pem"

const AccessToken = "access_token"

const HEARTBEAT_DEFAULT = 10

var log = logger.Logger

type ConfigOnboarding struct {
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
	Endpoint              string            `yaml:"endpoint"`
	ServiceClients        []string          `yaml:"serviceClients"`
	OutboundClients       []string          `yaml:"outboundClients"`
	NetworkStatusInterval time.Duration     `yaml:"networkStatusInterval"`
	NetworkEndpoints      []NetworkEndpoint `yaml:"networkEndpoints"`
}

type ConfigMetrics struct {
	Enabled  bool          `yaml:"enabled"`
	Endpoint string        `yaml:"endpoint"`
	Interval time.Duration `yaml:"interval"`
}

type NodeAgentConfig struct {
	Version    string           `yaml:"version"`
	LogLevel   string           `yaml:"logLevel"`
	GUID       string           `yaml:"GUID"`
	Onboarding ConfigOnboarding `yaml:"onboarding"`
	Auth       ConfigAuth       `yaml:"auth"`
	Status     ConfigStatus     `yaml:"status"`
	Metrics    ConfigMetrics    `yaml:"metrics"`
}

// Create a new Node agent configuration.
func New(cfgPath string) (*NodeAgentConfig, error) {
	// Requires config path to be configured
	if cfgPath == "" {
		log.Warnln("Configuration file not provided!")
		return nil, fmt.Errorf("config validation error: config file required")
	}

	log.Infoln("Reading configuration from config file at ", cfgPath)
	cfg, err := readConfigFile(cfgPath)
	if err != nil {
		log.Errorln("Error reading configuration file: ", err)
		return nil, err
	}

	cfg.setDefaults(cfgPath)
	err = cfg.validate()
	if err != nil {
		log.Errorln("Error validating configuration file data: ", err)
		return nil, err
	}
	log.Debugf("Loaded configuration: %+v", cfg)
	return cfg, nil
}

// validate checks if config values are as expected
func (cfg *NodeAgentConfig) validate() error {
	if cfg.Onboarding.Enabled && cfg.Onboarding.ServiceURL == "" {
		return fmt.Errorf("config validation err: onboarding.serviceURL is required")
	}

	if cfg.Auth.AccessTokenURL == "" {
		return fmt.Errorf("config validation err: auth.AccessTokenURL is required")
	}
	if cfg.Auth.RsTokenURL == "" {
		return fmt.Errorf("config validation err: auth.RsTokenURL is required")
	}
	if cfg.Auth.AccessTokenPath == "" {
		return fmt.Errorf("config validation err: auth.AccessTokenPath is required")
	}

	if cfg.GUID == "" {
		return fmt.Errorf("config validation err: GUID is required")
	}

	log.Infoln("configurations parsed successfully")
	return nil
}

func (cfg *NodeAgentConfig) setDefaults(cfgPath string) {

	if cfg.Onboarding.HeartbeatInterval <= 0*time.Second {
		log.Warnf("heartbeat not provided by %s, setting to default %d", cfgPath, HEARTBEAT_DEFAULT)
		cfg.Onboarding.HeartbeatInterval = HEARTBEAT_DEFAULT * time.Second
	}

	// interval to poll outbound endpoints is kept 6x of heartbeat on purpose (defaulting to 60 seconds
	// and capped to 60 seconds) because network calls can be costly to the service provider
	interval := cfg.Onboarding.HeartbeatInterval * 6
	if interval > 60*time.Second {
		interval = 60 * time.Second
	}
	cfg.Status.NetworkStatusInterval = interval
}

// readConfigFile loads yaml file into NodeAgentConfig type
func readConfigFile(path string) (*NodeAgentConfig, error) {
	// Read the YAML config file
	yamlFile, err := utils.ReadFileNoLinks(path)
	if err != nil {
		return nil, err
	}

	// Parse the YAML config file into a Config struct
	var config NodeAgentConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
