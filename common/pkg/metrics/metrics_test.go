// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
)

func TestMetricsWithInvalidEndpoint(t *testing.T) {
	ctx := t.Context()
	metricsEndpoint := "unix:///dummy"
	metricsInterval := 10 * time.Millisecond
	infoComponent := "Test Agent"
	infoVersion := "Test"

	shutdown, err := metrics.Init(ctx, metricsEndpoint, metricsInterval, infoComponent, infoVersion)
	require.NoError(t, err)
	err = shutdown(ctx)
	require.Error(t, err)
}

func TestMetricsWithValidEndpoint(t *testing.T) {
	fmt.Println("TestMetricsWithValidEndpoint()")
	ctx := t.Context()
	socketName := "/tmp/otel.sock"
	metricsEndpoint := "unix://" + socketName
	metricsInterval := 1 * time.Second
	infoComponent := "Test Agent"
	infoVersion := "Test"

	_, err := net.Listen("unix", socketName)
	require.NoError(t, err)
	defer func() {
		os.Remove(socketName)
	}()

	shutdown, err := metrics.Init(ctx, metricsEndpoint, metricsInterval, infoComponent, infoVersion)
	require.NoError(t, err)
	err = shutdown(ctx)
	// TODO mock otel collector. Change assert to NoError
	require.Error(t, err)
}

func TestMetricsWithNoEndpoint(t *testing.T) {
	metricsEndpoint := ""
	metricsInterval := 10 * time.Millisecond
	infoComponent := "Test Agent"
	infoVersion := "Test"

	shutdown, err := metrics.Init(t.Context(), metricsEndpoint, metricsInterval, infoComponent, infoVersion)
	require.Error(t, err)
	require.Nil(t, shutdown)
}
