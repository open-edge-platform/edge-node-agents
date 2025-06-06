// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/logger"
)

func TestNew(t *testing.T) {
	log := logger.New("Test Agent", "v1.23")
	assert.NotNil(t, log)
}
