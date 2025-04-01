// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package logger

import (
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/info"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/logger"
)

var Logger = logger.New(info.Component, info.Version)
