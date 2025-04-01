<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Platform Telemetry Agent

## Overview

This Platform Telemetry Agent for Observability as a Service (ObaaS) manages and updates node metric and log configurations, re-executing metric and log collectors with the latest configuration

## Develop

To develop Platform Telemetry Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/platform-telemetry-agent/go.mod).

## Building the Platform Telemetry Agent

### Binary Build

Run the `make ptabuild` command to build the platform telemetry agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd platform-telemetry-agent/
$ make ptabuild
$ ls build/artifacts/
platform-telemetry-agent
```

### Debian Package Build

Run the `make package` command to build the platform telemetry agent Debian package. The package can be found in the `build/artifacts` directory.

Example:

```bash
$ cd platform-telemetry-agent/
$ make package
$ ls build/artifacts/
package                                                 platform-telemetry-agent_<VERSION>_amd64.buildinfo  platform-telemetry-agent_<VERSION>_amd64.deb
platform-telemetry-agent_<VERSION>_amd64.build  platform-telemetry-agent_<VERSION>_amd64.changes
```

### Source tarball

Run the `make tarball` command to generate a tarball of the platform telemetry agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd platform-telemetry-agent/
$ make tarball
$ ls build/artifacts/
platform-telemetry-agent-<VERSION>  platform-telemetry-agent-<VERSION>.tar.gz
```

## Running the Platform Telemetry Agent Binary

To run the platform telemetry agent binary after compiling:

```
./build/artifacts/platform-telemetry-agent -config configs/platform-telemetry-agent.yaml 
```

## Installing the Platform Telemetry Agent

The platform telemetry agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/platform-telemetry-agent_<VERSION>_amd64.deb
```

## Uninstalling the Platform Telemetry Agent

To uninstall the platform telemetry agent, use `apt`:

```bash
sudo apt-get purge -y platform-telemetry-agent
```

## SystemD Service Management

- **Status**:

    ```
    sudo systemctl status platform-telemetry-agent
    ```

- **Start**:

    ```
    sudo systemctl start platform-telemetry-agent
    ```

- **Stop**:

    ```
    sudo systemctl stop platform-telemetry-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u platform-telemetry-agent -f
```

## Additional Commands

- **Build platform telemetry agent binary and mock binaries**:

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
