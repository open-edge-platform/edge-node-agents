/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

// Context Timeouts
const clientDialTimeoutInSeconds = 5
const sourceTimeoutInSeconds = 15
const emtSoftwareUpdateTimerInSeconds = 2100 // 35 minutes - accounts for 30min HTTP timeout + disk check + download time
const defaultSoftwareUpdateTimerInSeconds = 660
const configTimeoutInSeconds = 15
const firmwareUpdateTimerInSeconds = 90
const queryTimeoutInSeconds = 15
