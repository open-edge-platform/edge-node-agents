// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

func TestGetNetworkSerialsSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testData, err := os.ReadFile("./testdata/lshw_real.json")
	require.NoError(t, err)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, testData, nil)

	serials, err := GetNetworkSerials(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"01:23:34:45:56:67",
		"12:23:34:45:56:67",
		"23:34:45:56:67:89",
		"34:45:56:67:78:89",
	}, serials)
}

func TestGetNetworkSerialsCommandFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, nil, os.ErrPermission)

	_, err := GetNetworkSerials(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read network devices")
}

func TestGetNetworkSerialsMalformedJSON(t *testing.T) {
	testutil.ClearMockOutputs()
	badJSON := []byte(`{ this is not valid json ]`)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, badJSON, nil)

	_, err := GetNetworkSerials(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "unable to unmarshal network serials")
}

func TestGetNetworkSerialsNoSerialsFound(t *testing.T) {
	testutil.ClearMockOutputs()
	// All serial fields empty
	mock := []byte(`[{"serial":""},{"serial":""}]`)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, mock, nil)

	_, err := GetNetworkSerials(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "no network serials found")
}

func TestGetNetworkSerialsMissingSerialField(t *testing.T) {
	testutil.ClearMockOutputs()
	// No "serial" field at all
	mock := []byte(`[{"id":"network:0"},{"id":"network:1"}]`)
	testutil.SetMockOutput("sudo", []string{"lshw", "-json", "-class", "network"}, mock, nil)

	_, err := GetNetworkSerials(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "no network serials found")
}
