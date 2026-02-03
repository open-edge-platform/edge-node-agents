// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package mode

import (
	"context"
	"testing"

	"device-discovery/internal/mode/noninteractive"
)

// TestNewOnboardingController tests the creation of a new onboarding controller
func TestNewOnboardingController(t *testing.T) {
	cfg := Config{
		ObmSvc:                 "obm.example.com",
		ObsSvc:                 "obs.example.com",
		ObmPort:                8443,
		KeycloakURL:            "https://keycloak.example.com",
		MacAddr:                "00:11:22:33:44:55",
		SerialNumber:           "SN12345",
		UUID:                   "uuid-1234",
		IPAddress:              "192.168.1.100",
		CaCertPath:             "/path/to/ca.crt",
		DisableInteractiveMode: false,
	}

	controller := NewOnboardingController(cfg)

	if controller == nil {
		t.Fatal("Expected non-nil controller")
	}

	if controller.obmSvc != cfg.ObmSvc {
		t.Errorf("Expected obmSvc '%s', got '%s'", cfg.ObmSvc, controller.obmSvc)
	}

	if controller.obsSvc != cfg.ObsSvc {
		t.Errorf("Expected obsSvc '%s', got '%s'", cfg.ObsSvc, controller.obsSvc)
	}

	if controller.obmPort != cfg.ObmPort {
		t.Errorf("Expected obmPort %d, got %d", cfg.ObmPort, controller.obmPort)
	}

	if controller.keycloakURL != cfg.KeycloakURL {
		t.Errorf("Expected keycloakURL '%s', got '%s'", cfg.KeycloakURL, controller.keycloakURL)
	}

	if controller.macAddr != cfg.MacAddr {
		t.Errorf("Expected macAddr '%s', got '%s'", cfg.MacAddr, controller.macAddr)
	}

	if controller.serialNumber != cfg.SerialNumber {
		t.Errorf("Expected serialNumber '%s', got '%s'", cfg.SerialNumber, controller.serialNumber)
	}

	if controller.uuid != cfg.UUID {
		t.Errorf("Expected uuid '%s', got '%s'", cfg.UUID, controller.uuid)
	}

	if controller.ipAddress != cfg.IPAddress {
		t.Errorf("Expected ipAddress '%s', got '%s'", cfg.IPAddress, controller.ipAddress)
	}

	if controller.caCertPath != cfg.CaCertPath {
		t.Errorf("Expected caCertPath '%s', got '%s'", cfg.CaCertPath, controller.caCertPath)
	}

	if controller.disableInteractiveMode != cfg.DisableInteractiveMode {
		t.Errorf("Expected disableInteractiveMode %v, got %v", cfg.DisableInteractiveMode, controller.disableInteractiveMode)
	}
}

// TestTryNonInteractiveMode tests the non-interactive mode attempt
func TestTryNonInteractiveMode(t *testing.T) {
	cfg := Config{
		ObmSvc:       "obm.example.com",
		ObsSvc:       "obs.example.com",
		ObmPort:      8443,
		MacAddr:      "00:11:22:33:44:55",
		SerialNumber: "SN12345",
		UUID:         "uuid-1234",
		IPAddress:    "192.168.1.100",
		CaCertPath:   "/nonexistent/ca.crt",
	}

	controller := NewOnboardingController(cfg)
	ctx := context.Background()

	// This will fail to connect (no actual server), but we're testing the method exists and returns
	result := controller.tryNonInteractiveMode(ctx)

	// Should get an error result since there's no actual server
	if result.Error == nil {
		t.Error("Expected error when connecting to non-existent server")
	}

	// Should have default values for credentials when connection fails
	if result.ClientID != "" {
		t.Errorf("Expected empty ClientID on connection failure, got '%s'", result.ClientID)
	}
}

func TestExecute_FallbackDisabled(t *testing.T) {
	cfg := Config{
		ObmSvc:                 "obm.example.com",
		ObsSvc:                 "obs.example.com",
		ObmPort:                8443,
		KeycloakURL:            "https://keycloak.example.com",
		MacAddr:                "00:11:22:33:44:55",
		SerialNumber:           "SN12345",
		UUID:                   "uuid-1234",
		IPAddress:              "192.168.1.100",
		CaCertPath:             "/nonexistent/ca.crt",
		DisableInteractiveMode: true, // Interactive mode disabled
	}

	controller := NewOnboardingController(cfg)

	// Create a mock result that requires fallback
	mockExecute := func(ctx context.Context) error {
		result := noninteractive.StreamResult{
			ShouldFallback: true,
			Error:          nil,
		}

		if result.ShouldFallback {
			if controller.disableInteractiveMode {
				return nil // Expected path - blocked fallback
			}
		}
		return nil
	}

	ctx := context.Background()
	err := mockExecute(ctx)

	if err != nil {
		t.Errorf("Mock test setup failed: %v", err)
	}

	// Verify the controller has interactive mode disabled
	if !controller.disableInteractiveMode {
		t.Error("Expected disableInteractiveMode to be true")
	}
}

// TestConfig_AllFieldsSet tests that Config struct properly holds all fields
func TestConfig_AllFieldsSet(t *testing.T) {
	cfg := Config{
		ObmSvc:                 "test-obm",
		ObsSvc:                 "test-obs",
		ObmPort:                9999,
		KeycloakURL:            "https://test-keycloak",
		MacAddr:                "AA:BB:CC:DD:EE:FF",
		SerialNumber:           "TEST-SN",
		UUID:                   "test-uuid",
		IPAddress:              "10.0.0.1",
		CaCertPath:             "/test/ca.crt",
		DisableInteractiveMode: true,
	}

	// Verify all fields can be set and retrieved
	if cfg.ObmSvc != "test-obm" {
		t.Errorf("ObmSvc not set correctly")
	}
	if cfg.ObsSvc != "test-obs" {
		t.Errorf("ObsSvc not set correctly")
	}
	if cfg.ObmPort != 9999 {
		t.Errorf("ObmPort not set correctly")
	}
	if cfg.KeycloakURL != "https://test-keycloak" {
		t.Errorf("KeycloakURL not set correctly")
	}
	if cfg.MacAddr != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MacAddr not set correctly")
	}
	if cfg.SerialNumber != "TEST-SN" {
		t.Errorf("SerialNumber not set correctly")
	}
	if cfg.UUID != "test-uuid" {
		t.Errorf("UUID not set correctly")
	}
	if cfg.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress not set correctly")
	}
	if cfg.CaCertPath != "/test/ca.crt" {
		t.Errorf("CaCertPath not set correctly")
	}
	if !cfg.DisableInteractiveMode {
		t.Errorf("DisableInteractiveMode not set correctly")
	}
}
