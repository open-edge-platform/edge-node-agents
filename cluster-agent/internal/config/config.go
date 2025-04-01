// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"time"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"gopkg.in/yaml.v3"
)

var log = logger.Logger

type JWT struct {
	AccessTokenPath string `yaml:"accessTokenPath"`
}

type Config struct {
	Version         string        `yaml:"version"`
	GUID            string        `yaml:"GUID"`
	LogLevel        string        `yaml:"logLevel"`
	ServerAddr      string        `yaml:"clusterOrchestratorURL"`
	Heartbeat       time.Duration `yaml:"heartbeat"`
	MetricsEndpoint string        `yaml:"metricsEndpoint"`
	MetricsInterval time.Duration `yaml:"metricsInterval"`
	StatusEndpoint  string        `yaml:"statusEndpoint"`
	JWT             JWT           `yaml:"jwt"`
}

func New(cfgPath string) (*Config, error) {
	log.WithField("config path", cfgPath)

	content, err := utils.ReadFileNoLinks(cfgPath)
	if err != nil {
		log.WithField("err", err).Error("Loading config failed")
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.WithField("err", err).Error("Unmarshaling failed")
		return nil, err
	}

	config.setDefaults(cfgPath)

	err = config.validate()
	if err != nil {
		log.WithField("err", err).Error("Config validation failed")
		return nil, err
	}

	return &config, nil
}

func (cfg *Config) setDefaults(cfgPath string) {
	if cfg.LogLevel == "" {
		log.Warnf("log level not provided by %s, setting to default value", cfgPath)
		cfg.LogLevel = "info"
	}

	if cfg.Heartbeat <= 0*time.Second {
		log.Warnf("heartbeat not provided by %s, setting to default value", cfgPath)
		cfg.Heartbeat = 10 * time.Second
	}
}

func (cfg *Config) validate() error {
	if cfg.GUID == "" {
		return fmt.Errorf("GUID is required")
	}

	if cfg.ServerAddr == "" {
		return fmt.Errorf("clusterOrchestratorURL is required")
	}

	if cfg.JWT.AccessTokenPath == "" {
		return fmt.Errorf("JWT.accessTokenPath is required")
	}

	return nil
}
