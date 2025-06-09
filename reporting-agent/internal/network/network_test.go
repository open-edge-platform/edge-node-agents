// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"os"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestGetNetworkSerialsSuccess(t *testing.T) {
	testutils.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lshw_real.json")
	require.NoError(t, err)
	testutils.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, testData, nil)

	serials, err := GetNetworkSerials(testutils.TestCmdExecutor)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"01:23:34:45:56:67",
		"12:23:34:45:56:67",
		"23:34:45:56:67:89",
		"34:45:56:67:78:89",
	}, serials)
}

func TestGetNetworkSerialsCommandFailure(t *testing.T) {
	testutils.ClearMockOutputs()
	testutils.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, nil, os.ErrPermission)

	_, err := GetNetworkSerials(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read network devices")
}

func TestGetNetworkSerialsMalformedJSON(t *testing.T) {
	testutils.ClearMockOutputs()
	badJSON := []byte(`{ this is not valid json ]`)
	testutils.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, badJSON, nil)

	_, err := GetNetworkSerials(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "unable to unmarshal network serials")
}

func TestGetNetworkSerialsNoSerialsFound(t *testing.T) {
	testutils.ClearMockOutputs()
	// All serial fields empty
	mock := []byte(`[{"serial":""},{"serial":""}]`)
	testutils.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, mock, nil)

	_, err := GetNetworkSerials(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "no network serials found")
}

func TestGetNetworkSerialsMissingSerialField(t *testing.T) {
	testutils.ClearMockOutputs()
	// No "serial" field at all
	mock := []byte(`[{"id":"network:0"},{"id":"network:1"}]`)
	testutils.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, mock, nil)

	_, err := GetNetworkSerials(testutils.TestCmdExecutor)
	require.ErrorContains(t, err, "no network serials found")
}
