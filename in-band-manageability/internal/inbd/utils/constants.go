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

// DownloadDir is the directory where the downloaded file will be stored.
const DownloadDir = "/var/cache/manageability/repository-tool/sota"

// SOTADownloadDir is the directory where the downloaded file will be stored.
const SOTADownloadDir = IntelManageabilityCachePathPrefix + "/repository-tool/sota"

// JWTTokenPath is the path to the JWT token file used for accessing the release service.
const JWTTokenPath = "/etc/intel_edge_node/tokens/release-service/access_token" // #nosec G101 -- This is a file path, not a hardcoded credential

// OTAPackageCertPath is the path to the OTA package certificate.
const OTAPackageCertPath = "/etc/intel-manageability/public/ota_package_cert.pem"

// ConfigFileName is the expected configuration file name.
const ConfigFileName = "intel_manageability.conf"

// MinKeySizeBits is the minimum allowed RSA key size in bits for signature verification.
const MinKeySizeBits = 3000

// Context Timeouts
const SignatureVerificationTimeoutInSeconds = 30
