// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	TokenFolder             = "/dev/shm" // #nosec G101 -- This is a path, not a credential
	EnvConfigPath           = "/etc/device-discovery/validated-config.env"
	ExtraHostsFile          = "/etc/hosts"
	AccessTokenFile         = TokenFolder + "/idp_access_token"
	ReleaseTokenFile        = TokenFolder + "/release_token"
	KeycloakTokenURL        = "/realms/master/protocol/openid-connect/token"
	ReleaseTokenURL         = "/token"
	ClientCredentialsFolder = "/etc/intel_edge_node/client-credentials/" // #nosec G101 -- This is a path, not a credential
	ClientIDPath            = ClientCredentialsFolder + "client_id"
	ClientSecretPath        = ClientCredentialsFolder + "client_secret"
	KernelArgsFilePath      = "/proc/cmdline"
	ProjectIDPath           = ClientCredentialsFolder + "project_id"
)

// UpdateHosts updates /etc/hosts with extra host mappings.
func UpdateHosts(extraHosts string) error {
	// Update hosts if they were provided
	if extraHosts != "" {
		// Replace commas with newlines and remove double quotes
		extraHostsNeeded := strings.ReplaceAll(extraHosts, ",", "\n")
		extraHostsNeeded = strings.ReplaceAll(extraHostsNeeded, "\"", "")

		// Append to /etc/hosts
		hostsFile := "/etc/hosts"
		err := os.WriteFile(hostsFile, []byte(extraHostsNeeded), os.ModeAppend|0644)
		if err != nil {
			return fmt.Errorf("error updating /etc/hosts: %w", err)
		}

		fmt.Println("Adding extra host mappings completed")
	}
	return nil
}

// SaveToFile writes data to the specified file path with the given permissions.
// Creates the directory path if it doesn't exist.
func SaveToFile(path, data string) error {
	// Extract directory path from file path
	dir := filepath.Dir(path)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Writer interface to write data
	_, err = io.WriteString(file, data)
	return err
}

// kernelConfig holds the parsed kernel configuration from kernel arguments.
type kernelConfig struct {
	WorkerID string
	Debug    string
	Timeout  string
}

// parseCmdLine parses the kernel command line arguments and returns a kernelConfig.
func parseCmdLine(cmdLines []string) (kernelConfig, error) {
	var cfg kernelConfig
	for i := range cmdLines {
		cmdLine := strings.Split(cmdLines[i], "=")
		if len(cmdLine) == 0 {
			continue
		}

		// Check if we have both key and value (cmdLine must have at least 2 elements)
		if len(cmdLine) < 2 {
			continue
		}

		switch cmd := cmdLine[0]; cmd {
		case "worker_id":
			cfg.WorkerID = cmdLine[1]
		case "DEBUG":
			cfg.Debug = cmdLine[1]
		case "TIMEOUT":
			cfg.Timeout = cmdLine[1]
		}
	}
	return cfg, nil
}

// parseKernelArguments reads the kernel command line from the specified file
// and returns the parsed configuration.
func parseKernelArguments(kernelArgsFilePath string) (kernelConfig, error) {
	content, err := os.ReadFile(kernelArgsFilePath)
	if err != nil {
		return kernelConfig{}, err
	}
	cmdLines := strings.Split(string(content), " ")
	return parseCmdLine(cmdLines)
}

// Config holds all command-line and file-based configuration
type Config struct {
	// Config file
	ConfigFile string

	// Kernel arguments flag
	UseKernelArgs bool

	// Service endpoints
	ObmSvc      string
	ObsSvc      string
	ObmPort     int
	KeycloakURL string

	// Device information
	MacAddr      string
	SerialNumber string
	UUID         string
	IPAddress    string

	// Optional configuration
	ExtraHosts             string
	CaCertPath             string
	Debug                  bool
	Timeout                time.Duration
	DisableInteractiveMode bool

	// Auto-detection flags
	AutoDetect bool
}

