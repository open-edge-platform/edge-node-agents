// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds the configuration for the application.
type Config struct {
	K8s     K8sConfig     `mapstructure:"k8s"`
	Backend BackendConfig `mapstructure:"backend"`
}

// K8sConfig holds Kubernetes-related configuration paths.
type K8sConfig struct {
	K3sKubectlPath     string `mapstructure:"k3sKubectlPath"`
	K3sKubeConfigPath  string `mapstructure:"k3sKubeConfigPath"`
	Rke2KubectlPath    string `mapstructure:"rke2KubectlPath"`
	Rke2KubeConfigPath string `mapstructure:"rke2KubeConfigPath"`
}

// BackendConfig holds backend and backoff configuration.
type BackendConfig struct {
	Backoff BackendBackoffConfig `mapstructure:"backoff"`
}

// BackendBackoffConfig holds backoff configuration for backend communication.
type BackendBackoffConfig struct {
	MaxTries uint `mapstructure:"maxTries"`
}

// Loader loads configuration and holds logger.
type Loader struct {
	log *zap.SugaredLogger
}

// NewConfigLoader creates a new Loader with the given logger.
func NewConfigLoader(log *zap.SugaredLogger) *Loader {
	return &Loader{log: log}
}

// Load loads config using viper or returns defaults.
func (cl *Loader) Load(cmd *cobra.Command) Config {
	defCfg := setDefaults()
	configPath, _ := cmd.Flags().GetString("config") //nolint:errcheck // Ignoring error, potential empty string will be handled below

	v := viper.New()
	v.SetDefault("k8s.k3sKubectlPath", defCfg.K8s.K3sKubectlPath)
	v.SetDefault("k8s.k3sKubeConfigPath", defCfg.K8s.K3sKubeConfigPath)
	v.SetDefault("k8s.rke2KubectlPath", defCfg.K8s.Rke2KubectlPath)
	v.SetDefault("k8s.rke2KubeConfigPath", defCfg.K8s.Rke2KubeConfigPath)
	v.SetDefault("backend.backoff.maxTries", defCfg.Backend.Backoff.MaxTries)

	if configPath == "" {
		cl.log.Infow("No config file provided, using default configuration", "config", defCfg)
		return defCfg
	}

	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		cl.log.Warnw(fmt.Sprintf("Config file %q not found or unreadable, using default configuration", configPath), "err", err, "config", defCfg)
		return defCfg
	}
	cl.log.Infof("Loaded configuration from %s", configPath)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		cl.log.Warnw("Failed to unmarshal config, using default configuration", "config", defCfg)
		return defCfg
	}

	cl.log.Infow("Final configuration used", "config", cfg)
	return cfg
}

func setDefaults() Config {
	return Config{
		K8s: K8sConfig{
			K3sKubectlPath:     "/var/lib/rancher/k3s/bin/k3s kubectl",
			K3sKubeConfigPath:  "/etc/rancher/k3s/k3s.yaml",
			Rke2KubectlPath:    "/var/lib/rancher/rke2/bin/kubectl",
			Rke2KubeConfigPath: "/etc/rancher/rke2/rke2.yaml",
		},
		Backend: BackendConfig{
			Backoff: BackendBackoffConfig{
				MaxTries: 20,
			},
		},
	}
}
