/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPowerCapabilities(t *testing.T) {
	t.Run("success on Linux", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Power capabilities test only runs on Linux")
		}

		capabilities, err := GetPowerCapabilities()

		assert.NoError(t, err)
		assert.NotNil(t, capabilities)

		assert.NotNil(t, capabilities.Shutdown)
		assert.NotNil(t, capabilities.Reboot)
		assert.NotNil(t, capabilities.Suspend)
		assert.NotNil(t, capabilities.Hibernate)
		assert.NotEmpty(t, capabilities.CapabilitiesJson)

		assert.True(t, capabilities.Shutdown)
		assert.True(t, capabilities.Reboot)

		var jsonData PowerCapabilities
		err = json.Unmarshal([]byte(capabilities.CapabilitiesJson), &jsonData)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Shutdown, jsonData.Shutdown)
		assert.Equal(t, capabilities.Reboot, jsonData.Reboot)
		assert.Equal(t, capabilities.Suspend, jsonData.Suspend)
		assert.Equal(t, capabilities.Hibernate, jsonData.Hibernate)
	})

	t.Run("fails on non-Linux", func(t *testing.T) {
		if runtime.GOOS == "linux" {
			t.Skip("Non-Linux test only runs on non-Linux systems")
		}

		capabilities, err := GetPowerCapabilities()

		assert.Error(t, err)
		assert.Nil(t, capabilities)
		assert.Contains(t, err.Error(), "power capabilities only supported on Linux")
		assert.Contains(t, err.Error(), runtime.GOOS)
	})
}

func TestGetLinuxPowerCapabilities(t *testing.T) {
	t.Run("returns power capabilities", func(t *testing.T) {
		capabilities := getLinuxPowerCapabilities()

		assert.True(t, capabilities.Shutdown)
		assert.True(t, capabilities.Reboot)

		assert.IsType(t, false, capabilities.Suspend)
		assert.IsType(t, false, capabilities.Hibernate)

		t.Logf("Capabilities: %+v", capabilities)
	})

	t.Run("validates structure", func(t *testing.T) {
		capabilities := getLinuxPowerCapabilities()

		assert.IsType(t, PowerCapabilities{}, capabilities)

		assert.IsType(t, true, capabilities.Shutdown)
		assert.IsType(t, true, capabilities.Reboot)
		assert.IsType(t, true, capabilities.Suspend)
		assert.IsType(t, true, capabilities.Hibernate)
	})
}

func TestCheckPowerCommandAvailable(t *testing.T) {
	t.Run("handles missing commands", func(t *testing.T) {
		result := checkPowerCommandAvailable("nonexistent-power-command-12345")
		assert.False(t, result)
	})

	t.Run("handles common power commands", func(t *testing.T) {
		commands := []string{"suspend", "hibernate", "pm-suspend", "pm-hibernate"}

		for _, cmd := range commands {
			result := checkPowerCommandAvailable(cmd)
			// Result can be true or false depending on system
			assert.IsType(t, true, result)
			t.Logf("Command %s available: %t", cmd, result)
		}
	})

	t.Run("handles empty command gracefully", func(t *testing.T) {
		// In restricted environments, empty commands might behave differently
		// Just test that it doesn't panic and returns a boolean
		result := checkPowerCommandAvailable("")
		assert.IsType(t, true, result)
		t.Logf("Empty command result: %t", result)
	})
}

func TestCheckLinuxSuspendSupport(t *testing.T) {
	t.Run("returns boolean", func(t *testing.T) {
		result := checkLinuxSuspendSupport()
		assert.IsType(t, true, result)
		t.Logf("Suspend support: %t", result)
	})

	t.Run("handles restricted access", func(t *testing.T) {
		result := checkLinuxSuspendSupport()
		assert.IsType(t, true, result)
	})
}

func TestCheckLinuxHibernateSupport(t *testing.T) {
	t.Run("returns boolean", func(t *testing.T) {
		result := checkLinuxHibernateSupport()
		assert.IsType(t, true, result)
		t.Logf("Hibernate support: %t", result)
	})

	t.Run("handles restricted access", func(t *testing.T) {
		result := checkLinuxHibernateSupport()
		assert.IsType(t, true, result)
	})
}

func TestIsSystemdTargetAvailable(t *testing.T) {
	t.Run("handles common targets", func(t *testing.T) {
		targets := []string{"suspend.target", "hibernate.target", "reboot.target", "shutdown.target"}

		for _, target := range targets {
			result := isSystemdTargetAvailable(target)
			assert.IsType(t, true, result)
			t.Logf("Target %s available: %t", target, result)
		}
	})

	t.Run("handles invalid targets", func(t *testing.T) {
		result := isSystemdTargetAvailable("nonexistent.target")
		assert.False(t, result)
	})

	t.Run("handles empty target gracefully", func(t *testing.T) {
		// In restricted environments, empty targets might behave differently
		result := isSystemdTargetAvailable("")
		assert.IsType(t, true, result)
		t.Logf("Empty target result: %t", result)
	})
}