// LoadFromFile loads configuration from a file and updates cfg.
// The file should be in KEY=VALUE format (one per line, # for comments).
// Values from the config file can be overwritten by higher priority sources later.
func LoadFromFile(cfg *Config, configFile string) error {
	file, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Parse config file
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format at line %d: %s (expected KEY=VALUE)", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Apply value to config (will be overwritten by higher priority sources later)
		if err := ApplyValue(cfg, key, value); err != nil {
			return fmt.Errorf("error at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// LoadFromKernelArgs reads and parses kernel arguments from /proc/cmdline.
// Values from kernel arguments can be overwritten by config file and CLI flags later.
func LoadFromKernelArgs(cfg *Config) error {

	// Parse kernel arguments using internal parseKernelArguments function
	kernelCfg, err := parseKernelArguments(KernelArgsFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse kernel arguments from %s: %w", KernelArgsFilePath, err)
	}

	// Apply kernel argument values (will be overwritten by CLI flags later if they're set)
	// Map worker_id from Tinkerbell to MAC address
	if kernelCfg.WorkerID != "" {
		cfg.MacAddr = kernelCfg.WorkerID
		fmt.Printf("Mapped worker_id from kernel args to MAC address: %s\n", kernelCfg.WorkerID)
	}

	// DEBUG flag from kernel args
	if kernelCfg.Debug != "" {
		debug, err := strconv.ParseBool(kernelCfg.Debug)
		if err == nil {
			cfg.Debug = debug
			if debug {
				fmt.Println("Debug mode enabled via kernel arguments")
			}
		}
	}

	// TIMEOUT flag from kernel args
	if kernelCfg.Timeout != "" {
		timeout, err := time.ParseDuration(kernelCfg.Timeout)
		if err == nil {
			cfg.Timeout = timeout
			fmt.Printf("Timeout set from kernel args: %v\n", cfg.Timeout)
		}
	}

	return nil
}

// ApplyValue applies a configuration value to the appropriate field.
// Simply updates the value - priority is handled by calling order.
func ApplyValue(cfg *Config, key, value string) error {
	// Map config file keys to field names
	switch key {
	case "OBM_SVC":
		cfg.ObmSvc = value
	case "OBS_SVC":
		cfg.ObsSvc = value
	case "OBM_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port value '%s': %w", value, err)
		}
		cfg.ObmPort = port
	case "KEYCLOAK_URL":
		cfg.KeycloakURL = value
	case "MAC":
		cfg.MacAddr = value
	case "SERIAL":
		cfg.SerialNumber = value
	case "UUID":
		cfg.UUID = value
	case "IP":
		cfg.IPAddress = value
	case "EXTRA_HOSTS":
		cfg.ExtraHosts = value
	case "CA_CERT":
		cfg.CaCertPath = value
	case "DEBUG":
		debug, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid debug value '%s': %w", value, err)
		}
		cfg.Debug = debug
	case "TIMEOUT":
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid timeout value '%s': %w", value, err)
		}
		cfg.Timeout = timeout
	case "AUTO_DETECT":
		autoDetect, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto-detect value '%s': %w", value, err)
		}
		cfg.AutoDetect = autoDetect
	case "DISABLE_INTERACTIVE":
		disableInteractive, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid disable-interactive value '%s': %w", value, err)
		}
		cfg.DisableInteractiveMode = disableInteractive
	case "USE_KERNEL_ARGS":
		// Parse this value to enable kernel args parsing if set in config
		useKernelArgs, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid use-kernel-args value '%s': %w", value, err)
		}
		cfg.UseKernelArgs = useKernelArgs
	default:
		// Unknown key - skip it with a warning
		fmt.Fprintf(os.Stderr, "Warning: unknown config key '%s' (skipping)\n", key)
	}

	return nil
}

// Validate checks if all required configuration fields are set.
// Returns an error listing any missing critical fields.
func Validate(cfg *Config) error {
	var missing []string

	if cfg.ObmSvc == "" {
		missing = append(missing, "OBM_SVC")
	}
	if cfg.ObsSvc == "" {
		missing = append(missing, "OBS_SVC")
	}
	if cfg.ObmPort == 0 {
		missing = append(missing, "OBM_PORT")
	}
	if cfg.KeycloakURL == "" {
		missing = append(missing, "KEYCLOAK_URL")
	}
	if cfg.CaCertPath == "" {
		missing = append(missing, "CA_CERT")
	}
	if cfg.MacAddr == "" {
		missing = append(missing, "MAC")
	}
	if cfg.SerialNumber == "" {
		missing = append(missing, "SERIAL")
	}
	if cfg.UUID == "" {
		missing = append(missing, "UUID")
	}
	if cfg.IPAddress == "" {
		missing = append(missing, "IP")
	}

	// Only fail on critical missing fields
	criticalMissing := []string{}
	for _, field := range missing {
		// These fields can be auto-detected, so they're not critical at this stage
		if field != "SERIAL" && field != "UUID" && field != "IP" && field != "MAC" {
			criticalMissing = append(criticalMissing, field)
		}
	}

	if len(criticalMissing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(criticalMissing, ", "))
	}

	return nil
}

// WriteToFile writes the validated configuration to a file in KEY=VALUE format.
func WriteToFile(cfg *Config) error {
	// Ensure the directory exists
	configDir := strings.TrimSuffix(EnvConfigPath, "/"+strings.Split(EnvConfigPath, "/")[len(strings.Split(EnvConfigPath, "/"))-1])
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create the config file
	file, err := os.Create(EnvConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# Device Discovery Validated Configuration\n")
	fmt.Fprintf(file, "# Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	// Write configuration values
	fmt.Fprintf(file, "OBM_SVC=%s\n", cfg.ObmSvc)
	fmt.Fprintf(file, "OBS_SVC=%s\n", cfg.ObsSvc)
	fmt.Fprintf(file, "OBM_PORT=%d\n", cfg.ObmPort)
	fmt.Fprintf(file, "KEYCLOAK_URL=%s\n", cfg.KeycloakURL)
	fmt.Fprintf(file, "MAC=%s\n", cfg.MacAddr)
	fmt.Fprintf(file, "SERIAL=%s\n", cfg.SerialNumber)
	fmt.Fprintf(file, "UUID=%s\n", cfg.UUID)
	fmt.Fprintf(file, "IP=%s\n", cfg.IPAddress)
	fmt.Fprintf(file, "CA_CERT=%s\n", cfg.CaCertPath)

	if cfg.ExtraHosts != "" {
		fmt.Fprintf(file, "EXTRA_HOSTS=%s\n", cfg.ExtraHosts)
	}

	fmt.Fprintf(file, "DEBUG=%t\n", cfg.Debug)
	fmt.Fprintf(file, "DISABLE_INTERACTIVE=%t\n", cfg.DisableInteractiveMode)
	if cfg.Debug {
		fmt.Fprintf(file, "TIMEOUT=%s\n", cfg.Timeout.String())
	}

	fmt.Printf("Configuration written to: %s\n", EnvConfigPath)
	return nil
}
