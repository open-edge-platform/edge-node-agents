/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package osupdater updates the OS.
package osupdater

// Cleaner is an interface that contains the method to 
// clean the files after an OS update.
type Cleaner interface {
	Clean() error
}
