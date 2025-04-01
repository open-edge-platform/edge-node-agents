// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package auth_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/auth"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/testutil"
	"github.com/stretchr/testify/assert"
)

const tokenPath = "/tmp"

const testAuthAccessTokenURL = "keycloak.test"
const testAuthRsTokenURL = "token-provider.test"
const testAuthAccessTokenPath = "/tmp/tokens/"                            // #nosec G101
const testAuthClientCredsPath = "/etc/intel_edge_node/client-credentials" // #nosec G101

var testAuthTokenClients = []string{"na", "pua", "cluster-agent"}

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

func TestPersistToken(t *testing.T) {
	tokenFile := filepath.Join(tokenPath, config.AccessToken)
	_, err := os.Create(tokenFile)
	assert.Nil(t, err)
	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, tokenFile)
	defer os.Remove(tokenFile)
	assert.Nil(t, err)
}

// No write perm
func TestPersistTokenExistsNoWritePerm(t *testing.T) {

	uid := testutil.SetNonRootUser(t)
	defer testutil.ResetUser(t, uid)

	err := os.MkdirAll("/tmp/test1", 0755)
	assert.Nil(t, err)
	tokenFile := filepath.Join("/tmp/test1", config.AccessToken)
	_, err = os.OpenFile(tokenFile, os.O_CREATE, 0500)
	assert.Nil(t, err)

	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, tokenFile)
	assert.NotNil(t, err)
	defer os.RemoveAll("/tmp/test1")
}

func TestCheckTokenExpiry(t *testing.T) {
	expired := auth.IsTokenRefreshRequired(time.Now())
	assert.True(t, expired)
}

func TestInitializeTokenManager(t *testing.T) {
	authConf := getAuthConfig()
	tokMgr := auth.NewTokenManager(authConf)
	assert.NotNil(t, tokMgr)
}

func TestPopulateTokenClients(t *testing.T) {
	authConf := getAuthConfig()
	tokMgr := auth.NewTokenManager(authConf)
	assert.NotNil(t, tokMgr)

	err := os.MkdirAll(testAuthAccessTokenPath+"na", 0755)
	assert.Nil(t, err)

	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, testAuthAccessTokenPath+"na"+"/access_token")
	assert.Nil(t, err)

	tokMgr.PopulateTokenClients(authConf)
	fmt.Println(tokMgr.TokenClients[0].ClientName)
	fmt.Println(tokMgr.TokenClients[0].AccessToken)
	assert.Equal(t, tokMgr.TokenClients[0].AccessToken, token)
	defer os.RemoveAll(testAuthAccessTokenPath)
}

func TestGetExpiryFromJWT(t *testing.T) {
	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	expiry, err := auth.GetExpiryFromJWT(token)
	assert.Nil(t, err)
	assert.NotZero(t, expiry)
}

func TestPopulateTokenClientsSymlink(t *testing.T) {
	authConf := getAuthConfig()
	tokMgr := auth.NewTokenManager(authConf)
	assert.NotNil(t, tokMgr)

	err := os.MkdirAll(testAuthAccessTokenPath+"na", 0755)
	assert.Nil(t, err)
	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, testAuthAccessTokenPath+"na"+"/actual_access_token")
	assert.Nil(t, err)
	err = os.Symlink(testAuthAccessTokenPath+"na"+"/actual_access_token", testAuthAccessTokenPath+"na"+"/access_token")
	assert.Nil(t, err)
	defer os.RemoveAll(testAuthAccessTokenPath)

	tokMgr.PopulateTokenClients(authConf)
	assert.Empty(t, tokMgr.TokenClients[0].AccessToken)
}

func TestGetNodeAgentTokenSymlink(t *testing.T) {
	authConf := getAuthConfig()

	err := os.MkdirAll(testAuthAccessTokenPath+"node-agent", 0755)
	assert.Nil(t, err)
	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, testAuthAccessTokenPath+"node-agent"+"/actual_access_token")
	assert.Nil(t, err)
	err = os.Symlink(testAuthAccessTokenPath+"node-agent"+"/actual_access_token", testAuthAccessTokenPath+"node-agent"+"/access_token")
	assert.Nil(t, err)
	defer os.RemoveAll(testAuthAccessTokenPath)

	token = auth.GetNodeAgentToken(authConf)
	assert.Empty(t, token)
}

func TestGetNodeAgentToken(t *testing.T) {
	authConf := getAuthConfig()
	err := os.MkdirAll(testAuthAccessTokenPath+"node-agent", 0755)
	assert.Nil(t, err)

	token, err := testutil.GenerateJWT()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	err = auth.PersistToken(token, filepath.Join(testAuthAccessTokenPath, "node-agent", config.AccessToken))
	assert.Nil(t, err)
	na_token := auth.GetNodeAgentToken(authConf)
	defer os.RemoveAll(testAuthAccessTokenPath)
	assert.NotEmpty(t, na_token)
}
