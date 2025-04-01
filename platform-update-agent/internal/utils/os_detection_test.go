// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockFileReader is a mock implementation of FileReader for testing purposes.
type MockFileReader struct {
	Content string
	Err     error
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return []byte(m.Content), nil
}

func TestDetectOS_Ubuntu(t *testing.T) {
	reader := &MockFileReader{
		Content: `PRETTY_NAME="Ubuntu 22.04.5 LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.5 LTS (Jammy Jellyfish)"
VERSION_CODENAME=jammy
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
UBUNTU_CODENAME=jammy`,
	}

	osType, err := DetectOS(reader, "")
	assert.NoError(t, err)
	assert.Equal(t, "ubuntu", osType)
}

func TestDetectOS_Emt(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="Edge Microvisor Toolkit"
VERSION="3.0.20250312"
ID="Edge Microvisor Toolkit"
VERSION_ID="3.0"
PRETTY_NAME="Edge Microvisor Toolkit 3.0"
ANSI_COLOR="1;34"
HOME_URL="https://github.com/open-edge-platform/edge-microvisor-toolkit"
BUG_REPORT_URL="https://github.com/open-edge-platform/edge-microvisor-toolkit"
SUPPORT_URL="https://github.com/open-edge-platform/edge-microvisor-toolkit"`,
	}

	osType, err := DetectOS(reader, "")
	assert.NoError(t, err)
	assert.Equal(t, "emt", osType)
}

func TestDetectOS_Unknown(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="UnknownOS"
ID=unknown
VERSION="1.0"`,
	}

	osType, err := DetectOS(reader, "")
	assert.Error(t, err)
	assert.Equal(t, "", osType)
}

func TestDetectOS_ForcedUbuntu(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="Some Other OS"
ID=other
VERSION="1.0"`,
	}

	osType, err := DetectOS(reader, "ubuntu")
	assert.NoError(t, err)
	assert.Equal(t, "ubuntu", osType)
}

func TestDetectOS_ForcedDebian(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="Some Other OS"
ID=other
VERSION="1.0"`,
	}

	osType, err := DetectOS(reader, "debian")
	assert.NoError(t, err)
	assert.Equal(t, "debian", osType)
}

func TestDetectOS_ForcedEmt(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="Some Other OS"
ID=other
VERSION="1.0"`,
	}

	osType, err := DetectOS(reader, "emt")
	assert.NoError(t, err)
	assert.Equal(t, "emt", osType)
}

func TestDetectOS_ForcedInvalid(t *testing.T) {
	reader := &MockFileReader{
		Content: `NAME="Some Other OS"
ID=other
VERSION="1.0"`,
	}

	osType, err := DetectOS(reader, "invalid")
	assert.Error(t, err)
	assert.Equal(t, "", osType)
}
