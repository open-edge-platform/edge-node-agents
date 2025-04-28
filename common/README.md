<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Common

## Overview

This folder contains common code that is shared across all of the Edge Node Agents. It provides:

- Agent Status API
- Message Logger
- Agent Metrics Provider
- Unit Test and other Common Utilities

## Develop

To develop the common code, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/common/go.mod).

## Commands for Development

- **Generate updated protobuf files**:

    ```
    make buf-gen
    ```

- **Run unit tests**:

    ```
    make test
    ```

- **Run linters**:

    ```
    make lint
    ```

- **Get code coverage from unit tests**:

    ```
    make cover
    ```

## License

Apache-2.0
