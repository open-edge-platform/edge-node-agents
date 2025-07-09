/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/afero"
)

// Configurations represents the structure of the XML configuration file
type Configurations struct {
	OSUpdater struct {
		TrustedRepositories    []string `json:"trustedRepositories"`
		ProceedWithoutRollback bool     `json:"proceedWithoutRollback"`
	} `json:"os_updater"`

	LUKS struct {
		VolumePath     string `json:"volumePath"`
		MapperName     string `json:"mapperName"`
		MountPoint     string `json:"mountPoint"`
		PasswordLength int    `json:"passwordLength"`
		Size           int    `json:"size"`
		UseTPM         bool   `json:"useTPM"`
		User           string `json:"user"`
		Group          string `json:"group"`
		Password       []byte `json:"password,omitempty"` // Optional field, may not be present in the config
	} `json:"luks"`
}

// LoadConfig loads the XML configuration file
func LoadConfig(fs afero.Fs, filePath string) (*Configurations, error) {
	content, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Unmarshal the JSON content into the Configurations struct
	var config Configurations
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	return &config, nil
}

// IsTrustedRepository checks if the given URL is in the list of trusted repositories
func IsTrustedRepository(url string, config *Configurations) bool {
	log.Printf("Checking if URL '%s' is in trusted repositories", url)
	for _, repo := range config.OSUpdater.TrustedRepositories {
		if strings.HasPrefix(url, repo) {
			return true
		}
	}
	return false
}

// IsProceedWithoutRollback checks if the rollback is allowed based on the configuration
func IsProceedWithoutRollback(config *Configurations) bool {
	log.Printf("Checking if rollback is allowed")
	return config.OSUpdater.ProceedWithoutRollback
}
