// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTempConfigFile creates a temporary config file with the given content
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.env")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	return configPath
}

func TestLoadFromFile_BasicLoading(t *testing.T) {
	configContent := `# Test config
OBM_SVC=config.example.com
OBS_SVC=stream.example.com
OBM_PORT=50051
KEYCLOAK_URL=keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
SERIAL=CONFIG_SERIAL
UUID=11111111-1111-1111-1111-111111111111
IP=192.168.1.50
CA_CERT=/path/to/cert
DEBUG=true
TIMEOUT=10m
AUTO_DETECT=false
DISABLE_INTERACTIVE=true
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &Config{}

	err := LoadFromFile(cfg, configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify all values are loaded from config
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ObmSvc", cfg.ObmSvc, "config.example.com"},
		{"ObsSvc", cfg.ObsSvc, "stream.example.com"},
		{"ObmPort", cfg.ObmPort, 50051},
		{"KeycloakURL", cfg.KeycloakURL, "keycloak.example.com"},
		{"MacAddr", cfg.MacAddr, "AA:BB:CC:DD:EE:FF"},
		{"SerialNumber", cfg.SerialNumber, "CONFIG_SERIAL"},
		{"UUID", cfg.UUID, "11111111-1111-1111-1111-111111111111"},
		{"IPAddress", cfg.IPAddress, "192.168.1.50"},
		{"CaCertPath", cfg.CaCertPath, "/path/to/cert"},
		{"Debug", cfg.Debug, true},
		{"Timeout", cfg.Timeout, 10 * time.Minute},
		{"AutoDetect", cfg.AutoDetect, false},
		{"DisableInteractiveMode", cfg.DisableInteractiveMode, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoadFromFile_WithComments(t *testing.T) {
	configContent := `# This is a comment
# Another comment
OBM_SVC=obm.example.com

# Comment in the middle
OBS_SVC=obs.example.com
OBM_PORT=50051

# End comment
`

	configPath := createTempConfigFile(t, configContent)
	cfg := &Config{}

	err := LoadFromFile(cfg, configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.ObmSvc != "obm.example.com" {
		t.Errorf("ObmSvc = %v, want obm.example.com", cfg.ObmSvc)
	}
	if cfg.ObsSvc != "obs.example.com" {
		t.Errorf("ObsSvc = %v, want obs.example.com", cfg.ObsSvc)
	}
	if cfg.ObmPort != 50051 {
		t.Errorf("ObmPort = %v, want 50051", cfg.ObmPort)
	}
}

func TestLoadFromFile_WithQuotes(t *testing.T) {
	configContent := `OBM_SVC="quoted.example.com"
OBS_SVC='single-quoted.example.com'
KEYCLOAK_URL=unquoted.example.com
`

	configPath := createTempConfigFile(t, configContent)
	cfg := &Config{}

	err := LoadFromFile(cfg, configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.ObmSvc != "quoted.example.com" {
		t.Errorf("ObmSvc = %v, want quoted.example.com", cfg.ObmSvc)
	}
	if cfg.ObsSvc != "single-quoted.example.com" {
		t.Errorf("ObsSvc = %v, want single-quoted.example.com", cfg.ObsSvc)
	}
	if cfg.KeycloakURL != "unquoted.example.com" {
		t.Errorf("KeycloakURL = %v, want unquoted.example.com", cfg.KeycloakURL)
	}
}

func TestLoadFromFile_EmptyLines(t *testing.T) {
	configContent := `
OBM_SVC=obm.example.com


OBS_SVC=obs.example.com

`

	configPath := createTempConfigFile(t, configContent)
	cfg := &Config{}

	err := LoadFromFile(cfg, configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.ObmSvc != "obm.example.com" {
		t.Errorf("ObmSvc = %v, want obm.example.com", cfg.ObmSvc)
	}
	if cfg.ObsSvc != "obs.example.com" {
		t.Errorf("ObsSvc = %v, want obs.example.com", cfg.ObsSvc)
	}
}

func TestLoadFromFile_InvalidFormat(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "missing equals sign",
			content: "OBM_SVC obm.example.com",
			wantErr: true,
		},
		{
			name:    "invalid port",
			content: "OBM_PORT=not_a_number",
			wantErr: true,
		},
		{
			name:    "invalid boolean",
			content: "DEBUG=maybe",
			wantErr: true,
		},
		{
			name:    "invalid timeout",
			content: "TIMEOUT=not_a_duration",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := createTempConfigFile(t, tt.content)
			cfg := &Config{}

			err := LoadFromFile(cfg, configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile_NonexistentFile(t *testing.T) {
	cfg := &Config{}
	err := LoadFromFile(cfg, "/nonexistent/path/to/config.env")
	if err == nil {
		t.Error("LoadFromFile should fail for nonexistent file")
	}
}

func TestApplyValue_AllFields(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		validate func(*Config) error
	}{
		{
			name:  "OBM_SVC",
			key:   "OBM_SVC",
			value: "obm.example.com",
			validate: func(cfg *Config) error {
				if cfg.ObmSvc != "obm.example.com" {
					t.Errorf("ObmSvc = %v, want obm.example.com", cfg.ObmSvc)
				}
				return nil
			},
		},
		{
			name:  "OBS_SVC",
			key:   "OBS_SVC",
			value: "obs.example.com",
			validate: func(cfg *Config) error {
				if cfg.ObsSvc != "obs.example.com" {
					t.Errorf("ObsSvc = %v, want obs.example.com", cfg.ObsSvc)
				}
				return nil
			},
		},
		{
			name:  "OBM_PORT",
			key:   "OBM_PORT",
			value: "50051",
			validate: func(cfg *Config) error {
				if cfg.ObmPort != 50051 {
					t.Errorf("ObmPort = %v, want 50051", cfg.ObmPort)
				}
				return nil
			},
		},
		{
			name:  "KEYCLOAK_URL",
			key:   "KEYCLOAK_URL",
			value: "keycloak.example.com",
			validate: func(cfg *Config) error {
				if cfg.KeycloakURL != "keycloak.example.com" {
					t.Errorf("KeycloakURL = %v, want keycloak.example.com", cfg.KeycloakURL)
				}
				return nil
			},
		},
		{
			name:  "MAC",
			key:   "MAC",
			value: "AA:BB:CC:DD:EE:FF",
			validate: func(cfg *Config) error {
				if cfg.MacAddr != "AA:BB:CC:DD:EE:FF" {
					t.Errorf("MacAddr = %v, want AA:BB:CC:DD:EE:FF", cfg.MacAddr)
				}
				return nil
			},
		},
		{
			name:  "DEBUG true",
			key:   "DEBUG",
			value: "true",
			validate: func(cfg *Config) error {
				if cfg.Debug != true {
					t.Errorf("Debug = %v, want true", cfg.Debug)
				}
				return nil
			},
		},
		{
			name:  "DEBUG false",
			key:   "DEBUG",
			value: "false",
			validate: func(cfg *Config) error {
				if cfg.Debug != false {
					t.Errorf("Debug = %v, want false", cfg.Debug)
				}
				return nil
			},
		},
		{
			name:  "TIMEOUT",
			key:   "TIMEOUT",
			value: "5m",
			validate: func(cfg *Config) error {
				if cfg.Timeout != 5*time.Minute {
					t.Errorf("Timeout = %v, want 5m", cfg.Timeout)
				}
				return nil
			},
		},
		{
			name:  "AUTO_DETECT",
			key:   "AUTO_DETECT",
			value: "true",
			validate: func(cfg *Config) error {
				if cfg.AutoDetect != true {
					t.Errorf("AutoDetect = %v, want true", cfg.AutoDetect)
				}
				return nil
			},
		},
		{
			name:  "DISABLE_INTERACTIVE",
			key:   "DISABLE_INTERACTIVE",
			value: "true",
			validate: func(cfg *Config) error {
				if cfg.DisableInteractiveMode != true {
					t.Errorf("DisableInteractiveMode = %v, want true", cfg.DisableInteractiveMode)
				}
				return nil
			},
		},
		{
			name:  "USE_KERNEL_ARGS",
			key:   "USE_KERNEL_ARGS",
			value: "true",
			validate: func(cfg *Config) error {
				if cfg.UseKernelArgs != true {
					t.Errorf("UseKernelArgs = %v, want true", cfg.UseKernelArgs)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			err := ApplyValue(cfg, tt.key, tt.value)
			if err != nil {
				t.Fatalf("ApplyValue() error = %v", err)
			}
			tt.validate(cfg)
		})
	}
}

func TestApplyValue_UnknownKey(t *testing.T) {
	cfg := &Config{}
	// Unknown keys should not cause errors, just warnings
	err := ApplyValue(cfg, "UNKNOWN_KEY", "some_value")
	if err != nil {
		t.Errorf("ApplyValue() should not error on unknown keys, got: %v", err)
	}
}

func TestValidate_AllFieldsPresent(t *testing.T) {
	cfg := &Config{
		ObmSvc:       "obm.example.com",
		ObsSvc:       "obs.example.com",
		ObmPort:      50051,
		KeycloakURL:  "keycloak.example.com",
		CaCertPath:   "/path/to/cert",
		MacAddr:      "AA:BB:CC:DD:EE:FF",
		SerialNumber: "SERIAL123",
		UUID:         "11111111-1111-1111-1111-111111111111",
		IPAddress:    "192.168.1.100",
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() should not error with all fields present, got: %v", err)
	}
}

func TestValidate_MissingCriticalFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "missing ObmSvc",
			cfg: &Config{
				ObsSvc:      "obs.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
				CaCertPath:  "/path/to/cert",
			},
			wantErr: true,
		},
		{
			name: "missing ObsSvc",
			cfg: &Config{
				ObmSvc:      "obm.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
				CaCertPath:  "/path/to/cert",
			},
			wantErr: true,
		},
		{
			name: "missing ObmPort",
			cfg: &Config{
				ObmSvc:      "obm.example.com",
				ObsSvc:      "obs.example.com",
				KeycloakURL: "keycloak.example.com",
				CaCertPath:  "/path/to/cert",
			},
			wantErr: true,
		},
		{
			name: "missing KeycloakURL",
			cfg: &Config{
				ObmSvc:     "obm.example.com",
				ObsSvc:     "obs.example.com",
				ObmPort:    50051,
				CaCertPath: "/path/to/cert",
			},
			wantErr: true,
		},
		{
			name: "missing CaCertPath",
			cfg: &Config{
				ObmSvc:      "obm.example.com",
				ObsSvc:      "obs.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing non-critical fields (Serial, UUID, IP, MAC)",
			cfg: &Config{
				ObmSvc:      "obm.example.com",
				ObsSvc:      "obs.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
				CaCertPath:  "/path/to/cert",
				// SerialNumber, UUID, IPAddress, MacAddr can be auto-detected
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	// Skip this test as WriteToFile uses a hardcoded path (/etc/hook/env_config)
	// which requires root permissions and would interfere with the system
	t.Skip("WriteToFile uses hardcoded path - skipping test")
}

func TestWriteToFile_CreatesDirectory(t *testing.T) {
	// Skip this test as WriteToFile uses a hardcoded path (/etc/hook/env_config)
	// which requires root permissions and would interfere with the system
	t.Skip("WriteToFile uses hardcoded path - skipping test")
}

func TestLoadFromKernelArgs(t *testing.T) {
	// Create a temporary kernel args file
	tmpDir := t.TempDir()
	kernelArgsPath := filepath.Join(tmpDir, "cmdline")

	content := "console=ttyS0 DEBUG=true TIMEOUT=10m worker_id=test-worker-123"
	err := os.WriteFile(kernelArgsPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp kernel args file: %v", err)
	}

	// This test would require modifying LoadFromKernelArgs to accept a path parameter
	// For now, we'll skip this test as it requires access to /proc/cmdline
	t.Skip("Skipping kernel args test - requires modifying function to accept custom path")
}
