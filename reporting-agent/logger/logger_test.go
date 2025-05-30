// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestGetSingletonAndThreadSafe ensures that Get returns the same instance on multiple calls and is thread-safe.
func TestGetSingletonAndThreadSafe(t *testing.T) {
	logger1 := Get()
	logger2 := Get()
	require.NotNil(t, logger1, "Logger instance should not be nil")
	require.Same(t, logger1, logger2, "Get should return the same instance")

	var wg sync.WaitGroup
	var loggers [10]*zap.SugaredLogger
	for i := range loggers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			loggers[idx] = Get()
		}(i)
	}
	wg.Wait()
	for i := 1; i < len(loggers); i++ {
		require.Same(t, loggers[0], loggers[i], "All concurrent Get calls should return the same instance")
	}
}
