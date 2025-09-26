/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package common provides utilities used by multiple packages
package common

// RebootCmd is the command used to reboot the system.
// This is used by the EMT and Ubuntu OS updaters.
const RebootCmd = "/usr/sbin/reboot"

// ShutdownCmd is the command used to shutdown the system.
const ShutdownCmd = "/usr/sbin/shutdown"

// TruncateCmd is the command used to truncate the state file.
const TruncateCmd = "truncate"

// OsUpdateToolPathCmd is the command to execute the OS update tool script.
// OsUpdateTool will be changed in 3.1 release. Have to change the name and API call.
// Check https://github.com/intel-sandbox/os.linux.tiberos.ab-update.go/blob/main/README.md
const OsUpdateToolCmd = "/usr/bin/os-update-tool.sh"

// GPGCmd is the command to execute the GPG tool for signature verification.
const GPGCmd = "/usr/bin/gpg"

// IPCmd is the command to execute the ip tool.
const IPCmd = "ip"

// SnapperCmd is the command to execute the snapper tool.
const SnapperCmd = "snapper"

// AptGetCmd is the command to execute the apt-get tool.
const AptGetCmd = "/usr/bin/apt-get"

// DpkgCmd is the command to execute the dpkg tool.
const DpkgCmd = "/usr/bin/dpkg"
