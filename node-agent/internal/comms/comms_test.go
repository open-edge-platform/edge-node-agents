// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const guid = "TEST-GUID-TEST-GUID"

const testAuthAccessTokenURL = "keycloak.kind.internal"
const testAuthRsTokenURL = "release.kind.internal"
const testAuthAccessTokenPath = "/tmp/tokens/"     // #nosec G101
const testAuthClientCredsPath = "/tmp/credentials" // #nosec G101

const clientId = "host-manager-m2m-client"              // #nosec G101
const clientSecret = "c6vQ3ljDGIHFLpIozopJwy7BYeL4mwvw" // #nosec G101

var testAuthTokenClients = []string{"node-agent", "hd-agent", "cluster-agent", "platform-update-agent", "platform-observability-agent", "platform-telemetry-agent", "prometheus", "fluent-bit"}

func getAuthConfig() config.ConfigAuth {
	authConf := config.ConfigAuth{
		AccessTokenURL:  testAuthAccessTokenURL,
		AccessTokenPath: testAuthAccessTokenPath,
		RsTokenURL:      testAuthRsTokenURL,
		ClientCredsPath: testAuthClientCredsPath,
		TokenClients:    testAuthTokenClients,
	}
	return authConf
}

func newTestIDPServer(t *testing.T, method string, response any, resource string, returnCode int) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		require.Equal(t, resource, req.URL.String())
		require.Equal(t, method, req.Method)
		if returnCode != http.StatusOK {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err := io.ReadAll(req.Body)
		require.Nil(t, err)
		var res []byte
		if str, ok := response.(string); ok {
			res = []byte(str)
		} else {
			res, err = json.Marshal(response)
			require.Nil(t, err)
		}
		require.Nil(t, err)
		_, err = rw.Write(res)
		require.Nil(t, err)
	}))
}

func TestGetAuthCli(t *testing.T) {
	authConf := getAuthConfig()
	cli, err := comms.GetAuthCli(authConf.AccessTokenURL, guid, nil)
	assert.Nil(t, err)
	assert.NotEmpty(t, cli.BaseUrl)
}

func TestProvisionAccessToken(t *testing.T) {
	authConf := getAuthConfig()

	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)

	tokenResp := oauth2.Token{
		AccessToken: token,
		//Expiry:      time.Now(),
	}

	// Create a new test server with the desired response
	ts := newTestIDPServer(t, http.MethodPost, tokenResp, "/realms/master/protocol/openid-connect/token", http.StatusOK)
	defer ts.Close()

	// create client credentials files
	err = os.MkdirAll(testAuthClientCredsPath, 0755)
	assert.Nil(t, err)
	cliendIdFile := filepath.Join(testAuthClientCredsPath, "client_id")
	_, err = os.OpenFile(cliendIdFile, os.O_CREATE, 0644)
	assert.Nil(t, err)
	err = os.WriteFile(cliendIdFile, []byte(clientId), 0600)
	assert.Nil(t, err)

	clientSecretFile := filepath.Join(testAuthClientCredsPath, "client_secret")
	_, err = os.OpenFile(clientSecretFile, os.O_CREATE, 0644)
	assert.Nil(t, err)
	err = os.WriteFile(clientSecretFile, []byte(clientSecret), 0600)
	assert.Nil(t, err)

	u, _ := url.Parse(ts.URL)

	// Add mock server certificate to system cert pool
	certPool, _ := x509.SystemCertPool()
	certPool.AddCert(ts.Certificate())
	authCli, err := comms.GetAuthCli(u.Host, guid, certPool)
	assert.Nil(t, err)

	ctx := context.Background()

	accessToken, err := authCli.ProvisionAccessToken(ctx, authConf)
	assert.Nil(t, err)
	assert.NotEmpty(t, accessToken.AccessToken)
	defer os.RemoveAll(testAuthClientCredsPath)
}

