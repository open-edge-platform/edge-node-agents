<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Platform Manageability Agent

## Overview

Platform Manageability Agent (PMA) is part of the Edge Manageability Framework. It provides a unified interface for managing platform-level manageability features on Edge Node devices, with a primary focus on Intel vPro technology and Device Management Toolkit activation. The agent integrates RPC (Remote provisioning client) and LMS (Local Management Service) configurations to enable comprehensive device management capabilities.

It:

- Manages platform-level manageability features for Edge Nodes using Intel vPro technology
- Enables Device Management Toolkit activation for remote device management
- Integrates RPC and LMS configurations for comprehensive manageability operations
- Provides hardware monitoring and management capabilities
- Handles firmware and platform configuration operations
- Reports platform status and metrics to the Edge Infrastructure Manager

## Develop

To develop Platform Manageability Agent, the following prerequisites are required:

- [Go programming language](https://go.dev)
- [Golangci-lint](https://github.com/golangci/golangci-lint)

The required Go version for the agents is outlined in the [go.mod file](https://github.com/open-edge-platform/edge-node-agents/blob/main/platform-manageability-agent/go.mod).

## Building the Platform Manageability Agent

### Binary Build

Run the `make pmabuild` command to build the platform manageability agent binary. The compiled binary can be found in the `build/artifacts` directory.

Example:

```bash
$ cd platform-manageability-agent/
$ make pmabuild
$ ls build/artifacts/package
platform-manageability-agent
```

### Debian Package Build

Run the `make package` command to build the platform manageability agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd platform-manageability-agent/
$ make package
$ ls build/artifacts/
platform-manageability-agent_<VERSION>_amd64.build  platform-manageability-agent_<VERSION>_amd64.buildinfo  platform-manageability-agent_<VERSION>_amd64.changes  platform-manageability-agent_<VERSION>_amd64.deb  package
```

### Tarball Build

Run the `make tarball` command to build a tarball containing the Platform Manageability Agent source code and vendored dependencies. This tarball can be used for installation in EMT (Edge Management Toolkit) via an RPM package.

Example

```bash
$ cd platform-manageability-agent/
$ make tarball
$ ls build/artifacts/
platform-manageability-agent-<VERSION>.tar.gz
```

## Configuration

The Platform Manageability Agent is configured using a YAML configuration file located at:
`/etc/edge-node/node/confs/platform-manageability-agent.yaml`

Key configuration sections include:

- `manageability`: Core agent settings and service URLs
- `auth`: Authentication and token management settings  
- `status`: Status reporting and client configuration
- `metricsEndpoint`: Metrics collection endpoint

## Installation

The agent can be installed using the Debian package:

```bash
sudo apt install -y ./build/package/platform-manageability-agent_<VERSION>_amd64.deb
```

This will:

- Create the `platform-manageability-agent` system user
- Set up necessary directories and permissions
- Install the configuration files
- Configure the systemd service

## Running

Start the agent using systemd:

```bash
sudo systemctl start platform-manageability-agent
sudo systemctl enable platform-manageability-agent
```

Check the status:

```bash
sudo systemctl status platform-manageability-agent
```

View logs:

```bash
sudo journalctl -u platform-manageability-agent -f
```

## Running the Agent Manually

You can run the Platform Manageability Agent directly from the command line for development or troubleshooting. The `--config` flag is required to specify the configuration file location.

Example:

```bash
./build/artifacts/platform-manageability-agent --config /etc/edge-node/node/confs/platform-manageability-agent.yaml
```

- The agent will log to stdout/stderr by default.
- Use the `--config` flag to point to your YAML configuration file.
- You can also use `platform-manageability-agent version` to print the agent version and exit.

## Security

- The agent is designed to run as a dedicated system user (`platform-manageability-agent`) with least privilege.
- Configuration files (such as `/etc/edge-node/platform-manageability/confs/platform-manageability-agent.yaml`) should be readable only by the agent user and root.
- Secrets (tokens, credentials) should never be stored in logs or world-readable files.
- Always review configuration and permissions after installation.

## Logging

- The agent uses structured logging via [logrus](https://github.com/sirupsen/logrus).
- Log level is controlled by the `logLevel` field in the configuration file. Supported values: `debug`, `info`, `error`.
- By default, logs are written to stdout/stderr and captured by systemd/journal.
- To increase log verbosity for troubleshooting, set `logLevel: debug` in the config and restart the agent.

## Troubleshooting

- To check agent status: `sudo systemctl status platform-manageability-agent`
- To view logs: `sudo journalctl -u platform-manageability-agent -f`
- If the agent fails to start:
  - Check the configuration file path and permissions.
  - Review logs for errors related to configuration, permissions, or dependencies.
  - Enable debug logging for more detailed output.
- For metrics issues: If metrics initialization fails, the agent logs an error but continues running (metrics are optional).
- For further help, consult the project documentation or open an issue on the project repository.

## Uninstall

To completely remove the Platform Manageability Agent from your system, use the following command:

```bash
sudo apt purge -y platform-manageability-agent
```

- This will remove the agent binary, systemd service, all installed files, and configuration files.
- The system user `platform-manageability-agent` may also remain; you can remove it manually if desired.

## License

This project is licensed under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).
