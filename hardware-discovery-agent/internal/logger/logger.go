// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package logger

import (
	"github.com/open-edge-platform/edge-node-agents/common/pkg/logger"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/info"
)

var Logger = logger.New(info.Component, info.Version)
