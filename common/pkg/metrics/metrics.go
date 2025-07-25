// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

func newResource(name, version string) *resource.Resource {
	attributes := append(resource.Default().Attributes(), semconv.ServiceName(name),
		semconv.ServiceVersion(version))

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...)
}

func newMeterProvider(ctx context.Context, res *resource.Resource, endpoint string, interval time.Duration) (*metric.MeterProvider, error) {
	// #FIXME : https://github.com/grpc/grpc-go/issues/8207 If https_proxy
	// is set failure seen in dial to a unix domain socket. Current workaround
	// is to disable proxy usage as we know this is a unix socket.
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithDialOption(grpc.WithNoProxy()),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(interval))))

	return meterProvider, nil
}

// Init initializes the OpenTelemetry metrics collection for the agent.
func Init(ctx context.Context, endpoint string, interval time.Duration, name, version string) (func(context.Context) error, error) {
	if endpoint == "" {
		return nil, errors.New("no metrics endpoint provided, metrics will not be collected for the agent")
	}

	res := newResource(name, version)

	meterProvider, err := newMeterProvider(ctx, res, endpoint, interval)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(meterProvider)

	// collect host metrics - processCPUTime, hostCPUTime, hostMemoryUsage, hostMemoryUtilization, networkIOUsage
	err = host.Start()
	if err != nil {
		_ = meterProvider.Shutdown(ctx) //nolint:errcheck // Ignoring error as we are returning an error anyway
		return nil, err
	}

	return meterProvider.Shutdown, nil
}
