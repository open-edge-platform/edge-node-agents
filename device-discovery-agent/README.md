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

```bash
./build/artifacts/device-discovery-agent -config configs/device-discovery-agent.env
```

Note: The `-config` flag expects a KEY=VALUE format file (.env), NOT YAML.

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

The Device Discovery Agent can be configured using multiple sources. Configuration values are loaded in a specific priority order, where higher priority sources override lower priority ones.

### Configuration Sources

1. **Configuration File** (KEY=VALUE format): `/etc/edge-node/node/confs/device-discovery-agent.env`
2. **Kernel Arguments**: Values from `/proc/cmdline` (if `-use-kernel-args` flag is enabled)
3. **CLI Flags**: Command-line arguments

### Configuration File Format

The configuration file uses KEY=VALUE format (not YAML):

```bash
# Service Endpoints
OBM_SVC=localhost
OBS_SVC=localhost
OBM_PORT=50051
KEYCLOAK_URL=keycloak.example.com
CA_CERT=/etc/intel_edge_node/orch-ca-cert/orch-ca.pem

# Auto-detection
AUTO_DETECT=true

# Optional Settings
DEBUG=false
TIMEOUT=5m
DISABLE_INTERACTIVE=false
USE_KERNEL_ARGS=false
```

### Configuration Priority Order

The agent loads configuration in the following priority order (from lowest to highest):

1. **Default values** (lowest priority) - Built-in defaults
2. **Kernel arguments** - Values from `/proc/cmdline` (if `-use-kernel-args` is enabled)
3. **Configuration file** - Values from file specified with `-config` flag
4. **CLI flags** (highest priority) - Command-line arguments

**Important Notes:**
- Higher priority sources **override** values from lower priority sources
- CLI flags always take precedence over all other configuration sources
- The configuration file overrides kernel arguments
- Kernel arguments override default values
- The `-use-kernel-args` flag enables reading from `/proc/cmdline`
- Supported kernel arguments: `worker_id` (mapped to MAC address), `DEBUG`, `TIMEOUT`

## Usage

Device Discovery Agent operates as a CLI utility with command-line flags for configuration.

### Basic Usage

```bash
./device-discovery-agent [OPTIONS]
```

### Required Flags

The following flags are required unless specified in a configuration file:

- `-obm-svc` - Onboarding manager service address (hostname or IP)
- `-obs-svc` - Onboarding stream service address (hostname or IP)
- `-obm-port` - Onboarding manager port (default: 50051)
- `-keycloak-url` - Keycloak authentication URL (hostname or IP)
- `-ca-cert` - Path to CA certificate (required for TLS)
- `-mac` - MAC address of the device (required unless using `-auto-detect`)

### Optional Flags

**Configuration File:**
- `-config` - Path to configuration file in KEY=VALUE format (e.g., `/etc/edge-node/node/confs/device-discovery-agent.env`)

**Device Information (auto-detected if not provided):**
- `-serial` - Serial number (auto-detected using dmidecode)
- `-uuid` - System UUID (auto-detected using dmidecode)
- `-ip` - IP address (auto-detected from MAC address)

**Auto-Detection:**
- `-auto-detect` - Auto-detect all system information (MAC, serial, UUID, IP)

**Additional Options:**
- `-extra-hosts` - Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')
- `-debug` - Enable debug mode with extended logging
- `-timeout` - Timeout duration for debug mode (default: 5m)
- `-disable-interactive` - Disable interactive mode fallback
- `-use-kernel-args` - Read configuration from kernel command line (/proc/cmdline)

### Examples

#### 1. Using configuration file
```bash
./device-discovery-agent -config /etc/edge-node/node/confs/device-discovery-agent.env
```

#### 2. Auto-detect all system information
```bash
./device-discovery-agent \
      -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -ca-cert /etc/intel_edge_node/orch-ca-cert/orch-ca.pem \
      -auto-detect
```

