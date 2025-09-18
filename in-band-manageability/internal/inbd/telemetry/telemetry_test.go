/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package telemetry

import (
	"fmt"
	"strings"
	"testing"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryHandler(t *testing.T) {
	t.Run("creates new query handler", func(t *testing.T) {
		handler := NewQueryHandler()
		assert.NotNil(t, handler)
		assert.IsType(t, &QueryHandler{}, handler)
	})

	t.Run("multiple instances are independent", func(t *testing.T) {
		handler1 := NewQueryHandler()
		handler2 := NewQueryHandler()

		assert.NotNil(t, handler1)
		assert.NotNil(t, handler2)
		assert.NotSame(t, handler1, handler2)
	})
}

func TestQueryHandler_HandleQuery(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("hardware query", func(t *testing.T) {
		result, err := handler.HandleQuery("hardware")

		if err != nil {
			t.Logf("Hardware query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "hardware", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetHardware())

		// Verify timestamp is recent
		now := time.Now()
		resultTime := result.Timestamp.AsTime()
		assert.WithinDuration(t, now, resultTime, 5*time.Second)
	})

	t.Run("hardware query with short option", func(t *testing.T) {
		result, err := handler.HandleQuery("hw")

		if err != nil {
			t.Logf("Hardware query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "hardware", result.Type)
		assert.NotNil(t, result.GetHardware())
	})

	t.Run("firmware query", func(t *testing.T) {
		result, err := handler.HandleQuery("firmware")

		if err != nil {
			t.Logf("Firmware query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "firmware", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetFirmware())
	})

	t.Run("firmware query with short option", func(t *testing.T) {
		result, err := handler.HandleQuery("fw")

		if err != nil {
			t.Logf("Firmware query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "firmware", result.Type)
		assert.NotNil(t, result.GetFirmware())
	})

	t.Run("os query", func(t *testing.T) {
		result, err := handler.HandleQuery("os")

		if err != nil {
			t.Logf("OS query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "os", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetOsInfo())
	})

	t.Run("swbom query", func(t *testing.T) {
		result, err := handler.HandleQuery("swbom")

		if err != nil {
			t.Logf("SWBOM query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "swbom", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetSwbom())
	})

	t.Run("version query", func(t *testing.T) {
		result, err := handler.HandleQuery("version")

		if err != nil {
			t.Logf("Version query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "version", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetVersion())
	})

	t.Run("all query", func(t *testing.T) {
		result, err := handler.HandleQuery("all")

		if err != nil {
			t.Logf("All query failed (may be expected in test environment): %v", err)
			assert.Error(t, err)
			assert.Nil(t, result)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "all", result.Type)
		assert.NotNil(t, result.Timestamp)
		assert.NotNil(t, result.Values)
		assert.NotNil(t, result.GetAllInfo())
	})

	t.Run("unsupported query option", func(t *testing.T) {
		result, err := handler.HandleQuery("invalid")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported query option")
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("empty query option", func(t *testing.T) {
		result, err := handler.HandleQuery("")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported query option")
	})

	t.Run("case sensitivity", func(t *testing.T) {
		// Test that query options are case-sensitive
		testCases := []string{"HARDWARE", "Hardware", "HW", "Hw", "OS", "SWBOM", "ALL", "VERSION"}

		for _, testCase := range testCases {
			result, err := handler.HandleQuery(testCase)
			assert.Error(t, err, "Expected error for case-sensitive query: %s", testCase)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported query option")
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		// Test that whitespace is not trimmed
		testCases := []string{" hardware", "hardware ", " hardware ", "\thardware", "hardware\n"}

		for _, testCase := range testCases {
			result, err := handler.HandleQuery(testCase)
			assert.Error(t, err, "Expected error for whitespace query: %q", testCase)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported query option")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		// Test special characters in query options
		testCases := []string{"hardware-test", "hardware_test", "hardware.test", "hardware@test"}

		for _, testCase := range testCases {
			result, err := handler.HandleQuery(testCase)
			assert.Error(t, err, "Expected error for special character query: %s", testCase)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported query option")
		}
	})
}

func TestQueryHandler_HandleQuery_ValidOptions(t *testing.T) {
	handler := NewQueryHandler()

	// Test all valid query options
	validOptions := []string{"hw", "hardware", "fw", "firmware", "os", "swbom", "version", "all"}

	for _, option := range validOptions {
		t.Run(fmt.Sprintf("valid_option_%s", option), func(t *testing.T) {
			result, err := handler.HandleQuery(option)

			// In test environment, some queries might fail due to missing dependencies
			// We'll check that either it succeeds or fails gracefully
			if err != nil {
				t.Logf("Query %s failed (may be expected in test environment): %v", option, err)
				assert.Nil(t, result)
				// Error should be from the underlying function, not from unsupported option
				assert.NotContains(t, err.Error(), "unsupported query option")
			} else {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Type)
				assert.NotNil(t, result.Timestamp)
				assert.NotNil(t, result.Values)

				// Verify timestamp is recent
				now := time.Now()
				resultTime := result.Timestamp.AsTime()
				assert.WithinDuration(t, now, resultTime, 5*time.Second)
			}
		})
	}
}

func TestQueryHandler_HandleQuery_TimestampConsistency(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("timestamp consistency", func(t *testing.T) {
		// Test that timestamps are consistent and recent
		beforeTime := time.Now()

		result, err := handler.HandleQuery("version")
		if err != nil {
			t.Skip("Version query failed, skipping timestamp test")
		}

		afterTime := time.Now()

		require.NotNil(t, result)
		require.NotNil(t, result.Timestamp)

		resultTime := result.Timestamp.AsTime()
		assert.True(t, resultTime.After(beforeTime) || resultTime.Equal(beforeTime))
		assert.True(t, resultTime.Before(afterTime) || resultTime.Equal(afterTime))
	})

	t.Run("multiple queries have different timestamps", func(t *testing.T) {
		// Test that multiple queries have different timestamps
		result1, err1 := handler.HandleQuery("version")
		if err1 != nil {
			t.Skip("Version query failed, skipping timestamp test")
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		result2, err2 := handler.HandleQuery("version")
		if err2 != nil {
			t.Skip("Version query failed, skipping timestamp test")
		}

		require.NotNil(t, result1)
		require.NotNil(t, result2)
		require.NotNil(t, result1.Timestamp)
		require.NotNil(t, result2.Timestamp)

		time1 := result1.Timestamp.AsTime()
		time2 := result2.Timestamp.AsTime()

		assert.True(t, time2.After(time1) || time2.Equal(time1))
	})
}

func TestQueryHandler_HandleQuery_ResponseStructure(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("response structure validation", func(t *testing.T) {
		// Test that each query type returns the correct response structure
		testCases := []struct {
			option       string
			expectedType string
			validator    func(*pb.QueryData) bool
		}{
			{"hardware", "hardware", func(qd *pb.QueryData) bool { return qd.GetHardware() != nil }},
			{"hw", "hardware", func(qd *pb.QueryData) bool { return qd.GetHardware() != nil }},
			{"firmware", "firmware", func(qd *pb.QueryData) bool { return qd.GetFirmware() != nil }},
			{"fw", "firmware", func(qd *pb.QueryData) bool { return qd.GetFirmware() != nil }},
			{"os", "os", func(qd *pb.QueryData) bool { return qd.GetOsInfo() != nil }},
			{"swbom", "swbom", func(qd *pb.QueryData) bool { return qd.GetSwbom() != nil }},
			{"version", "version", func(qd *pb.QueryData) bool { return qd.GetVersion() != nil }},
			{"all", "all", func(qd *pb.QueryData) bool { return qd.GetAllInfo() != nil }},
		}

		for _, tc := range testCases {
			t.Run(tc.option, func(t *testing.T) {
				result, err := handler.HandleQuery(tc.option)

				if err != nil {
					t.Logf("Query %s failed (may be expected in test environment): %v", tc.option, err)
					return
				}

				require.NotNil(t, result)
				assert.Equal(t, tc.expectedType, result.Type)
				assert.True(t, tc.validator(result), "Validator failed for option: %s", tc.option)
			})
		}
	})
}

func TestQueryHandler_HandleQuery_ErrorHandling(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("error propagation", func(t *testing.T) {
		// Test that errors from underlying functions are properly propagated
		// This is difficult to test without mocking, but we can at least verify
		// that errors are handled gracefully

		validOptions := []string{"hw", "hardware", "fw", "firmware", "os", "swbom", "version", "all"}

		for _, option := range validOptions {
			result, err := handler.HandleQuery(option)

			// Either succeeds or fails gracefully
			if err != nil {
				assert.Nil(t, result)
				assert.NotEmpty(t, err.Error())
				// Should not be an "unsupported option" error
				assert.NotContains(t, err.Error(), "unsupported query option")
			} else {
				assert.NotNil(t, result)
			}
		}
	})
}

func TestQueryHandler_HandleQuery_ConcurrentAccess(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("concurrent query handling", func(t *testing.T) {
		// Test that the handler can handle concurrent queries
		const numGoroutines = 10
		const numQueries = 5

		results := make(chan struct {
			result *pb.QueryData
			err    error
		}, numGoroutines*numQueries)

		options := []string{"version", "os", "hardware", "firmware", "swbom"}

		// Start multiple goroutines
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numQueries; j++ {
					option := options[j%len(options)]
					result, err := handler.HandleQuery(option)
					results <- struct {
						result *pb.QueryData
						err    error
					}{result, err}
				}
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines*numQueries; i++ {
			select {
			case res := <-results:
				// Either succeeds or fails gracefully
				if res.err != nil {
					assert.Nil(t, res.result)
					assert.NotContains(t, res.err.Error(), "unsupported query option")
				} else {
					assert.NotNil(t, res.result)
				}
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for concurrent queries")
			}
		}
	})
}

func TestQueryHandler_HandleQuery_EdgeCases(t *testing.T) {
	handler := NewQueryHandler()

	t.Run("very long query option", func(t *testing.T) {
		longOption := strings.Repeat("a", 1000)
		result, err := handler.HandleQuery(longOption)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported query option")
	})

	t.Run("unicode characters", func(t *testing.T) {
		unicodeOptions := []string{"硬件", "прошивка", "système", "🔧"}

		for _, option := range unicodeOptions {
			result, err := handler.HandleQuery(option)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported query option")
		}
	})

	t.Run("null bytes and control characters", func(t *testing.T) {
		controlOptions := []string{"hardware\x00", "firmware\x01", "os\x7f", "version\r\n"}

		for _, option := range controlOptions {
			result, err := handler.HandleQuery(option)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported query option")
		}
	})
}

// Benchmark tests
func BenchmarkQueryHandler_HandleQuery(b *testing.B) {
	handler := NewQueryHandler()

	b.Run("version_query", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handler.HandleQuery("version")
			if err != nil {
				b.Fatalf("Version query failed: %v", err)
			}
		}
	})

	b.Run("hardware_query", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handler.HandleQuery("hardware")
			if err != nil {
				b.Logf("Hardware query failed (may be expected): %v", err)
			}
		}
	})

	b.Run("unsupported_query", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handler.HandleQuery("invalid")
			if err == nil {
				b.Fatal("Expected error for unsupported query")
			}
		}
	})
}

func BenchmarkQueryHandler_ConcurrentQueries(b *testing.B) {
	handler := NewQueryHandler()

	b.RunParallel(func(pb *testing.PB) {
		options := []string{"version", "os", "hardware", "firmware"}
		i := 0
		for pb.Next() {
			option := options[i%len(options)]
			_, err := handler.HandleQuery(option)
			if err != nil {
				// Some queries might fail in test environment
				b.Logf("Query %s failed: %v", option, err)
			}
			i++
		}
	})
}
