<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# In-Band Manageability

## Overview

In-Band Manageability (INBM) is part of the Edge Manageability Framework. It provides in-band management capabilities for edge devices, including firmware updates, configuration management, and system operations.

It consists of:
- **inbd**: The In-Band Manageability daemon that provides core management services
- **inbc**: The In-Band Manageability client for interacting with the daemon

Key features:
- System firmware and OS updates (SOTA, FOTA)
- Configuration management
- Power management operations
- Secure communication with signature verification
- gRPC-based client-server architecture

## Develop

To develop In-Band Manageability, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)
- [Buf](https://buf.build) for Protocol Buffer management

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/in-band-manageability/go.mod).

## Building the In-Band Manageability Tools

### Binary Build

Run the `make build` command to build both inbc and inbd binaries. The compiled binaries can be found in the `build/artifacts` directory.

Example:

```bash
$ cd in-band-manageability/
$ make build
$ ls build/artifacts/
inbc  inbd  LICENSE  retain-3rd-party-notices  third-party-programs.txt
```

### Individual Binary Builds

To build individual components:

```bash
# Build only the client (inbc)
$ make inbcbuild

# Build only the daemon (inbd)  
$ make inbdbuild
```

### Debian Package Build

Run the `make package` command to build the In-Band Manageability Debian package. The package can be found in the `build/artifacts` directory.

Example:

```bash
$ cd in-band-manageability/
$ make package
$ ls build/artifacts/
in-band-manageability_<VERSION>_amd64.deb  package
```

### Source Tarball

Run the `make tarball` command to generate a tarball of the In-Band Manageability code. The tarball can be found in the `build/artifacts` directory.

Example:

```bash
$ cd in-band-manageability/
$ make tarball
$ ls build/artifacts/
in-band-manageability-<VERSION>  in-band-manageability-<VERSION>.tar.gz
```

## Running the In-Band Manageability Tools

### Running the Daemon (inbd)

To run the inbd daemon after compiling:

```bash
sudo ./build/artifacts/inbd -s /tmp/inbd.sock
```

This will start the daemon and listen for client connections on `/tmp/inbd.sock`.

### Running the Client (inbc)

To run the inbc client and connect to the daemon:

```bash
sudo ./build/artifacts/inbc --socket /tmp/inbd.sock sota --mode full --reboot=false
```

This will connect to the daemon via `/tmp/inbd.sock` and send a SOTA (Software Over-The-Air) update command.

### Additional Client Commands

The inbc client supports various management operations:

```bash
# Configuration updates
sudo ./build/artifacts/inbc --socket /tmp/inbd.sock config --path /path/to/config.conf

# Firmware updates  
sudo ./build/artifacts/inbc --socket /tmp/inbd.sock fota --path /path/to/firmware.bin

# System reboot
sudo ./build/artifacts/inbc --socket /tmp/inbd.sock reboot

# Query system information
sudo ./build/artifacts/inbc --socket /tmp/inbd.sock query --option all
```

## Installing the In-Band Manageability Package

The In-Band Manageability Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/in-band-manageability_<VERSION>_amd64.deb
```

## Uninstalling the In-Band Manageability Package

To uninstall the In-Band Manageability package, use `apt`:

```bash
sudo apt-get purge -y intel-inbm
```

## SystemD Service Management

The inbd daemon can be managed as a systemd service:

- **Status**:
    ```bash
    sudo systemctl status inbd
    ```

- **Start**:
    ```bash
    sudo systemctl start inbd
    ```

- **Stop**:
    ```bash
    sudo systemctl stop inbd
    ```

- **Enable** (start automatically on boot):
    ```bash
    sudo systemctl enable inbd
    ```

- **View logs**:
    ```bash
    sudo journalctl -fu inbd
    ```

## Development Commands

### Testing

Run the test suite:

```bash
$ make test
```

### Linting

Run code linting:

```bash
$ make lint
```

### Code Coverage

Generate code coverage reports:

```bash
$ make cover
```

### Protocol Buffer Generation

Regenerate Protocol Buffer files:

```bash
$ make generate-proto
```

### Cleaning Build Artifacts

Clean the build directory:

```bash
$ make clean
```
