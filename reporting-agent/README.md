<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Reporting Agent

## Overview

The Reporting Agent is responsible for collecting a comprehensive set of metrics and system information from Open Edge Platform installations.
It gathers data from a variety of sources to provide a detailed snapshot of the system's hardware, software, and runtime environment.

Collected data sources include:

- **lscpu**: Provides detailed information about the CPU architecture and capabilities.
- **lsblk**: Lists information about all available or the specified block devices.
- **kubectl**: Collects Kubernetes cluster and node information, including node status and resource usage.
- **dmidecode**: Extracts hardware information from the system's DMI (SMBIOS) tables, such as BIOS, system, and memory details.
- **lshw**: Delivers comprehensive hardware configuration details, including memory, CPU, disks, and network interfaces.
- **date**: Captures the current system date and time.
- **locale**: Reports the system's locale and language settings.
- **uname**: Provides kernel name, version, and other system identifiers.
- **/etc/os-release**: Reads operating system identification data.
- **/proc/uptime**: Retrieves the system uptime in seconds.

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

## Security

The endpoint specified in the `/etc/edge-node/metrics/endpoint` file must use the `https` protocol.

To authenticate with the backend, the application requires a user and password, which must be provided in the `/etc/edge-node/metrics/token` file in the format `username:password`.

TLS version 1.3 is used for backend communication if supported by the server; otherwise, TLS 1.2 is used.

The user running the application should be added to the sudoers file ([see config/sudoers.d/reporting-agent](config/sudoers.d/reporting-agent)), as the `dmidecode` and `lshw` applications require such privileges.

The same user must also have execute access to the `kubectl` binary and read access to the `kubeconfig` file. The paths to these files are specified in the [`reporting-agent.yaml`](config/reporting-agent.yaml) configuration file ([see config/reporting-agent.yaml](config/reporting-agent.yaml)).

## License

Apache-2.0