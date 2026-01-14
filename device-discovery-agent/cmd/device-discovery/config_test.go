// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// resetFlags resets the flag package state for testing
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

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

func TestLoadConfigFile_BasicLoading(t *testing.T) {
	resetFlags()

	configContent := `# Test config
OBM_SVC=config.example.com
OBS_SVC=stream.example.com
OBM_PORT=50051
KEYCLOAK_URL=keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
SERIAL=CONFIG_SERIAL
UUID=11111111-1111-1111-1111-111111111111
IP=192.168.1.50
DEBUG=true
TIMEOUT=10m
AUTO_DETECT=false
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	// Parse empty flags (no CLI overrides)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
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
		{"Debug", cfg.Debug, true},
		{"Timeout", cfg.Timeout, 10 * time.Minute},
		{"AutoDetect", cfg.AutoDetect, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoadConfigFile_CLIFlagOverride(t *testing.T) {
	resetFlags()

	configContent := `OBM_SVC=config.example.com
OBS_SVC=stream.example.com
OBM_PORT=50051
KEYCLOAK_URL=keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
DEBUG=false
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	// Simulate CLI flags being set
	flag.String("obm-svc", "", "")
	flag.String("mac", "", "")
	flag.Bool("debug", false, "")

	// Set specific values via "CLI" (before parsing config file)
	cfg.ObmSvc = "cli.example.com"    // This would be set by CLI flag
	cfg.MacAddr = "11:22:33:44:55:66" // This would be set by CLI flag
	cfg.Debug = true                  // This would be set by CLI flag

	// Mark these as explicitly set
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("obm-svc", "cli.example.com", "")
	flag.String("mac", "11:22:33:44:55:66", "")
	flag.Bool("debug", true, "")

	// Simulate the flags being visited
	flag.Set("obm-svc", "cli.example.com")
	flag.Set("mac", "11:22:33:44:55:66")
	flag.Set("debug", "true")

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	// CLI flags should win
	if cfg.ObmSvc != "cli.example.com" {
		t.Errorf("ObmSvc should be from CLI, got %v", cfg.ObmSvc)
	}
	if cfg.MacAddr != "11:22:33:44:55:66" {
		t.Errorf("MacAddr should be from CLI, got %v", cfg.MacAddr)
	}
	if cfg.Debug != true {
		t.Errorf("Debug should be from CLI, got %v", cfg.Debug)
	}

	// Config file values should be used for non-overridden flags
	if cfg.ObsSvc != "stream.example.com" {
		t.Errorf("ObsSvc should be from config, got %v", cfg.ObsSvc)
	}
	if cfg.ObmPort != 50051 {
		t.Errorf("ObmPort should be from config, got %v", cfg.ObmPort)
	}
}

func TestLoadConfigFile_PartialOverride(t *testing.T) {
	resetFlags()

	configContent := `OBM_SVC=config-obm.example.com
OBS_SVC=config-obs.example.com
OBM_PORT=50051
KEYCLOAK_URL=config-keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
SERIAL=CONFIG_SERIAL
UUID=11111111-1111-1111-1111-111111111111
IP=192.168.1.50
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	// Simulate only MAC being set via CLI
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("mac", "FF:EE:DD:CC:BB:AA", "")
	flag.Set("mac", "FF:EE:DD:CC:BB:AA")
	cfg.MacAddr = "FF:EE:DD:CC:BB:AA"

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	// MAC should be from CLI
	if cfg.MacAddr != "FF:EE:DD:CC:BB:AA" {
		t.Errorf("MacAddr should be from CLI, got %v", cfg.MacAddr)
	}

	// All other values should be from config
	if cfg.ObmSvc != "config-obm.example.com" {
		t.Errorf("ObmSvc should be from config, got %v", cfg.ObmSvc)
	}
	if cfg.ObsSvc != "config-obs.example.com" {
		t.Errorf("ObsSvc should be from config, got %v", cfg.ObsSvc)
	}
	if cfg.SerialNumber != "CONFIG_SERIAL" {
		t.Errorf("SerialNumber should be from config, got %v", cfg.SerialNumber)
	}
	if cfg.UUID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("UUID should be from config, got %v", cfg.UUID)
	}
}

func TestLoadConfigFile_EmptyAndComments(t *testing.T) {
	resetFlags()

	configContent := `# This is a comment
# Another comment

OBM_SVC=example.com

# Empty lines above and below

OBM_PORT=50051
# Inline comment - key should still work
KEYCLOAK_URL=keycloak.example.com
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	if cfg.ObmSvc != "example.com" {
		t.Errorf("ObmSvc = %v, want example.com", cfg.ObmSvc)
	}
	if cfg.ObmPort != 50051 {
		t.Errorf("ObmPort = %v, want 50051", cfg.ObmPort)
	}
	if cfg.KeycloakURL != "keycloak.example.com" {
		t.Errorf("KeycloakURL = %v, want keycloak.example.com", cfg.KeycloakURL)
	}
}

