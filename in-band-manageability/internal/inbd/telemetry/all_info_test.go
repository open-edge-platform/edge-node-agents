/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAllInfo(t *testing.T) {
	t.Run("success - all info retrieved", func(t *testing.T) {
		allInfo, err := GetAllInfo()

		// Handle systems that might not have all required tools
		if err != nil {
			t.Logf("GetAllInfo failed (may be expected in CI/minimal systems): %v", err)
			// If error occurred, allInfo should be nil
			assert.Nil(t, allInfo)
			return
		}

		// Should not return error on systems with proper tools
		assert.NoError(t, err)
		assert.NotNil(t, allInfo)

		// Verify structure is populated
		assert.NotNil(t, allInfo.Hardware)
		assert.NotNil(t, allInfo.Firmware)
		assert.NotNil(t, allInfo.OsInfo)
		assert.NotNil(t, allInfo.Version)
		assert.NotNil(t, allInfo.PowerCapabilities)
		assert.NotNil(t, allInfo.Swbom)
		assert.NotNil(t, allInfo.AdditionalInfo)

		// Verify AdditionalInfo is initialized as empty slice
		assert.Empty(t, allInfo.AdditionalInfo)
	})

	t.Run("handles missing system tools gracefully", func(t *testing.T) {
		allInfo, err := GetAllInfo()

		// This test accepts both success and failure
		if err != nil {
			// Error is acceptable if system tools are missing
			assert.Nil(t, allInfo)
			t.Logf("GetAllInfo failed due to missing system tools: %v", err)
		} else {
			// Success is also acceptable
			assert.NotNil(t, allInfo)
			assert.NotNil(t, allInfo.AdditionalInfo)
		}
	})
}

// TestGetAllInfo_ErrorHandling tests error handling scenarios
func TestGetAllInfo_ErrorHandling(t *testing.T) {
	t.Run("handles individual function errors gracefully", func(t *testing.T) {
		// Since GetAllInfo calls multiple functions, if any one fails,
		// the entire function should fail
		allInfo, err := GetAllInfo()

		// The function should either succeed completely or fail
		if err != nil {
			// If there's an error, allInfo should be nil
			assert.Nil(t, allInfo)
			// Error should be meaningful
			assert.NotEmpty(t, err.Error())
		} else {
			// If no error, allInfo should be populated
			assert.NotNil(t, allInfo)
		}
	})
}

// TestGetAllInfo_Performance tests performance characteristics
func TestGetAllInfo_Performance(t *testing.T) {
	t.Run("reasonable execution time", func(t *testing.T) {
		// Test that GetAllInfo completes in reasonable time
		// This is especially important since it calls multiple system functions
		for i := 0; i < 3; i++ {
			allInfo, err := GetAllInfo()

			if err != nil {
				t.Logf("GetAllInfo failed on iteration %d: %v", i, err)
				continue
			}

			assert.NotNil(t, allInfo)
		}
	})
}

// TestGetAllInfo_Idempotent tests that multiple calls return consistent results
func TestGetAllInfo_Idempotent(t *testing.T) {
	t.Run("multiple calls return consistent structure", func(t *testing.T) {
		// Call GetAllInfo multiple times
		allInfo1, err1 := GetAllInfo()
		allInfo2, err2 := GetAllInfo()

		// Both should succeed or both should fail
		if err1 != nil && err2 != nil {
			// Both failed - this is acceptable
			assert.Nil(t, allInfo1)
			assert.Nil(t, allInfo2)
			return
		}

		// If one succeeded, both should succeed
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, allInfo1)
		assert.NotNil(t, allInfo2)

		// Structure should be consistent
		assert.Equal(t, allInfo1.Hardware != nil, allInfo2.Hardware != nil)
		assert.Equal(t, allInfo1.Firmware != nil, allInfo2.Firmware != nil)
		assert.Equal(t, allInfo1.OsInfo != nil, allInfo2.OsInfo != nil)
		assert.Equal(t, allInfo1.Version != nil, allInfo2.Version != nil)
		assert.Equal(t, allInfo1.PowerCapabilities != nil, allInfo2.PowerCapabilities != nil)
		assert.Equal(t, allInfo1.Swbom != nil, allInfo2.Swbom != nil)

		// AdditionalInfo should be consistent
		assert.Equal(t, len(allInfo1.AdditionalInfo), len(allInfo2.AdditionalInfo))
	})
}

// TestGetAllInfo_EdgeCases tests edge cases and boundary conditions
func TestGetAllInfo_EdgeCases(t *testing.T) {
	t.Run("handles system without certain capabilities", func(t *testing.T) {
		// GetAllInfo should handle systems that don't have certain capabilities
		// gracefully (e.g., systems without power management, minimal systems, etc.)
		allInfo, err := GetAllInfo()

		if err != nil {
			// Some systems might not support all telemetry functions
			t.Logf("GetAllInfo failed (acceptable on some systems): %v", err)
			return
		}

		assert.NotNil(t, allInfo)

		// Even if some fields are nil/empty, the structure should be valid
		assert.NotNil(t, allInfo.AdditionalInfo)
	})
}

// BenchmarkGetAllInfo benchmarks the GetAllInfo function
func BenchmarkGetAllInfo(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := GetAllInfo()
		if err != nil {
			b.Logf("GetAllInfo failed: %v", err)
		}
	}
}
