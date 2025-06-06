// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	ssmock "github.com/open-edge-platform/edge-node-agents/common/pkg/status-server-mock"
)

func main() {
	err := ssmock.RunMockStatusServer()
	if err != nil {
		fmt.Printf("Error running status server mock: %v", err)
	}
}
