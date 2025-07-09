// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package logger provides centralized logging functionality for the Platform Manageability Agent
package logger

import (
	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance for the Platform Manageability Agent
var Logger = logrus.New()

func init() {
	// Set default log format and level
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	Logger.SetLevel(logrus.InfoLevel)
}
