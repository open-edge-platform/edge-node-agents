// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

func TestNoNullByteString(t *testing.T) {
	assert.True(t, utils.NoNullByteString("/etc/test/go"))
	assert.False(t, utils.NoNullByteString("/etc/test\000/go"))
	assert.False(t, utils.NoNullByteString("\000"))
	assert.False(t, utils.NoNullByteString("/etc/test\x00/go"))
	assert.False(t, utils.NoNullByteString("\x00"))
	assert.False(t, utils.NoNullByteString("/etc/test\u0000/go"))
	assert.False(t, utils.NoNullByteString("\u0000"))
	assert.True(t, utils.NoNullByteString("http://asdf.com.%00.com"))
}

func TestNoNullByteURL(t *testing.T) {
	assert.True(t, utils.NoNullByteURL("http://asdf.com"))
	assert.False(t, utils.NoNullByteURL("asdf.%00.com"))
	assert.False(t, utils.NoNullByteURL("asdf\x00com"))
	assert.False(t, utils.NoNullByteURL("asdf\000com"))
	assert.True(t, utils.NoNullByteURL("https://here%40.com%26"))
	assert.False(t, utils.NoNullByteURL("https://here%00.com%26"))
	assert.True(t, utils.NoNullByteURL("here.com/%26name=xyz"))
	assert.False(t, utils.NoNullByteURL("here.com/%2/name=xyz"))
	assert.False(t, utils.NoNullByteURL("here.com/%00name=xyz"))
}
