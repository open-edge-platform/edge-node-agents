// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"device-discovery/internal/config"
	"device-discovery/internal/mode"
	"device-discovery/internal/sysinfo"
)

func main() {
	// Initialize config with defaults
	cfg := &config.Config{}

	// Priority order (lowest to highest): defaults < kernel args < config file < CLI flags
	// Apply in order so that higher priority values overwrite lower priority ones

	// Step 1: Parse CLI flags to get -config and -use-kernel-args flags
	parseInitialFlags(cfg)

	// Track if kernel args were explicitly enabled via CLI
	kernelArgsFromCLI := cfg.UseKernelArgs

	// Step 2: Load config file first (if specified)
	// This allows us to check if USE_KERNEL_ARGS is set in the config
	if cfg.ConfigFile != "" {
		if err := config.LoadFromFile(cfg, cfg.ConfigFile); err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	}

	// Step 3: Parse kernel arguments if enabled (by CLI or config file)
	// Kernel args have lower priority than config file, so we parse them first,
	// then reload config file to let it override
	if cfg.UseKernelArgs {
		if err := config.LoadFromKernelArgs(cfg); err != nil {
			log.Fatalf("Failed to parse kernel arguments: %v", err)
		}

		// If USE_KERNEL_ARGS was set in config file (not CLI), reload config to override kernel args
		if !kernelArgsFromCLI && cfg.ConfigFile != "" {
			if err := config.LoadFromFile(cfg, cfg.ConfigFile); err != nil {
				log.Fatalf("Failed to reload config file: %v", err)
			}
		}
	}

	// Step 4: Re-parse CLI flags - overrides everything (highest priority)
	parseFinalFlags(cfg)

	// Validate required flags if not auto-detecting
	if !cfg.AutoDetect {
		validateConfig(cfg)
	}

	// Auto-detect system information if requested
	if cfg.AutoDetect || cfg.MacAddr != "" {
		if err := sysinfo.AutoDetectSystemInfo(cfg); err != nil {
			log.Fatalf("Failed to auto-detect system information: %v", err)
		}
	}

	// Validate after auto-detection
	validateConfig(cfg)

	// Write validated configuration to file
	if err := config.WriteToFile(cfg); err != nil {
		log.Fatalf("Failed to write configuration to file: %v", err)
	}

	// Add extra hosts if provided
	if cfg.ExtraHosts != "" {
		if err := config.UpdateHosts(cfg.ExtraHosts); err != nil {
			log.Fatalf("Failed to add extra hosts: %v", err)
		}
	}

	// Display configuration
	fmt.Println("Device Discovery Configuration:")
	fmt.Printf("  Onboarding Manager: %s:%d\n", cfg.ObmSvc, cfg.ObmPort)
	fmt.Printf("  Onboarding Stream: %s:%d\n", cfg.ObsSvc, cfg.ObmPort)
	fmt.Printf("  Keycloak URL: %s\n", cfg.KeycloakURL)
	fmt.Printf("  MAC Address: %s\n", cfg.MacAddr)
	fmt.Printf("  Serial Number: %s\n", cfg.SerialNumber)
	fmt.Printf("  UUID: %s\n", cfg.UUID)
	fmt.Printf("  IP Address: %s\n", cfg.IPAddress)
	fmt.Printf("  Debug Mode: %v\n", cfg.Debug)
	if cfg.Debug {
		fmt.Printf("  Timeout: %v\n", cfg.Timeout)
	}
	fmt.Println()

	// Run device discovery
	if err := deviceDiscovery(cfg); err != nil {
		log.Fatalf("Device discovery failed: %v", err)
	}

	fmt.Println("Device discovery completed successfully")
}

// parseInitialFlags does the first parse to get -config and -use-kernel-args flags only
func parseInitialFlags(cfg *config.Config) {
	// We only need these two flags in the initial parse
	flag.StringVar(&cfg.ConfigFile, "config", "", "Path to configuration file (optional)")
	flag.BoolVar(&cfg.UseKernelArgs, "use-kernel-args", false, "Read configuration from kernel arguments (/proc/cmdline)")

	// Parse to get config file path and kernel args flag
	flag.Parse()
}

