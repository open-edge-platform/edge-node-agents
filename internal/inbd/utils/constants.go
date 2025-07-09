/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

// ConfigFilePath is the path to the configuration file.
const ConfigFilePath = "/etc/intel_manageability.conf"

// IntelManageabilityCachePathPrefix is the prefix for the Intel Manageability cache path.
const IntelManageabilityCachePathPrefix = "/var/cache/manageability"

// SOTADownloadDir is the directory where the downloaded file will be stored.
const SOTADownloadDir = IntelManageabilityCachePathPrefix + "/repository-tool/sota"

// JWTTokenPath is the path to the JWT token file used for accessing the release service.
const JWTTokenPath = "/etc/intel_edge_node/tokens/release-service/access_token"
