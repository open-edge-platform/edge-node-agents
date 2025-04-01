// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"

	"github.com/stretchr/testify/assert"
)

func TestMetricsWithInvalidEndpoint(t *testing.T) {
	ctx := context.Background()
	MetricsEndpoint := "unix:///dummy"
	MetricsInterval := 10 * time.Millisecond
	infoComponent := "Test Agent"
	infoVersion := "Test"

	shutdown, err := metrics.Init(ctx, MetricsEndpoint, MetricsInterval, infoComponent, infoVersion)
	assert.NoError(t, err)
	err = shutdown(ctx)
	assert.Error(t, err)
}

func TestMetricsWithValidEndpoint(t *testing.T) {
	fmt.Println("TestMetricsWithValidEndpoint()")
	ctx := context.Background()
	socketName := "/tmp/otel.sock"
	MetricsEndpoint := "unix://" + socketName
	MetricsInterval := 1 * time.Second
	infoComponent := "Test Agent"
	infoVersion := "Test"

	_, err := net.Listen("unix", socketName)
	assert.NoError(t, err)
	defer func() {
		os.Remove(socketName)
	}()

	shutdown, err := metrics.Init(ctx, MetricsEndpoint, MetricsInterval, infoComponent, infoVersion)
	assert.NoError(t, err)
	err = shutdown(ctx)
	// TODO mock otel collector. Change assert to NoError
	assert.Error(t, err)
}

func TestMetricsWithNoEndpoint(t *testing.T) {
	ctx := context.Background()
	MetricsEndpoint := ""
	MetricsInterval := 10 * time.Millisecond
	infoComponent := "Test Agent"
	infoVersion := "Test"

	shutdown, err := metrics.Init(ctx, MetricsEndpoint, MetricsInterval, infoComponent, infoVersion)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
}
