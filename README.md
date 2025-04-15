<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Edge Node Agents

## Overview

This repository contains the combined implementations for all of the agents installed
into the OS on the Edge Nodes.

## Get Started

The repository comprises the following agents:

- [**Cluster Agent**](cluster-agent/): handles bootstraping and removal of the Kubernetes
  Engine on the Edge Nodeas well as registering the node with the Cluster Orchestrator service.
- [**Common**](common/): contains common code and packages used by all agents.
- [**Hardware Discovery Agent**](hardware-discovery-agent/): detects the HW features available
  on the Edge Node and reports the HW details to the Host Resource Manager.
- [**Node Agent**](node-agent/): registers and authenicates the Edge Node with the Host Manager
  as well as managing the tokens used by other agents on the Edge Node and reporting the status
  of the Edge Node.
- [**Platform Observability Agent**](platform-observability-agent/): gathers logs and HW metrics
  from the Edge Node and reports them to the Orchestrator.
- [**Platform Telemetry Agent**](platform-telemetry-agent/): manages and updates the log and
  metric collection by the Platform Observability Agent based on the telemetry profile provided
  by the Telemetry Manager.
- [**Platform Update Agent**](platform-update-agent/): handles OS and system updates on the
  Edge Node as requested by the Maintenance Manager.

## Develop

To develop one of the Agents please follow it's specific guide present in the README.md of its specific folder.

## Contribute

To learn how to contribute to the project, see the [Contributor's
Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html).

## Community and Support

To learn more about the project, its community, and governance, visit
the \[Edge Orchestrator Community\](<https://website-name.com>).

For support, start with \[Troubleshooting\](<https://website-name.com>) or
\[contact us\](<https://website-name.com>).

## License

Each agent is licensed under [Apache 2.0][apache-license].

Last Updated Date: April 7, 2025

[apache-license]: LICENSES/Apache-2.0.txt
