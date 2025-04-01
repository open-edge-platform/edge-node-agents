// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func newTestServer() *httptest.Server {
	return httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
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

	ctx := context.Background()
	testConfig, err := utils.GetAuthConfig(ctx, server.Certificate())
	assert.Nil(t, err)
	assert.NotNil(t, testConfig)
	assert.NotNil(t, testConfig.RootCAs)
	assert.Equal(t, tls.RequireAndVerifyClientCert, testConfig.ClientAuth)
	assert.Equal(t, false, testConfig.InsecureSkipVerify)
	assert.Equal(t, minVersion, testConfig.MinVersion)
	assert.Equal(t, cipher, testConfig.CipherSuites)

	tr := &http.Transport{
		TLSClientConfig: testConfig,
	}

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	resp, err := client.Get(server.URL)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetAuthContext(t *testing.T) {
	tFile, err := os.CreateTemp("", "access-token")
	require.Nil(t, err)

	_, err = tFile.Write([]byte("token"))
	require.Nil(t, err)

	defer tFile.Close()
	defer os.Remove(tFile.Name())

	ctx := context.Background()
	newCtx := utils.GetAuthContext(ctx, tFile.Name())

	val, _ := metadata.FromOutgoingContext(newCtx)
	assert.Equal(t, val.Get("authorization"), []string{"Bearer token"})

}

func TestGetAuthContext_noFile(t *testing.T) {

	ctx := context.Background()
	newCtx := utils.GetAuthContext(ctx, "/tmp/no-file")

	val, _ := metadata.FromOutgoingContext(newCtx)
	assert.Equal(t, val.Get("authorization"), []string{"Bearer"})

}

func TestGetPerRPCCreds_noFile(t *testing.T) {

	tSource := utils.GetPerRPCCreds("/tmp/no-file")

	assert.NotNil(t, tSource)

}

func TestGetPerRPCCreds(t *testing.T) {
	tFile, err := os.CreateTemp("", "access-token")
	require.Nil(t, err)

	_, err = tFile.Write([]byte("token"))
	require.Nil(t, err)

	defer tFile.Close()
	defer os.Remove(tFile.Name())

	tSource := utils.GetPerRPCCreds(tFile.Name())

	assert.NotNil(t, tSource)

}