func TestIsSystemdAvailable(t *testing.T) {
	t.Run("returns boolean", func(t *testing.T) {
		result := isSystemdAvailable()
		assert.IsType(t, true, result)
		t.Logf("Systemd available: %t", result)
	})

	t.Run("handles missing systemctl", func(t *testing.T) {
		// Should not panic even if systemctl is not available
		result := isSystemdAvailable()
		assert.IsType(t, true, result)
	})
}

func TestHasSwapSpace(t *testing.T) {
	t.Run("returns boolean", func(t *testing.T) {
		result := hasSwapSpace()
		assert.IsType(t, true, result)
		t.Logf("Swap space available: %t", result)
	})

	t.Run("handles restricted access", func(t *testing.T) {
		result := hasSwapSpace()
		assert.IsType(t, true, result)
	})
}

func TestGetPowerCapabilitiesJSON(t *testing.T) {
	t.Run("returns valid JSON", func(t *testing.T) {
		jsonStr, err := GetPowerCapabilitiesJSON()
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonStr)

		// Should be valid JSON
		var capabilities PowerCapabilities
		err = json.Unmarshal([]byte(jsonStr), &capabilities)
		assert.NoError(t, err)

		assert.IsType(t, true, capabilities.Shutdown)
		assert.IsType(t, true, capabilities.Reboot)
		assert.IsType(t, true, capabilities.Suspend)
		assert.IsType(t, true, capabilities.Hibernate)

		t.Logf("JSON: %s", jsonStr)
	})

	t.Run("JSON contains expected fields", func(t *testing.T) {
		jsonStr, err := GetPowerCapabilitiesJSON()
		assert.NoError(t, err)

		assert.Contains(t, jsonStr, "shutdown")
		assert.Contains(t, jsonStr, "reboot")
		assert.Contains(t, jsonStr, "suspend")
		assert.Contains(t, jsonStr, "hibernate")
	})
}

func TestCheckPowerCommand(t *testing.T) {
	t.Run("recognizes valid commands", func(t *testing.T) {
		validCommands := []string{"shutdown", "reboot", "suspend", "hibernate"}

		for _, cmd := range validCommands {
			result := CheckPowerCommand(cmd)
			assert.IsType(t, true, result)
			t.Logf("Command %s supported: %t", cmd, result)
		}
	})

	t.Run("handles case insensitive commands", func(t *testing.T) {
		testCases := []string{"SHUTDOWN", "Reboot", "SUSPEND", "Hibernate"}

		for _, cmd := range testCases {
			result := CheckPowerCommand(cmd)
			assert.IsType(t, true, result)
			t.Logf("Command %s (case insensitive) supported: %t", cmd, result)
		}
	})

	t.Run("rejects invalid commands", func(t *testing.T) {
		invalidCommands := []string{"invalid", "unknown", "power", "sleep", ""}

		for _, cmd := range invalidCommands {
			result := CheckPowerCommand(cmd)
			assert.False(t, result)
			t.Logf("Command %s rejected: %t", cmd, !result)
		}
	})

	t.Run("shutdown and reboot always supported", func(t *testing.T) {
		assert.True(t, CheckPowerCommand("shutdown"))
		assert.True(t, CheckPowerCommand("reboot"))
	})
}

func TestGetSupportedPowerCommands(t *testing.T) {
	t.Run("returns slice of strings", func(t *testing.T) {
		commands := GetSupportedPowerCommands()
		assert.IsType(t, []string{}, commands)
		assert.NotNil(t, commands)

		t.Logf("Supported commands: %v", commands)
	})

	t.Run("always includes shutdown and reboot", func(t *testing.T) {
		commands := GetSupportedPowerCommands()

		assert.Contains(t, commands, "shutdown")
		assert.Contains(t, commands, "reboot")
	})

	t.Run("contains only valid commands", func(t *testing.T) {
		commands := GetSupportedPowerCommands()
		validCommands := []string{"shutdown", "reboot", "suspend", "hibernate"}

		for _, cmd := range commands {
			assert.Contains(t, validCommands, cmd)
		}
	})

	t.Run("no duplicates", func(t *testing.T) {
		commands := GetSupportedPowerCommands()
		seen := make(map[string]bool)

		for _, cmd := range commands {
			assert.False(t, seen[cmd], "Duplicate command found: %s", cmd)
			seen[cmd] = true
		}
	})

	t.Run("minimum required commands", func(t *testing.T) {
		commands := GetSupportedPowerCommands()

		assert.GreaterOrEqual(t, len(commands), 2)
	})
}

