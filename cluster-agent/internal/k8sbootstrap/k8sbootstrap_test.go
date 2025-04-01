// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Unit Tests for package k8sbootstrap: Testing DownloadInstallScript and InstallScript
package k8sbootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Testing successful install script .sh function
func TestExecRunSuccess(t *testing.T) {
	err := Execute(context.Background(), "echo Hello World")
	assert.NoError(t, err)
}

// Testing failure install script .sh function
func TestExecRunFailCommand(t *testing.T) {
	err := Execute(context.Background(), "non-existing-command")
	assert.Error(t, err)
	assert.Contains(t, "exit status 127", err.Error())
}