func TestLoadConfigFile_QuotedValues(t *testing.T) {
	resetFlags()

	configContent := `OBM_SVC="quoted.example.com"
OBS_SVC='single-quoted.example.com'
EXTRA_HOSTS="host1:192.168.1.1,host2:192.168.1.2"
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	if cfg.ObmSvc != "quoted.example.com" {
		t.Errorf("ObmSvc = %v, want quoted.example.com", cfg.ObmSvc)
	}
	if cfg.ObsSvc != "single-quoted.example.com" {
		t.Errorf("ObsSvc = %v, want single-quoted.example.com", cfg.ObsSvc)
	}
	if cfg.ExtraHosts != "host1:192.168.1.1,host2:192.168.1.2" {
		t.Errorf("ExtraHosts = %v, want host1:192.168.1.1,host2:192.168.1.2", cfg.ExtraHosts)
	}
}

func TestLoadConfigFile_InvalidFormat(t *testing.T) {
	resetFlags()

	tests := []struct {
		name          string
		content       string
		expectedError string
	}{
		{
			name:          "Missing equals sign",
			content:       "INVALID LINE WITHOUT EQUALS",
			expectedError: "invalid format at line 1",
		},
		{
			name:          "Invalid port",
			content:       "OBM_PORT=not_a_number",
			expectedError: "invalid port value",
		},
		{
			name:          "Invalid boolean",
			content:       "DEBUG=maybe",
			expectedError: "invalid debug value",
		},
		{
			name:          "Invalid timeout",
			content:       "TIMEOUT=invalid_duration",
			expectedError: "invalid timeout value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := createTempConfigFile(t, tt.content)

			cfg := &CLIConfig{
				ConfigFile: configPath,
			}

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			err := loadConfigFile(cfg)
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if !contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%v'", tt.expectedError, err)
			}
		})
	}
}

func TestLoadConfigFile_UnknownKeys(t *testing.T) {
	resetFlags()

	// This should not fail, just warn
	configContent := `OBM_SVC=example.com
UNKNOWN_KEY=some_value
ANOTHER_UNKNOWN=123
OBM_PORT=50051
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile should not fail on unknown keys: %v", err)
	}

	// Known keys should still work
	if cfg.ObmSvc != "example.com" {
		t.Errorf("ObmSvc = %v, want example.com", cfg.ObmSvc)
	}
	if cfg.ObmPort != 50051 {
		t.Errorf("ObmPort = %v, want 50051", cfg.ObmPort)
	}
}

func TestLoadConfigFile_FileNotFound(t *testing.T) {
	resetFlags()

	cfg := &CLIConfig{
		ConfigFile: "/nonexistent/path/config.env",
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "failed to open config file") {
		t.Errorf("Expected 'failed to open config file' error, got: %v", err)
	}
}

