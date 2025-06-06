// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// New creates a new log entry with the specified component and version.
func New(component, version string) *log.Entry {
	return log.WithFields(log.Fields{
		"component": component,
		"version":   version,
	})
}
