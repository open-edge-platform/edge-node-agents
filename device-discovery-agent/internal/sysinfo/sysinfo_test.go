// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sysinfo

import (
	"context"
	"testing"
	"time"
)

func TestGetIPAddressWithRetry(t *testing.T) {
	t.Run("NonexistentMAC_RetriesAndFails", func(t *testing.T) {
		// Given: A MAC address that doesn't exist
		macAddr := "FF:FF:FF:FF:FF:FF"
		retries := 3
		sleepDuration := 100 * time.Millisecond

		// When: Trying to get IP with retry
		start := time.Now()
		ip, err := GetIPAddressWithRetry(macAddr, retries, sleepDuration)
		duration := time.Since(start)

		// Then: Should fail after retries
		if err == nil {
			t.Errorf("Expected error for nonexistent MAC, got IP: %s", ip)
		}

		// Should have waited approximately retries * sleepDuration
		expectedMinDuration := time.Duration(retries-1) * sleepDuration
		if duration < expectedMinDuration {
			t.Errorf("Expected to wait at least %v, but only waited %v", expectedMinDuration, duration)
		}
	})

	t.Run("DefaultParameters", func(t *testing.T) {
		// Given: Invalid parameters (0 or negative)
		macAddr := "FF:FF:FF:FF:FF:FF"

		// When: Using default parameters
		start := time.Now()
		_, err := GetIPAddressWithRetry(macAddr, 0, 0)
		duration := time.Since(start)

		// Then: Should use defaults (10 retries, 3 seconds each)
		if err == nil {
			t.Error("Expected error for nonexistent MAC")
		}

		// Should have used default retry count (10 retries = 9 sleeps of 3 seconds)
		// We'll check for at least 5 seconds to account for execution time
		expectedMinDuration := 5 * time.Second
		if duration < expectedMinDuration {
			t.Errorf("Expected to wait at least %v (using defaults), but only waited %v", expectedMinDuration, duration)
		}
	})
}

func TestGetIPAddressWithContext(t *testing.T) {
	t.Run("ContextCancelled_ReturnsImmediately", func(t *testing.T) {
		// Given: A context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		macAddr := "FF:FF:FF:FF:FF:FF"

		// When: Trying to get IP with canceled context
		start := time.Now()
		ip, err := GetIPAddressWithContext(ctx, macAddr, 10, 3*time.Second)
		duration := time.Since(start)

		// Then: Should return immediately with context error
		if err == nil {
			t.Errorf("Expected context error, got IP: %s", ip)
		}
		if !contains(err.Error(), "canceled") {
			t.Errorf("Expected error to mention 'canceled', got: %v", err)
		}

		// Should return quickly (within 1 second)
		if duration > 1*time.Second {
			t.Errorf("Expected immediate return, but waited %v", duration)
		}
	})

	t.Run("ContextCancelledDuringSleep", func(t *testing.T) {
		// Given: A context that will be canceled during retry
		ctx, cancel := context.WithCancel(context.Background())

		macAddr := "FF:FF:FF:FF:FF:FF"

		// Cancel after 500ms
		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		// When: Trying to get IP (would normally take 3s * 10 = 30s)
		start := time.Now()
		ip, err := GetIPAddressWithContext(ctx, macAddr, 10, 3*time.Second)
		duration := time.Since(start)

		// Then: Should cancel within reasonable time after context cancellation
		if err == nil {
			t.Errorf("Expected context error, got IP: %s", ip)
		}
		if !contains(err.Error(), "canceled") {
			t.Errorf("Expected error to mention 'canceled', got: %v", err)
		}

		// Should return shortly after cancellation (within 2 seconds of cancel time)
		if duration > 3*time.Second {
			t.Errorf("Expected early return after context cancel, but waited %v", duration)
		}
	})

	t.Run("ContextWithTimeout", func(t *testing.T) {
		// Given: A context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		macAddr := "FF:FF:FF:FF:FF:FF"

		// When: Trying to get IP (would normally take longer than timeout)
		start := time.Now()
		ip, err := GetIPAddressWithContext(ctx, macAddr, 10, 500*time.Millisecond)
		duration := time.Since(start)

		// Then: Should timeout
		if err == nil {
			t.Errorf("Expected timeout error, got IP: %s", ip)
		}

		// Should respect the context timeout (1 second + small margin)
		if duration > 2*time.Second {
			t.Errorf("Expected timeout around 1s, but waited %v", duration)
		}
	})

	t.Run("NonexistentMAC_RetriesWithContext", func(t *testing.T) {
		// Given: Valid context with nonexistent MAC
		ctx := context.Background()
		macAddr := "FF:FF:FF:FF:FF:FF"
		retries := 3
		sleepDuration := 100 * time.Millisecond

		// When: Trying to get IP
		start := time.Now()
		ip, err := GetIPAddressWithContext(ctx, macAddr, retries, sleepDuration)
		duration := time.Since(start)

		// Then: Should fail after retries
		if err == nil {
			t.Errorf("Expected error for nonexistent MAC, got IP: %s", ip)
		}

		// Should have waited for retries
		expectedMinDuration := time.Duration(retries-1) * sleepDuration
		if duration < expectedMinDuration {
			t.Errorf("Expected to wait at least %v, but only waited %v", expectedMinDuration, duration)
		}
	})
}

func TestGetIPAddress_Integration(t *testing.T) {
	t.Run("LoopbackInterface", func(t *testing.T) {
		// This test verifies that loopback IPs are skipped
		// We can't test with actual IPs as they vary by system

		// Given: Loopback MAC (this won't exist, but tests the logic)
		macAddr := "00:00:00:00:00:00"

		// When: Getting IP
		ip, err := GetIPAddress(macAddr)

		// Then: Should not find loopback
		if err == nil {
			// If we somehow got an IP, verify it's not loopback
			if ip == "127.0.0.1" || ip == "::1" {
				t.Error("Should not return loopback IP")
			}
		}
	})
}

// TestRetryLogic_Documentation serves as documentation for the retry behavior
func TestRetryLogic_Documentation(t *testing.T) {
	t.Run("DocumentedBehavior", func(t *testing.T) {
		t.Log("IP Address Retry Logic (from wait_for_ip.sh):")
		t.Log("")
		t.Log("GetIPAddressWithRetry:")
		t.Log("  - Default: 10 retries, 3 seconds between attempts")
		t.Log("  - Waits for DHCP to assign IP to interface with matching MAC")
		t.Log("  - Returns immediately if IP is already assigned")
		t.Log("  - Returns error after all retries exhausted")
		t.Log("")
		t.Log("GetIPAddressWithContext:")
		t.Log("  - Same as GetIPAddressWithRetry but supports context cancellation")
		t.Log("  - Can be canceled mid-retry via context")
		t.Log("  - Useful with timeout contexts or manual cancellation")
		t.Log("  - Returns context.Canceled or context.DeadlineExceeded errors")
		t.Log("")
		t.Log("Use cases:")
		t.Log("  - Boot-time: Wait for DHCP assignment during system startup")
		t.Log("  - Container: Wait for network initialization")
		t.Log("  - Hot-plug: Wait for new interface to get IP")
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
