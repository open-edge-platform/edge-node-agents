<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Platform Update Agent

## Overview

This Platform Update Agent handles system updates on edge nodes, coordinating with the Maintenance Manager and INBM components.

## Develop

To develop Platform Update Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/platform-update-agent/go.mod).

## Building the Platform Update Agent

### Binary Build

Run the `make puabuild` command to build the platform update agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd platform-update-agent/
$ make ptabuild
$ ls build/artifacts/
platform-update-agent
```

### Debian Package Build

Run the `make package` command to build the platform update agent Debian package. The package can be found in the `build/artifacts` directory.

Example:

```bash
$ cd platform-update-agent/
$ make package
$ ls build/artifacts/
package                                              platform-update-agent_<VERSION>_amd64.buildinfo  platform-update-agent_<VERSION>_amd64.deb
platform-update-agent_<VERSION>_amd64.build  platform-update-agent_<VERSION>_amd64.changes
```

### Source tarball

Run the `make tarball` command to generate a tarball of the platform update agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd platform-update-agent/
$ make tarball
$ ls build/artifacts/
platform-update-agent-<VERSION>  platform-update-agent-<VERSION>.tar.gz
```

## Running the Platform Update Agent Binary

To run the platform update agent binary after compiling:

```
./build/artifacts/platform-update-agent -config configs/platform-update-agent.yaml 
```

## Installing the Platform Update Agent

The platform Update agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/platform-update-agent_<VERSION>_amd64.deb
```

## Uninstalling the Platform Update Agent

To uninstall the platform update agent, use `apt`:

```bash
sudo apt-get purge -y platform-update-agent
```

## SystemD Service Management

- **Status**:

    ```
    sudo systemctl status platform-update-agent
    ```

- **Start**:

    ```
    sudo systemctl start platform-update-agent
    ```

- **Stop**:

    ```
    sudo systemctl stop platform-update-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u platform-update-agent -f
```

## Additional Commands

- **Build platform update agent binary and mock binaries**:

    ```
    make build
    ```

- **Run unit tests**:

    ```
    make test
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

## INBM Framework Integration

The Intel In-Band Manageability (INBM) framework has been integrated into this platform-update-agent. The INBM framework provides:

- **inbc**: CLI client for manageability commands (SOTA, FOTA, configuration)
- **inbd**: Daemon service for system management and updates
- **Comprehensive**: Firmware updates, OS updates, configuration management, telemetry

### Building with INBM

```bash
# Build INBM
make inbm-build

# Test INBM
make inbm-test
```

### INBM Binaries

After building, the following artifacts are available:
- `build/artifacts/inbc`                       - INBM CLI client
- `build/artifacts/inbd`                       - INBM daemon service
- `build/artifacts/install-tc.sh`              - Installs inbm for Ubuntu
- `build/artifacts/uninstall-tc.sh`            - Uninstalls inbm for Ubuntu
- `build/artifacts/intel-inbm.deb`             - Intel In-Band Manageability package
- `build/artifacts/LICENSE`                    - INBM license
- `build/artifacts/retain-3rd-party-notices`   - Retained 3rd party notices
- `build/artifacts/third-party-programs.txt`   - 3rd party licenses and notices

For detailed INBM documentation, see [inbm/README.md](inbm/README.md).
