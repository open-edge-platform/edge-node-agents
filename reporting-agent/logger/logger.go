// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once     sync.Once
	instance *zap.SugaredLogger
)

// Get returns the singleton SugaredLogger instance, initializing it if necessary.
func Get() *zap.SugaredLogger {
	once.Do(func() {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.StacktraceKey = ""                      // disable stacktrace key
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // time in ISO8601 format (e.g. "2006-01-02T15:04:05.000Z0700")
		core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), os.Stdout, zapcore.InfoLevel)
		logger := zap.New(core)
		instance = logger.Sugar()
	})
	return instance
}