#### 3. Specify MAC address, auto-detect other info
```bash
./device-discovery-agent \
      -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -ca-cert /etc/intel_edge_node/orch-ca-cert/orch-ca.pem \
      -mac 00:11:22:33:44:55
```

#### 4. Fully manual configuration
```bash
./device-discovery-agent \
      -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -ca-cert /etc/intel_edge_node/orch-ca-cert/orch-ca.pem \
      -mac 00:11:22:33:44:55 \
      -serial ABC123 \
      -uuid 12345678-1234-1234-1234-123456789012 \
      -ip 192.168.1.100
```

#### 5. With debug mode and extra hosts
```bash
./device-discovery-agent \
      -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -ca-cert /etc/intel_edge_node/orch-ca-cert/orch-ca.pem \
      -auto-detect \
      -debug \
      -timeout 10m \
      -extra-hosts "registry.local:10.0.0.1,api.local:10.0.0.2"
```

#### 6. Override config file values with CLI flags
```bash
# Config file has OBM_SVC=localhost, but we override it
./device-discovery-agent \
      -config /etc/edge-node/node/confs/device-discovery-agent.env \
      -obm-svc production.example.com \
      -debug
```

#### 7. Using kernel arguments with configuration file
```bash
# Reads kernel args first, then config file (which overrides kernel args),
# then CLI flags (which override everything)
./device-discovery-agent \
      -config /etc/edge-node/node/confs/device-discovery-agent.env \
      -use-kernel-args \
      -debug
```

### Configuration Priority Example

Here's how the priority system works in practice:

Suppose you have:
- **Kernel arguments** (`/proc/cmdline`): `DEBUG=false TIMEOUT=5m`
- **Config file** (`device-discovery-agent.env`): `DEBUG=true TIMEOUT=10m OBM_SVC=config.example.com`
- **CLI flags**: `-debug=false -obm-svc=cli.example.com`

The final configuration will be:
- `DEBUG=false` (from CLI flag - highest priority)
- `TIMEOUT=10m` (from config file - overrides kernel args)
- `OBM_SVC=cli.example.com` (from CLI flag - highest priority)

The agent processes configuration in this order:
1. Loads defaults
2. Reads kernel arguments (if `-use-kernel-args` enabled)
3. Reads config file (if `-config` provided)
4. Applies CLI flags (final override)


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
- **Config struct** - Main configuration structure with all agent settings
- **LoadFromFile** - Loads configuration from KEY=VALUE format files
- **LoadFromKernelArgs** - Parses configuration from `/proc/cmdline`
- **ApplyValue** - Maps configuration keys to struct fields with type conversion
- **Validate** - Validates required configuration fields
- **WriteToFile** - Persists validated configuration to file
- Host file updates (`UpdateHosts`)
- Environment variable loading (`LoadEnvConfig`, `ReadEnvVars`)
- File I/O operations (`SaveToFile`)
- Temporary script creation (`CreateTempScript`)
- Constants for file paths

### internal/parser
Kernel command line argument parsing (legacy support).

### internal/sysinfo
System information retrieval using dmidecode and network interfaces:
- Hardware serial number (via `sudo dmidecode`)
- System UUID (via `sudo dmidecode`)
- IP address lookup by MAC
- Primary MAC address detection

**Note:** Requires sudo permissions for dmidecode commands. See `/etc/sudoers.d/device-discovery-agent` for configured permissions.

### internal/info
Version information and build metadata:
- Component name
- Version string (injected at build time)

## Auto-Detection Features

The application can automatically detect system information:

1. **Serial Number** - Retrieved using `sudo dmidecode -s system-serial-number`
2. **UUID** - Retrieved using `sudo dmidecode -s system-uuid`
3. **MAC Address** - Automatically detects the primary network interface MAC using Go's `net.Interfaces()`
4. **IP Address** - Automatically detected from network interfaces associated with the specified MAC address

When using `-auto-detect`, all system information is automatically gathered. Individual fields can also be auto-detected by omitting the corresponding flag.

## License

Apache-2.0