func TestProvisionAccessTokenClientIDSymlink(t *testing.T) {
	authConf := getAuthConfig()

	// create client credentials files
	err := os.MkdirAll(testAuthClientCredsPath, 0755)
	assert.Nil(t, err)
	clientIdFile := filepath.Join(testAuthClientCredsPath, "actual_client_id")
	_, err = os.OpenFile(clientIdFile, os.O_CREATE, 0644)
	assert.Nil(t, err)
	err = os.WriteFile(clientIdFile, []byte(clientId), 0600)
	assert.Nil(t, err)
	symlinkFile := filepath.Join(testAuthClientCredsPath, "client_id")
	err = os.Symlink(clientIdFile, symlinkFile)
	assert.Nil(t, err)
	defer os.RemoveAll(testAuthClientCredsPath)

	// Create authCli with dummy endpoint
	certPool, _ := x509.SystemCertPool()
	authCli, err := comms.GetAuthCli("localhost:123", guid, certPool)
	assert.Nil(t, err)

	ctx := context.Background()

	token, err := authCli.ProvisionAccessToken(ctx, authConf)
	fmt.Printf("%v\n", err)
	assert.NotNil(t, err)
	assert.Empty(t, token)
}

func TestProvisioningAccessTokenClientSecretSymlink(t *testing.T) {
	authConf := getAuthConfig()

	// create client credentials files
	err := os.MkdirAll(testAuthClientCredsPath, 0755)
	assert.Nil(t, err)
	clientIdFile := filepath.Join(testAuthClientCredsPath, "client_id")
	_, err = os.OpenFile(clientIdFile, os.O_CREATE, 0644)
	assert.Nil(t, err)
	err = os.WriteFile(clientIdFile, []byte(clientId), 0600)
	assert.Nil(t, err)

	clientSecretFile := filepath.Join(testAuthClientCredsPath, "actual_client_secret")
	_, err = os.OpenFile(clientSecretFile, os.O_CREATE, 0644)
	assert.Nil(t, err)
	err = os.WriteFile(clientSecretFile, []byte(clientSecret), 0600)
	assert.Nil(t, err)
	symlinkFile := filepath.Join(testAuthClientCredsPath, "client_secret")
	err = os.Symlink(clientSecretFile, symlinkFile)
	assert.Nil(t, err)
	defer os.RemoveAll(testAuthClientCredsPath)

	// Create authCli with dummy endpoint
	certPool, _ := x509.SystemCertPool()
	authCli, err := comms.GetAuthCli("localhost:123", guid, certPool)
	assert.Nil(t, err)

	ctx := context.Background()

	token, err := authCli.ProvisionAccessToken(ctx, authConf)
	fmt.Printf("%v\n", err)
	assert.NotNil(t, err)
	assert.Empty(t, token)
}

func TestProvisionReleaseServiceToken(t *testing.T) {
	accessToken := "TEST-ACCESS-TOKEN"
	authConf := getAuthConfig()

	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	ts := newTestIDPServer(t, http.MethodGet, token, "/token", http.StatusOK)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)

	// Add mock server certificate to system cert pool
	certPool, _ := x509.SystemCertPool()
	certPool.AddCert(ts.Certificate())

	ctx := context.Background()

	// Now create release service token
	relAuthCli, err := comms.GetAuthCli(u.Host, guid, certPool)
	assert.Nil(t, err)

	relToken, err := relAuthCli.ProvisionReleaseServiceToken(ctx, authConf, accessToken)
	assert.Nil(t, err)
	assert.NotEmpty(t, relToken.AccessToken)
	defer os.RemoveAll(testAuthClientCredsPath)
}

func TestProvisionReleaseServiceTokenAnonymous(t *testing.T) {
	accessToken := "TEST-ACCESS-TOKEN"
	authConf := getAuthConfig()

	token := "anonymous"
	ts := newTestIDPServer(t, http.MethodGet, token, "/token", http.StatusOK)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)

	// Add mock server certificate to system cert pool
	certPool, _ := x509.SystemCertPool()
	certPool.AddCert(ts.Certificate())

	ctx := context.Background()

	// Now create release service token
	relAuthCli, err := comms.GetAuthCli(u.Host, guid, certPool)
	assert.Nil(t, err)

	relToken, err := relAuthCli.ProvisionReleaseServiceToken(ctx, authConf, accessToken)
	assert.Nil(t, err)
	assert.NotEmpty(t, relToken.AccessToken)
	assert.Equal(t, relToken.AccessToken, token)
	defer os.RemoveAll(testAuthClientCredsPath)
}
