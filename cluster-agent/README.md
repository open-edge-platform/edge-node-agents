<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Cluster Agent

## Overview

Cluster Agent is part of the Edge Manageability Framework. It provides:

- Registration in Cluster Orchestrator service
- Bootstrapping of Kubernetes Engine
- Removal of Kubernetes Engine

## Develop

To develop Cluster Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/cluster-agent/go.mod).

## Building the Cluster Agent

### Binary Build

Run the `make cabuild` command to build the cluster agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd cluster-agent/
$ make cabuild
$ ls build/artifacts/
cluster-agent
```

### Debian Package Build

Run the `make package` command to build the cluster agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd cluster-agent/
$ make package
$ ls build/artifacts/
cluster-agent_<VERSION>_amd64.build  cluster-agent_<VERSION>_amd64.buildinfo  cluster-agent_<VERSION>_amd64.changes  cluster-agent_<VERSION>_amd64.deb  package
```

### Source tarball

Run the `make tarball` command to generate a tarball of the cluster agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd cluster-agent/
$ make tarball
$ ls build/artifacts/
cluster-agent-<VERSION>  cluster-agent-<VERSION>.tar.gz
```

## Running the Cluster Agent Binary

To run the cluster agent binary after compiling:

```
./build/artifacts/cluster-agent -config configs/cluster-agent.yaml 
```

## Installing the Cluster Agent

The cluster agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/cluster-agent_<VERSION>_amd64.deb
```

## Uninstalling the Cluster Agent

To uninstall the cluster agent, use `apt`:

```bash
sudo apt-get purge -y cluster-agent
```

## SystemD Service Management

- **Status**

    ```
    sudo systemctl status cluster-agent
    ```

- **Start**

    ```
    sudo systemctl start cluster-agent
    ```

- **Stop**

    ```
    sudo systemctl stop cluster-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u cluster-agent
```

## Additional Commands for Development

- **Build cluster agent binary and mock binaries**:

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
