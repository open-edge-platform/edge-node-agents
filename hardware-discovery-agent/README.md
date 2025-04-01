<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Hardware Discovery Agent

## Overview

Hardware Discovery Agent is part of the Edge Manageability Framework. It is used to discover Edge Node host hardware features and report them to the Edge Infrastructure Manager service. It uses following data sources:

- `udevadm` command: monitoring on hardware changes, only monitor on `block` and `net` subsystem.
- `dmidecode` command: to extract system serial number.
- `/proc` directory: read `/proc/cpuinfo` for CPU information.
- `/sys` directory: read `/sys/devices/system/memory`, `/sys/class/net`, `/sys/block` for memory, Network Interface Card, and disk information.
- `lsmem` command: retrieves information related to the memory available on the Edge Node.
- `lsblk` command: provides disk information for the Edge Node.
- `lscpu` command: extracts information about the CPUs installed on the Edge Node.
- `lsusb` command: collects information on the USB devices connected.
- `lshw` command: collects information on the GPU devices connected to the Edge Node.
- `lscpi` command: collects information on the PCI addresses of GPU devices on the Edge Node.
- `ip` command: provides information on the IP addresses associated with the different Network interfaces on the Edge Node.
- `ipmitool` command: BMC interface information.
- `uname` command: provides information on the kernel version installed on the Edge Node.
- `lsb_release` command: provides information on the OS installed on the Edge Node.

If any tool is missing Hardware Discovery Agent will still generate the output and send it to the Edge Infrastructure Manager, but those fields populated by the missing tools will be left empty.

## Develop

To develop Hardware Discovery Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/hardware-discovery-agent/go.mod).

## Building the Hardware Discovery Agent

### Binary Build

Run the `make hdabuild` command to build the hardware discovery agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd hardware-discovery-agent/
$ make hdabuild
$ ls build/artifacts/
hd-agent
```

### Debian Package Build

Run the `make package` command to build the hardware discovery agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd hardware-discovery-agent/
$ make package
$ ls build/artifacts/
hardware-discovery-agent_<VERSION>_amd64.build      hardware-discovery-agent_<VERSION>_amd64.changes  package
hardware-discovery-agent_<VERSION>_amd64.buildinfo  hardware-discovery-agent_<VERSION>_amd64.deb
```

### Source tarball

Run the `make tarball` command to generate a tarball of the hardware discovery agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd hardware-discovery-agent/
$ make tarball
$ ls build/artifacts/
hardware-discovery-agent-<VERSION>  hardware-discovery-agent-<VERSION>.tar.gz
```

## Running the Hardware Discovery Agent Binary

To run the hardware discovery agent binary after compiling:

```
./build/artifacts/hd-agent -config config/hd-agent.yaml 
```

## Installing the Hardware Discovery Agent

The hardware discovery agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/hardware-discovery-agent_<VERSION>_amd64.deb
```

## Uninstalling the Hardware Discovery Agent

To uninstall the hardware discovery agent, use `apt`:

```bash
sudo apt-get purge -y hardware-discovery-agent
```

## SystemD Service Management

- **Status**

    ```
    sudo systemctl status hardware-discovery-agent
    ```

- **Start**

    ```
    sudo systemctl start hardware-discovery-agent
    ```

- **Stop**

    ```
    sudo systemctl stop hardware-discovery-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u hardware-discovery-agent
```

## Additional Commands for Development

- **Build hardware discovery agent binary and mock binaries**:

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