func TestLoadConfigFile_MultipleOverrides(t *testing.T) {
	resetFlags()

	configContent := `OBM_SVC=config.example.com
OBS_SVC=config-stream.example.com
OBM_PORT=50051
KEYCLOAK_URL=config-keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
SERIAL=CONFIG_SERIAL
UUID=11111111-1111-1111-1111-111111111111
IP=192.168.1.50
DEBUG=false
TIMEOUT=5m
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	// Simulate multiple CLI flags being set
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("obm-svc", "cli-obm.example.com", "")
	flag.String("mac", "FF:EE:DD:CC:BB:AA", "")
	flag.Bool("debug", true, "")
	flag.Duration("timeout", 15*time.Minute, "")

	flag.Set("obm-svc", "cli-obm.example.com")
	flag.Set("mac", "FF:EE:DD:CC:BB:AA")
	flag.Set("debug", "true")
	flag.Set("timeout", "15m")

	cfg.ObmSvc = "cli-obm.example.com"
	cfg.MacAddr = "FF:EE:DD:CC:BB:AA"
	cfg.Debug = true
	cfg.Timeout = 15 * time.Minute

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	// CLI overrides should win
	if cfg.ObmSvc != "cli-obm.example.com" {
		t.Errorf("ObmSvc should be from CLI, got %v", cfg.ObmSvc)
	}
	if cfg.MacAddr != "FF:EE:DD:CC:BB:AA" {
		t.Errorf("MacAddr should be from CLI, got %v", cfg.MacAddr)
	}
	if cfg.Debug != true {
		t.Errorf("Debug should be from CLI, got %v", cfg.Debug)
	}
	if cfg.Timeout != 15*time.Minute {
		t.Errorf("Timeout should be from CLI, got %v", cfg.Timeout)
	}

	// Config values should be used for non-overridden flags
	if cfg.ObsSvc != "config-stream.example.com" {
		t.Errorf("ObsSvc should be from config, got %v", cfg.ObsSvc)
	}
	if cfg.ObmPort != 50051 {
		t.Errorf("ObmPort should be from config, got %v", cfg.ObmPort)
	}
	if cfg.SerialNumber != "CONFIG_SERIAL" {
		t.Errorf("SerialNumber should be from config, got %v", cfg.SerialNumber)
	}
	if cfg.IPAddress != "192.168.1.50" {
		t.Errorf("IPAddress should be from config, got %v", cfg.IPAddress)
	}
}

func TestLoadConfigFile_AllTypesConversion(t *testing.T) {
	resetFlags()

	configContent := `OBM_PORT=8080
DEBUG=true
TIMEOUT=30m
AUTO_DETECT=false
`

	configPath := createTempConfigFile(t, configContent)

	cfg := &CLIConfig{
		ConfigFile: configPath,
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := loadConfigFile(cfg)
	if err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	// Test type conversions
	if cfg.ObmPort != 8080 {
		t.Errorf("ObmPort should be 8080, got %v", cfg.ObmPort)
	}
	if cfg.Debug != true {
		t.Errorf("Debug should be true, got %v", cfg.Debug)
	}
	if cfg.Timeout != 30*time.Minute {
		t.Errorf("Timeout should be 30m, got %v", cfg.Timeout)
	}
	if cfg.AutoDetect != false {
		t.Errorf("AutoDetect should be false, got %v", cfg.AutoDetect)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPriorityScenarios tests real-world priority scenarios between config file and CLI flags
func TestPriorityScenarios(t *testing.T) {
	t.Run("Scenario1_ConfigFileOnly", func(t *testing.T) {
		// When: Only config file is provided (no CLI flags)
		// Then: All values should come from config file
		resetFlags()

		configContent := `OBM_SVC=config.example.com
OBS_SVC=stream.example.com
OBM_PORT=50051
KEYCLOAK_URL=keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		assertEqual(t, "ObmSvc", cfg.ObmSvc, "config.example.com")
		assertEqual(t, "ObsSvc", cfg.ObsSvc, "stream.example.com")
		assertEqual(t, "ObmPort", cfg.ObmPort, 50051)
		assertEqual(t, "MacAddr", cfg.MacAddr, "AA:BB:CC:DD:EE:FF")
	})

	t.Run("Scenario2_CLIOverridesAll", func(t *testing.T) {
		// When: Both config file and CLI flags are provided
		// Then: CLI flags should win for all specified values
		resetFlags()

		configContent := `OBM_SVC=config.example.com
OBS_SVC=stream.example.com
MAC=AA:BB:CC:DD:EE:FF
DEBUG=false
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}

		// Simulate CLI flags being set
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.String("obm-svc", "cli.example.com", "")
		flag.String("mac", "FF:EE:DD:CC:BB:AA", "")
		flag.Bool("debug", true, "")

		flag.Set("obm-svc", "cli.example.com")
		flag.Set("mac", "FF:EE:DD:CC:BB:AA")
		flag.Set("debug", "true")

		cfg.ObmSvc = "cli.example.com"
		cfg.MacAddr = "FF:EE:DD:CC:BB:AA"
		cfg.Debug = true

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// CLI values should win
		assertEqual(t, "ObmSvc", cfg.ObmSvc, "cli.example.com")
		assertEqual(t, "MacAddr", cfg.MacAddr, "FF:EE:DD:CC:BB:AA")
		assertEqual(t, "Debug", cfg.Debug, true)

		// Non-overridden value should come from config
		assertEqual(t, "ObsSvc", cfg.ObsSvc, "stream.example.com")
	})

	t.Run("Scenario3_PartialCLIOverride", func(t *testing.T) {
		// When: Config file provides all values, CLI overrides only one
		// Then: CLI value wins for that one, rest come from config
		resetFlags()

		configContent := `OBM_SVC=config-obm.example.com
OBS_SVC=config-obs.example.com
OBM_PORT=50051
KEYCLOAK_URL=config-keycloak.example.com
MAC=AA:BB:CC:DD:EE:FF
SERIAL=CONFIG_SERIAL
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}

		// Only override MAC via CLI
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.String("mac", "99:88:77:66:55:44", "")
		flag.Set("mac", "99:88:77:66:55:44")
		cfg.MacAddr = "99:88:77:66:55:44"

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// CLI override should win
		assertEqual(t, "MacAddr", cfg.MacAddr, "99:88:77:66:55:44")

		// All other values should come from config
		assertEqual(t, "ObmSvc", cfg.ObmSvc, "config-obm.example.com")
		assertEqual(t, "ObsSvc", cfg.ObsSvc, "config-obs.example.com")
		assertEqual(t, "ObmPort", cfg.ObmPort, 50051)
		assertEqual(t, "KeycloakURL", cfg.KeycloakURL, "config-keycloak.example.com")
		assertEqual(t, "SerialNumber", cfg.SerialNumber, "CONFIG_SERIAL")
	})

	t.Run("Scenario4_MultipleTypedOverrides", func(t *testing.T) {
		// When: Different types of values are overridden
		// Then: Type conversions should work correctly for all
		resetFlags()

		configContent := `OBM_PORT=50051
DEBUG=false
TIMEOUT=5m
AUTO_DETECT=false
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}

		// Override with different types
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.Int("obm-port", 8080, "")
		flag.Bool("debug", true, "")
		flag.Duration("timeout", 15*time.Minute, "")

		flag.Set("obm-port", "8080")
		flag.Set("debug", "true")
		flag.Set("timeout", "15m")

		cfg.ObmPort = 8080
		cfg.Debug = true
		cfg.Timeout = 15 * time.Minute

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// CLI overrides should win
		assertEqual(t, "ObmPort", cfg.ObmPort, 8080)
		assertEqual(t, "Debug", cfg.Debug, true)
		assertEqual(t, "Timeout", cfg.Timeout, 15*time.Minute)

		// Non-overridden config value
		assertEqual(t, "AutoDetect", cfg.AutoDetect, false)
	})

	t.Run("Scenario5_EmptyVsExplicitValues", func(t *testing.T) {
		// When: Config has values, CLI flag is explicitly set to empty/zero
		// Then: CLI's explicit empty/zero should win
		resetFlags()

		configContent := `OBM_SVC=config.example.com
SERIAL=CONFIG_SERIAL
DEBUG=true
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}

		// Explicitly set to empty/false via CLI
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.String("serial", "", "")
		flag.Bool("debug", false, "")

		flag.Set("serial", "")
		flag.Set("debug", "false")

		cfg.SerialNumber = ""
		cfg.Debug = false

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// CLI's explicit empty/false should win
		assertEqual(t, "SerialNumber", cfg.SerialNumber, "")
		assertEqual(t, "Debug", cfg.Debug, false)

		// Non-overridden value
		assertEqual(t, "ObmSvc", cfg.ObmSvc, "config.example.com")
	})

	t.Run("Scenario6_ProductionDeployment", func(t *testing.T) {
		// Real-world scenario: Production config + runtime MAC override
		resetFlags()

		configContent := `# Production Infrastructure Config
OBM_SVC=obm.prod.example.com
OBS_SVC=obs.prod.example.com
OBM_PORT=50051
KEYCLOAK_URL=keycloak.prod.example.com
CA_CERT=/etc/certs/prod-ca.pem
DEBUG=false
TIMEOUT=5m
`
		configPath := createTempConfigFile(t, configContent)
		cfg := &CLIConfig{ConfigFile: configPath}

		// Runtime: override MAC for specific device
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.String("mac", "device-mac-address", "")
		flag.Set("mac", "device-mac-address")
		cfg.MacAddr = "device-mac-address"

		if err := loadConfigFile(cfg); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Infrastructure values from config
		assertEqual(t, "ObmSvc", cfg.ObmSvc, "obm.prod.example.com")
		assertEqual(t, "ObsSvc", cfg.ObsSvc, "obs.prod.example.com")
		assertEqual(t, "ObmPort", cfg.ObmPort, 50051)
		assertEqual(t, "KeycloakURL", cfg.KeycloakURL, "keycloak.prod.example.com")
		assertEqual(t, "CaCertPath", cfg.CaCertPath, "/etc/certs/prod-ca.pem")

		// Runtime device-specific override
		assertEqual(t, "MacAddr", cfg.MacAddr, "device-mac-address")
	})
}

// Helper function for assertions
func assertEqual(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// TestConfigFilePriorityDocumentation documents the priority behavior
func TestConfigFilePriorityDocumentation(t *testing.T) {
	t.Run("DocumentedBehavior", func(t *testing.T) {
		// This test serves as executable documentation of the priority rules
		t.Log("Config File and CLI Flag Priority Rules:")
		t.Log("")
		t.Log("1. CLI flags ALWAYS override config file values")
		t.Log("2. Config file provides default/base values")
		t.Log("3. Flags not set via CLI use config file values")
		t.Log("4. Explicit empty/zero CLI values override non-empty config values")
		t.Log("5. Type conversions are handled correctly (string, int, bool, duration)")
		t.Log("")
		t.Log("Use case: Infrastructure config in file + device-specific values via CLI")
	})
}