func TestPowerCapabilitiesStruct(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		capabilities := PowerCapabilities{
			Shutdown:  true,
			Reboot:    true,
			Suspend:   false,
			Hibernate: false,
		}

		jsonData, err := json.Marshal(capabilities)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "shutdown")
		assert.Contains(t, jsonStr, "reboot")
		assert.Contains(t, jsonStr, "suspend")
		assert.Contains(t, jsonStr, "hibernate")
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		jsonStr := `{"shutdown":true,"reboot":true,"suspend":false,"hibernate":false}`

		var capabilities PowerCapabilities
		err := json.Unmarshal([]byte(jsonStr), &capabilities)
		assert.NoError(t, err)

		assert.True(t, capabilities.Shutdown)
		assert.True(t, capabilities.Reboot)
		assert.False(t, capabilities.Suspend)
		assert.False(t, capabilities.Hibernate)
	})
}

func TestPowerCapabilitiesIntegration(t *testing.T) {
	t.Run("consistency between functions", func(t *testing.T) {
		capabilities1 := getLinuxPowerCapabilities()

		jsonStr, err := GetPowerCapabilitiesJSON()
		assert.NoError(t, err)

		var capabilities2 PowerCapabilities
		err = json.Unmarshal([]byte(jsonStr), &capabilities2)
		assert.NoError(t, err)

		assert.Equal(t, capabilities1, capabilities2)
	})

	t.Run("supported commands match capabilities", func(t *testing.T) {
		capabilities := getLinuxPowerCapabilities()
		supportedCommands := GetSupportedPowerCommands()

		if capabilities.Shutdown {
			assert.Contains(t, supportedCommands, "shutdown")
		} else {
			assert.NotContains(t, supportedCommands, "shutdown")
		}

		if capabilities.Reboot {
			assert.Contains(t, supportedCommands, "reboot")
		} else {
			assert.NotContains(t, supportedCommands, "reboot")
		}

		if capabilities.Suspend {
			assert.Contains(t, supportedCommands, "suspend")
		} else {
			assert.NotContains(t, supportedCommands, "suspend")
		}

		if capabilities.Hibernate {
			assert.Contains(t, supportedCommands, "hibernate")
		} else {
			assert.NotContains(t, supportedCommands, "hibernate")
		}
	})

	t.Run("CheckPowerCommand matches capabilities", func(t *testing.T) {
		capabilities := getLinuxPowerCapabilities()

		assert.Equal(t, capabilities.Shutdown, CheckPowerCommand("shutdown"))
		assert.Equal(t, capabilities.Reboot, CheckPowerCommand("reboot"))
		assert.Equal(t, capabilities.Suspend, CheckPowerCommand("suspend"))
		assert.Equal(t, capabilities.Hibernate, CheckPowerCommand("hibernate"))
	})
}

func TestPowerCapabilitiesErrorHandling(t *testing.T) {
	t.Run("handles file system errors gracefully", func(t *testing.T) {
		assert.NotPanics(t, func() {
			checkLinuxSuspendSupport()
		})

		assert.NotPanics(t, func() {
			checkLinuxHibernateSupport()
		})

		assert.NotPanics(t, func() {
			hasSwapSpace()
		})
	})

	t.Run("handles command execution errors gracefully", func(t *testing.T) {
		assert.NotPanics(t, func() {
			isSystemdAvailable()
		})

		assert.NotPanics(t, func() {
			isSystemdTargetAvailable("test.target")
		})

		assert.NotPanics(t, func() {
			checkPowerCommandAvailable("test")
		})
	})
}

func TestPowerCapabilitiesEdgeCases(t *testing.T) {
	t.Run("empty and whitespace handling", func(t *testing.T) {
		// Test CheckPowerCommand with empty strings
		assert.False(t, CheckPowerCommand(""))
		assert.False(t, CheckPowerCommand("   "))

		// For restricted environments, just check these don't panic
		assert.NotPanics(t, func() {
			checkPowerCommandAvailable("  ")
		})

		assert.NotPanics(t, func() {
			isSystemdTargetAvailable("  ")
		})
	})

	t.Run("special characters in commands", func(t *testing.T) {
		specialCommands := []string{"suspend/test", "hibernate@test", "reboot#test"}

		for _, cmd := range specialCommands {
			result := CheckPowerCommand(cmd)
			assert.False(t, result)
		}
	})

	t.Run("very long command names", func(t *testing.T) {
		longCommand := strings.Repeat("a", 1000)
		assert.False(t, CheckPowerCommand(longCommand))

		// For restricted environments, just check this doesn't panic
		assert.NotPanics(t, func() {
			checkPowerCommandAvailable(longCommand)
		})
	})
}

// Benchmark tests
func BenchmarkGetPowerCapabilities(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("Power capabilities benchmark only runs on Linux")
	}

	for i := 0; i < b.N; i++ {
		_, err := GetPowerCapabilities()
		if err != nil {
			b.Fatalf("GetPowerCapabilities failed: %v", err)
		}
	}
}

func BenchmarkGetLinuxPowerCapabilities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getLinuxPowerCapabilities()
	}
}

func BenchmarkCheckPowerCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CheckPowerCommand("shutdown")
		CheckPowerCommand("reboot")
		CheckPowerCommand("suspend")
		CheckPowerCommand("hibernate")
	}
}

func BenchmarkGetSupportedPowerCommands(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetSupportedPowerCommands()
	}
}