// parseFinalFlags parses all CLI flags and applies them (highest priority)
func parseFinalFlags(cfg *config.Config) {
	// Reset flag set to allow re-parsing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Config file
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "Path to configuration file (optional)")

	// Kernel arguments
	flag.BoolVar(&cfg.UseKernelArgs, "use-kernel-args", cfg.UseKernelArgs, "Read configuration from kernel arguments (/proc/cmdline)")

	// Service endpoints - use current values as defaults
	flag.StringVar(&cfg.ObmSvc, "obm-svc", cfg.ObmSvc, "Onboarding manager service address (required)")
	flag.StringVar(&cfg.ObsSvc, "obs-svc", cfg.ObsSvc, "Onboarding stream service address (required)")
	flag.IntVar(&cfg.ObmPort, "obm-port", cfg.ObmPort, "Onboarding manager port (required)")
	flag.StringVar(&cfg.KeycloakURL, "keycloak-url", cfg.KeycloakURL, "Keycloak authentication URL (required)")

	// Device information - use current values as defaults
	flag.StringVar(&cfg.MacAddr, "mac", cfg.MacAddr, "MAC address of the device (required unless auto-detect)")
	flag.StringVar(&cfg.SerialNumber, "serial", cfg.SerialNumber, "Serial number (auto-detected if not provided)")
	flag.StringVar(&cfg.UUID, "uuid", cfg.UUID, "System UUID (auto-detected if not provided)")
	flag.StringVar(&cfg.IPAddress, "ip", cfg.IPAddress, "IP address (auto-detected from MAC if not provided)")

	// Optional configuration - use current values as defaults
	flag.StringVar(&cfg.ExtraHosts, "extra-hosts", cfg.ExtraHosts, "Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')")
	flag.StringVar(&cfg.CaCertPath, "ca-cert", cfg.CaCertPath, "Path to CA certificate (required)")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "Enable debug mode with timeout")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Timeout duration for debug mode")
	flag.BoolVar(&cfg.DisableInteractiveMode, "disable-interactive", cfg.DisableInteractiveMode, "Disable interactive mode fallback")

	// Auto-detection - use current value as default
	flag.BoolVar(&cfg.AutoDetect, "auto-detect", cfg.AutoDetect, "Auto-detect all system information (MAC, serial, UUID, IP)")

	// Custom usage message
	flag.Usage = printUsage

	// Parse CLI flags - these will override any previously set values
	flag.Parse()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Device Discovery CLI - Onboard devices to the Open Edge platform\n\n")
	fmt.Fprintf(os.Stderr, "Required Options:\n")
	fmt.Fprintf(os.Stderr, "  -obm-svc string\n")
	fmt.Fprintf(os.Stderr, "        Onboarding manager service address\n")
	fmt.Fprintf(os.Stderr, "  -obs-svc string\n")
	fmt.Fprintf(os.Stderr, "        Onboarding stream service address\n")
	fmt.Fprintf(os.Stderr, "  -obm-port int\n")
	fmt.Fprintf(os.Stderr, "        Onboarding manager port\n")
	fmt.Fprintf(os.Stderr, "  -keycloak-url string\n")
	fmt.Fprintf(os.Stderr, "        Keycloak authentication URL\n")
	fmt.Fprintf(os.Stderr, "  -ca-cert string\n")
	fmt.Fprintf(os.Stderr, "        Path to CA certificate\n")
	fmt.Fprintf(os.Stderr, "  -mac string\n")
	fmt.Fprintf(os.Stderr, "        MAC address of the device (required unless -auto-detect is used)\n")
	fmt.Fprintf(os.Stderr, "\nOptional Device Information:\n")
	fmt.Fprintf(os.Stderr, "  -serial string\n")
	fmt.Fprintf(os.Stderr, "        Serial number (auto-detected if not provided)\n")
	fmt.Fprintf(os.Stderr, "  -uuid string\n")
	fmt.Fprintf(os.Stderr, "        System UUID (auto-detected if not provided)\n")
	fmt.Fprintf(os.Stderr, "  -ip string\n")
	fmt.Fprintf(os.Stderr, "        IP address (auto-detected from MAC if not provided)\n")
	fmt.Fprintf(os.Stderr, "\nAuto-Detection:\n")
	fmt.Fprintf(os.Stderr, "  -auto-detect\n")
	fmt.Fprintf(os.Stderr, "        Auto-detect all system information (MAC, serial, UUID, IP)\n")
	fmt.Fprintf(os.Stderr, "\nAdditional Options:\n")
	fmt.Fprintf(os.Stderr, "  -extra-hosts string\n")
	fmt.Fprintf(os.Stderr, "        Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')\n")
	fmt.Fprintf(os.Stderr, "  -disable-interactive\n")
	fmt.Fprintf(os.Stderr, "        Disable interactive mode fallback (fail if non-interactive mode fails)\n")
	fmt.Fprintf(os.Stderr, "  -debug\n")
	fmt.Fprintf(os.Stderr, "        Enable debug mode with timeout\n")
	fmt.Fprintf(os.Stderr, "  -timeout duration\n")
	fmt.Fprintf(os.Stderr, "        Timeout duration for debug mode (default: 5m0s)\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration File:\n")
	fmt.Fprintf(os.Stderr, "  -config string\n")
	fmt.Fprintf(os.Stderr, "        Path to configuration file (optional)\n")
	fmt.Fprintf(os.Stderr, "        Format: KEY=VALUE (one per line, # for comments)\n")
	fmt.Fprintf(os.Stderr, "\nKernel Arguments:\n")
	fmt.Fprintf(os.Stderr, "  -use-kernel-args\n")
	fmt.Fprintf(os.Stderr, "        Read configuration from kernel arguments (/proc/cmdline)\n")
	fmt.Fprintf(os.Stderr, "        Supports: worker_id (mapped to MAC), DEBUG, TIMEOUT\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration Priority (highest to lowest):\n")
	fmt.Fprintf(os.Stderr, "  1. CLI flags (highest priority)\n")
	fmt.Fprintf(os.Stderr, "  2. Config file (if -config is set)\n")
	fmt.Fprintf(os.Stderr, "  3. Kernel arguments (if -use-kernel-args is set)\n")
	fmt.Fprintf(os.Stderr, "  4. Default values (lowest priority)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # Auto-detect all system information\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -auto-detect\n\n")
	fmt.Fprintf(os.Stderr, "  # Specify MAC address, auto-detect others\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -mac 00:11:22:33:44:55\n\n")
	fmt.Fprintf(os.Stderr, "  # Fully manual configuration\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -mac 00:11:22:33:44:55 \\\n")
	fmt.Fprintf(os.Stderr, "    -serial ABC123 -uuid 12345678-1234-1234-1234-123456789012 -ip 192.168.1.100\n\n")
	fmt.Fprintf(os.Stderr, "  # With debug mode and extra hosts\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -auto-detect -debug -timeout 10m \\\n")
	fmt.Fprintf(os.Stderr, "    -extra-hosts \"registry.local:10.0.0.1,api.local:10.0.0.2\"\n\n")
	fmt.Fprintf(os.Stderr, "  # Using a configuration file\n")
	fmt.Fprintf(os.Stderr, "  %s -config /path/to/config.env\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  # Override config file with CLI flags\n")
	fmt.Fprintf(os.Stderr, "  %s -config /path/to/config.env -mac 00:11:22:33:44:55 -debug\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  # Use kernel arguments (useful for boot-time onboarding)\n")
	fmt.Fprintf(os.Stderr, "  %s -use-kernel-args -obm-svc obm.example.com -obs-svc obs.example.com \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -obm-port 50051 -keycloak-url keycloak.example.com -auto-detect\n\n")
	fmt.Fprintf(os.Stderr, "  # Combined: config file + kernel args + CLI override\n")
	fmt.Fprintf(os.Stderr, "  %s -config /etc/device-discovery.conf -use-kernel-args -debug\n\n", os.Args[0])
}

func validateConfig(cfg *config.Config) {
	err := config.Validate(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}
}

func deviceDiscovery(cfg *config.Config) error {
	var ctx context.Context
	var cancel context.CancelFunc

	if cfg.Debug {
		// Set a timeout when debug is true
		ctx, cancel = context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		fmt.Println("Starting device onboarding with timeout")
	} else {
		// Run without timeout if debug is false
		ctx = context.Background()
		fmt.Println("Starting device onboarding without timeout")
	}

	// Create controller configuration
	controllerCfg := mode.Config{
		ObmSvc:                 cfg.ObmSvc,
		ObsSvc:                 cfg.ObsSvc,
		ObmPort:                cfg.ObmPort,
		KeycloakURL:            cfg.KeycloakURL,
		MacAddr:                cfg.MacAddr,
		SerialNumber:           cfg.SerialNumber,
		UUID:                   cfg.UUID,
		IPAddress:              cfg.IPAddress,
		CaCertPath:             cfg.CaCertPath,
		DisableInteractiveMode: cfg.DisableInteractiveMode,
	}

	// Create and execute onboarding controller
	controller := mode.NewOnboardingController(controllerCfg)
	return controller.Execute(ctx)
}
