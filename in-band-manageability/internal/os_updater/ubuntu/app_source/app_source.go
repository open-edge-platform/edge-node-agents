/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package appsource provides functionality to add an application source.
package appsource

// Manager is an interface that defines the methods to add an application source.
type Manager interface {
	Add(sourceListFileName string, sources []string, gpgKeyURI string, gpgKeyName string) error
	Remove(fileName string, gpgKeyName string) error
}
