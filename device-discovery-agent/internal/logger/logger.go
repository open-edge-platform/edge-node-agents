// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance
var Logger = logrus.New()

// InitLogger initializes the logger with the specified debug mode
func InitLogger(debug bool) {
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if debug {
		Logger.SetLevel(logrus.DebugLevel)
		Logger.Debug("Debug mode enabled")
	} else {
		Logger.SetLevel(logrus.InfoLevel)
	}
}
