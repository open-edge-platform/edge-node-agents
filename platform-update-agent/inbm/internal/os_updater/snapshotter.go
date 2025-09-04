/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

// Snapshotter is an interface that contains the method to take a snapshot of the OS.
type Snapshotter interface {
	Snapshot() error
}
