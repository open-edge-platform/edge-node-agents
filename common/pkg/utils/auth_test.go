// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
)

func newTestServer() *httptest.Server {
	return httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
}

func TestGetAuthConfig(t *testing.T) {
	minVersion := uint16(tls.VersionTLS13)
	cipher := []uint16{
		tls.TLS_AES_256_GCM_SHA384,
	}

	server := newTestServer()
	server.StartTLS()
	defer server.Close()

	testConfig, err := utils.GetAuthConfig(t.Context(), server.Certificate())
	require.NoError(t, err)
	require.NotNil(t, testConfig)
	require.NotNil(t, testConfig.RootCAs)
	require.Equal(t, tls.RequireAndVerifyClientCert, testConfig.ClientAuth)
	require.False(t, testConfig.InsecureSkipVerify)
	require.Equal(t, minVersion, testConfig.MinVersion)
	require.Equal(t, cipher, testConfig.CipherSuites)

	tr := &http.Transport{
		TLSClientConfig: testConfig,
	}

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
}

func TestGetAuthContext(t *testing.T) {
	tFile, err := os.CreateTemp(t.TempDir(), "access-token")
	require.NoError(t, err)

	_, err = tFile.WriteString("token")
	require.NoError(t, err)

	defer tFile.Close()

	newCtx := utils.GetAuthContext(t.Context(), tFile.Name())

	val, _ := metadata.FromOutgoingContext(newCtx)
	require.Equal(t, []string{"Bearer token"}, val.Get("authorization"))
}

func TestGetAuthContext_noFile(t *testing.T) {
	newCtx := utils.GetAuthContext(t.Context(), "/tmp/no-file")

	val, _ := metadata.FromOutgoingContext(newCtx)
	require.Equal(t, []string{"Bearer"}, val.Get("authorization"))
}

func TestGetPerRPCCreds_noFile(t *testing.T) {
	tSource := utils.GetPerRPCCreds("/tmp/no-file")

	require.NotNil(t, tSource)
}

func TestGetPerRPCCreds(t *testing.T) {
	tFile, err := os.CreateTemp(t.TempDir(), "access-token")
	require.NoError(t, err)

	_, err = tFile.WriteString("token")
	require.NoError(t, err)

	defer tFile.Close()

	tSource := utils.GetPerRPCCreds(tFile.Name())

	require.NotNil(t, tSource)
}
