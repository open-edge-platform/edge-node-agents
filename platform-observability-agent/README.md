<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Paltform Observability Agent

## Overview

Platform Observability Agent is part of the Edge Manageability Framework. It is a common log and metrics scraper.

## Develop

To develop Platform Observability Agent, the following dependencies are required:
- [Fluent Bit](https://fluentbit.io/)
- [telegraf](https://www.influxdata.com/time-series-platform/telegraf)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)

They can be downloaded to `assets` directory with following commands
```
make download_deps
make unpack_deps
```

Versions of dependencies can be set in the Makefile in variables
- `FB_VERSION`
- `TELEGRAF_VERSION`
- `OTELCOL_VERSION`

## Building the Platform Observability Agent

### Debian Package Build

Run the `make package` command to build the platform observability agent Debian package. The package can be found in the `build/artifacts` directory.

Example

```bash
$ cd platform-observability-agent/
$ make package
$ ls build/artifacts/
package                                         platform-observability-agent_<VERSION>_amd64.buildinfo  platform-observability-agent_<VERSION>_amd64.deb
platform-observability-agent_<VERSION>_amd64.build  platform-observability-agent_<VERSION>_amd64.changes    platform-observability-agent-dbgsym_<VERSION>_amd64.ddeb
```

### Source tarball

Run the `make tarball` command to generate a tarball of the platform observability agent code. The tarball can be found in the `build/artifacts` directory.

Example

```bash
$ cd platform-observability-agent/
$ make tarball
$ ls build/artifacts/
platform-observability-agent-<VERSION>  platform-observability-agent-<VERSION>.tar.gz
```

## Installing the Platform Observability Agent

The platform observability agent Debian package can be installed using `apt`:

```bash
sudo apt install -y ./build/artifacts/platform-observability-agent_<VERSION>_amd64.deb
```

## Uninstalling the Platform Observability Agent

To uninstall the platform observability agent, use `apt`:

```bash
sudo apt-get purge -y platform-observability-agent
```

## SystemD Service Management

- **Status**

    ```
    sudo systemctl status platform-observability-logging
    sudo systemctl status platform-observability-health-check
    sudo systemctl status platform-observability-metrics
    sudo systemctl status platform-observability-collector
    ```

- **Start**

    ```
    sudo systemctl start platform-observability-logging
    sudo systemctl start platform-observability-health-check
    sudo systemctl start platform-observability-metrics
    sudo systemctl start platform-observability-collector
    ```

- **Stop**

    ```
    sudo systemctl stop platform-observability-logging
    sudo systemctl stop platform-observability-health-check
    sudo systemctl stop platform-observability-metrics
    sudo systemctl stop platform-observability-collector
    ```

## Logs Management

To view logs:

```
sudo journalctl -u platform-observability-logging
sudo journalctl -u platform-observability-health-check
sudo journalctl -u platform-observability-metrics
sudo journalctl -u platform-observability-collector
```

## Additional Commands for Development

- **Run package test**:

    ```
    make package_test
    ```

## License

Apache-2.0
