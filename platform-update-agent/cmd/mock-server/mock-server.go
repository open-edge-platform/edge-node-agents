// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"

	mock_server "github.com/open-edge-platform/edge-node-agents/platform-update-agent/cmd/mock-server/mock-server"
)

func main() {
	var serverType mock_server.ServerType

	flag.Var(&serverType, "server", "Server type (UBUNTU or EMT)")
	flag.Parse()

	fmt.Printf("Selected server type: %v\n", serverType.String())

	mock_server.StartMockServer(serverType)
}
