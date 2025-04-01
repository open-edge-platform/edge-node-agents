// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package info

import "fmt"

var version string // injected at build time
var commit string  // injected at build time

var Component string = "Cluster Agent"
var Version string = fmt.Sprintf("%s-%v", version, commit)
