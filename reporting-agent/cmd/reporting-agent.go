// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
)

func main() {
	dataCollected := model.InitializeRoot()
	fmt.Printf("Hello, I'm Reporting Agent and I will collect the data in this format: %v", dataCollected)
}
