// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/logger"
	"gopkg.in/yaml.v3"
)

var log = logger.Logger

type Onboarding struct {
	ServiceURL string `yaml:"serviceURL"`
}

type JWT struct {
	AccessTokenPath string `yaml:"accessTokenPath"`
}

type Config struct {
	Version         string        `json:"version" yaml:"version"`
	LogLevel        string        `json:"logLevel" yaml:"logLevel"`
	Onboarding      Onboarding    `json:"onboarding" yaml:"onboarding"`
	MetricsEndpoint string        `yaml:"metricsEndpoint"`
	MetricsInterval time.Duration `yaml:"metricsInterval"`
	StatusEndpoint  string        `yaml:"statusEndpoint"`
	UpdateInterval  time.Duration `yaml:"interval"`
	JWT             JWT           `yaml:"jwt"`
}

func New(configFile string) (*Config, error) {
	var config Config

	content, err := utils.ReadFileNoLinks(configFile)
	if err != nil {
		log.Errorf("loading config from %v failed: %v", configFile, err)
		return nil, err
	}

	if !strings.Contains(string(content), "onboarding") {
		log.Errorf("incorrect config file format provided! : %v", err)
		return nil, errors.New("incorrect config file format provided")
	}

	log.Infof("Using configuration file: %s", configFile)

	if err := yaml.Unmarshal(content, &config); err != nil {
		log.Errorf("unmarshaling config from %s failed: %v", configFile, err)
		return nil, err
	}

	if config.Version == "" {
		log.Warnf("version not provided by %s, setting to default value", configFile)
		config.Version = "v0.2.0"
	}
	if config.LogLevel == "" {
		log.Warnf("logLevel not provided by %s, setting to default value", configFile)
		config.LogLevel = "info"
	}
	if config.UpdateInterval <= 0 {
		log.Warnf("interval not provided by %s, setting to default value", configFile)
		config.UpdateInterval = 30 * time.Second
	}

	if config.LogLevel != "info" && config.LogLevel != "debug" && config.LogLevel != "warning" && config.LogLevel != "error" {
		log.Errorf("unsupported logLevel value provided by %s, exiting : %v", configFile, err)
		return nil, errors.New("unsupported logLevel value provided by config file")
	}
	if config.Onboarding.ServiceURL == "" {
		log.Errorf("URL for Edge Infrastructure Manager not provided by %s, exiting : %v", configFile, err)
		return nil, errors.New("URL for Edge Infrastructure Manager not provided by config file")
	}
	if config.JWT.AccessTokenPath == "" {
		log.Errorf("JWT not provided by %s, exiting : %v", configFile, err)
		return nil, errors.New("JWT not provided by config file")
	}

	log.Debugf("Loaded configuration: %v", config)
	return &config, nil
}
