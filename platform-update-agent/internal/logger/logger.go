// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package logger provides a global singleton logger instance that is safe for concurrent use by multiple goroutines.
// It offers a method to retrieve the logger instance and another to set a new logger instance in a thread-safe manner.
package logger

import (
	"sync"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/info"
	"github.com/sirupsen/logrus"
)

var (
	loggerInstance *logrus.Entry
	mu             sync.Mutex
)

// Logger provides a global singleton logger instance.
func Logger() *logrus.Entry {
	mu.Lock()
	defer mu.Unlock()
	if loggerInstance == nil {
		loggerInstance = logger.New(info.Component, info.Version)
	}
	return loggerInstance
}

// SetLogger sets a new logger instance in a thread-safe manner.
func SetLogger(newLogger *logrus.Entry) {
	mu.Lock()
	defer mu.Unlock()
	loggerInstance = newLogger
}
