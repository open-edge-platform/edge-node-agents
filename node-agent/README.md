<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Node Agent

## Overview

Node Agent is part of the Edge Manageability Framework. It is responsible to register and authenticate the Edge Node with the Edge Infrastructure Manager service when the Edge Node is first brought up. It is also responsible for creating and refreshing tokens for other agents running on the Edge Node. It reports status of Edge Node to the Edge Infrastructure Manager as it onboards.

It:
- Registers and authenticates Edge Node with Edge Infrastructure Manager service
- Reports status of Edge Node to the Edge Infrastructure Manager as it onboards
- Creates and refreshes tokens for other agents running on the Edge Node

## Develop

To develop Node Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/node-agent/go.mod).

## Building the Node Agent

### Binary Build

Run the `make nabuild` command to build the node agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd node-agent/
$ make nabuild
$ ls build/artifacts/
node-agent
```

### Debian Package Build

Run the `make package` command to build the node agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd node-agent/
$ make package
$ ls build/artifacts/
node-agent_<VERSION>_amd64.build  node-agent_<VERSION>_amd64.buildinfo  node-agent_<VERSION>_amd64.changes  node-agent_<VERSION>_amd64.deb  package
```

### Source tarball

Run the `make tarball` command to generate a tarball of the node agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd node-agent/
$ make tarball
$ ls build/artifacts/
node-agent-<VERSION>  node-agent-<VERSION>.tar.gz
```

## Running the Node Agent Binary

To run the node agent binary after compiling:

```
./build/artifacts/node-agent -config configs/node-agent.yaml 
```

## Installing the Node Agent

The node agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/node-agent_<VERSION>_amd64.deb
```

## Uninstalling the Node Agent

To uninstall the node agent, use `apt`:

```bash
sudo apt-get purge -y node-agent
```

## SystemD Service Management

- **Status**

    ```
    sudo systemctl status node-agent
    ```

- **Start**

    ```
    sudo systemctl start node-agent
    ```

- **Stop**

    ```
    sudo systemctl stop node-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u node-agent
```

## Additional Commands for Development

- **Build node agent binary and mock binaries**:

    ```
    make build
    ```

- **Run unit tests**:

    ```
    make unit-test
    ```

- **Run integration tests**:

    ```
    make integration_test
    ```

- **Run linters**:

    ```
    make lint
    ```

- **Get code coverage from unit and intergration tests**:

    ```
    make cover
    ```

- **Run package test**:

    ```
    make package_test
    ```

## License

Apache-2.0
