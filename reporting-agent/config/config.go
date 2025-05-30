// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/logger"
)

type Config struct {
	K8s K8sConfig `mapstructure:"k8s"`
}

type K8sConfig struct {
	K3sKubectlPath     string `mapstructure:"k3sKubectlPath"`
	K3sKubeConfigPath  string `mapstructure:"k3sKubeConfigPath"`
	Rke2KubectlPath    string `mapstructure:"rke2KubectlPath"`
	Rke2KubeConfigPath string `mapstructure:"rke2KubeConfigPath"`
}

// LoadConfig loads config using viper or returns defaults.
func LoadConfig(cmd *cobra.Command) Config {
	defCfg := setDefaults()
	configPath, _ := cmd.Flags().GetString("config") //nolint:errcheck // Ignoring error, potential empty string will be handled below

	v := viper.New()
	v.SetDefault("k8s.k3sKubectlPath", defCfg.K8s.K3sKubectlPath)
	v.SetDefault("k8s.k3sKubeConfigPath", defCfg.K8s.K3sKubeConfigPath)
	v.SetDefault("k8s.rke2KubectlPath", defCfg.K8s.Rke2KubectlPath)
	v.SetDefault("k8s.rke2KubeConfigPath", defCfg.K8s.Rke2KubeConfigPath)

	log := logger.Get()
	if configPath == "" {
		log.Infow("No config file provided, using default configuration", "config", defCfg)
		return defCfg
	}

	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		log.Warnw(fmt.Sprintf("Config file %q not found or unreadable, using default configuration", configPath), "err", err, "config", defCfg)
		return defCfg
	}
	log.Infof("Loaded configuration from %s", configPath)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Warnw("Failed to unmarshal config, using default configuration", "config", defCfg)
		return defCfg
	}

	log.Infow("Final configuration used", "config", cfg)
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
	}
}
