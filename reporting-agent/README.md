<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Reporting Agent

## Overview

Gathering statistics from Open Edge Platform installations

TBD

## Develop

To develop Reporting Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/reporting-agent/go.mod).

## Building the Reporting Agent

### Binary Build

Run the `make rabuild` command to build the reporting agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd reporting-agent/
$ make rabuild
$ ls build/artifacts/
reporting-agent
```

### Source tarball

Run the `make tarball` command to generate a tarball of the reporting agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd reporting-agent/
$ make tarball
$ ls build/artifacts/
reporting-agent-<VERSION>  reporting-agent-<VERSION>.tar.gz
```

## Running the Reporting Agent Binary

To run the Reporting agent binary after compiling:

```shell
./build/artifacts/reporting-agent
```

## Additional Commands for Development

- **Build reporting agent binary and mock binaries**:

    ```shell
    make build
    ```

- **Run unit tests**:

    ```shell
    make test
    ```

- **Run linters**:

    ```shell
    make lint
    ```

- **Get code coverage from unit tests**:

    ```shell
    make cover
    ```

## License

Apache-2.0
