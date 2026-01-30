<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Device Discovery Agent

## Overview

Device Discovery Agent is part of the Edge Manageability Framework. It is responsible for discovering and registering Edge Node devices with the Edge Infrastructure Manager service during initial onboarding. The agent collects system information and handles authentication with the onboarding service.

It:

- Discovers and registers Edge Node devices with the Edge Infrastructure Manager
- Collects hardware information (UUID, serial number, MAC address, IP address)
- Handles authentication with Keycloak and the onboarding service
- Supports both interactive and non-interactive onboarding modes
- Provides auto-detection of system information

The agent uses the following data sources:

- `dmidecode` command: to extract system serial number and UUID
- Network interfaces: to detect MAC addresses and IP addresses
- Keycloak: for authentication and token management

## Develop

To develop Device Discovery Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined [here](https://github.com/open-edge-platform/edge-node-agents/blob/main/device-discovery-agent/go.mod).

## Building the Device Discovery Agent

### Binary Build

Run the `make ddabuild` command to build the device discovery agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd device-discovery-agent/
$ make ddabuild
$ ls build/artifacts/
device-discovery-agent
```

### Debian Package Build

Run the `make package` command to build the device discovery agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd device-discovery-agent/
$ make package
$ ls build/artifacts/
device-discovery-agent_<VERSION>_amd64.build      device-discovery-agent_<VERSION>_amd64.changes  package
device-discovery-agent_<VERSION>_amd64.buildinfo  device-discovery-agent_<VERSION>_amd64.deb
```

### Source tarball

Run the `make tarball` command to generate a tarball of the device discovery agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd device-discovery-agent/
$ make tarball
$ ls build/artifacts/
device-discovery-agent-<VERSION>  device-discovery-agent-<VERSION>.tar.gz
```

## Running the Device Discovery Agent Binary

To run the device discovery agent binary after compiling:

```
./build/artifacts/device-discovery-agent -config configs/device-discovery-agent.yaml
```

## Installing the Device Discovery Agent

The device discovery agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/device-discovery-agent_<VERSION>_amd64.deb
```

## Uninstalling the Device Discovery Agent

To uninstall the device discovery agent, use `apt`:

```bash
sudo apt-get purge -y device-discovery-agent
```

## SystemD Service Management

- **Status**

    ```
    sudo systemctl status device-discovery-agent
    ```

- **Start**

    ```
    sudo systemctl start device-discovery-agent
    ```

- **Stop**

    ```
    sudo systemctl stop device-discovery-agent
    ```

## Logs Management

To view logs:

```
sudo journalctl -u device-discovery-agent
```

## Configuration

The Device Discovery Agent can be configured using a YAML configuration file located at:
`/etc/edge-node/node/confs/device-discovery-agent.yaml`

Key configuration sections include:

- `onboarding`: Onboarding service connection settings
- `discovery`: Device discovery intervals and retry settings
- `auth`: Keycloak authentication configuration
- `sysinfo`: System information collection settings

## Usage

Device Discovery Agent operates as a CLI utility with command-line flags for configuration.

### Basic Usage

```bash
./device-discovery-agent [OPTIONS]
```

### Required Flags

- `-obm-svc` - Onboarding manager service address
- `-obs-svc` - Onboarding stream service address  
- `-obm-port` - Onboarding manager port
- `-keycloak-url` - Keycloak authentication URL
- `-mac` - MAC address of the device (required unless using `-auto-detect`)

### Optional Flags

**Device Information:**
- `-serial` - Serial number (auto-detected if not provided)
- `-uuid` - System UUID (auto-detected if not provided)
- `-ip` - IP address (auto-detected from MAC if not provided)

**Auto-Detection:**
- `-auto-detect` - Auto-detect all system information (MAC, serial, UUID, IP)

**Additional Options:**
- `-extra-hosts` - Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')
- `-ca-cert` - Path to CA certificate (default: /etc/idp/server_cert.pem)
- `-debug` - Enable debug mode with timeout
- `-timeout` - Timeout duration for debug mode (default: 5m0s)

### Examples

#### 1. Auto-detect all system information
```bash
./device-discovery-agent -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -auto-detect
```

#### 2. Specify MAC address, auto-detect other info
```bash
./device-discovery-agent -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -mac 00:11:22:33:44:55
```

#### 3. Fully manual configuration
```bash
./device-discovery-agent -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -mac 00:11:22:33:44:55 \
      -serial ABC123 \
      -uuid 12345678-1234-1234-1234-123456789012 \
      -ip 192.168.1.100
```

#### 4. With debug mode and extra hosts
```bash
./device-discovery-agent -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -auto-detect \
      -debug \
      -timeout 10m \
      -extra-hosts "registry.local:10.0.0.1,api.local:10.0.0.2"
```

## Additional Commands for Development

- **Build device discovery agent binary and mock binaries**:

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

- **Get code coverage from unit and integration tests**:

    ```
    make cover
    ```

- **Run package test**:

    ```
    make package_test
    ```

## Package Overview

### cmd/device-discovery
Main application entry point. Contains the `main()` function, CLI flag parsing, and orchestration logic.

### internal/auth
Handles client authentication with Keycloak and token management:
- JWT access token retrieval
- Release token fetching
- Certificate-based authentication

### internal/connection
gRPC client for communicating with the onboarding manager:
- Stream-based non-interactive onboarding
- Interactive onboarding with JWT
- Retry logic with exponential backoff

### internal/config
Configuration management and utility functions:
- File I/O operations
- Host file updates
- Temporary script creation
- Constants for file paths

### internal/parser
Kernel command line argument parsing (legacy support).

### internal/sysinfo
System information retrieval using dmidecode and network interfaces:
- Hardware serial number
- System UUID
- IP address lookup by MAC
- Primary MAC address detection

### internal/info
Version information and build metadata:
- Component name
- Version string (injected at build time)

## Auto-Detection Features

The application can automatically detect system information:

1. **Serial Number** - Retrieved using `dmidecode -s system-serial-number`
2. **UUID** - Retrieved using `dmidecode -s system-uuid`
3. **MAC Address** - Automatically detects the primary network interface MAC
4. **IP Address** - Automatically detected from the specified MAC address

When using `-auto-detect`, all system information is automatically gathered. Individual fields can also be auto-detected by omitting the corresponding flag.

## License

Apache-2.0
