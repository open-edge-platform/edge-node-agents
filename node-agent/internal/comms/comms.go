// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comms

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/auth"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
	"golang.org/x/oauth2"
)

const TIMEOUT = 30 * time.Second
const EARLYEXPIRY = 15 * time.Minute

type Client struct {
	HostGUID   string
	BaseUrl    *url.URL
	HttpClient *http.Client
}

type CSRPayload struct {
	Csr []byte `json:"csr"`
}

type CertResponseContent struct {
	Cert string `json:"certificate"`
}

type CertResponsePayload struct {
	Success CertResponseContent `json:"success"`
}

// Initialize logger
var log = logger.Logger

func new(serverUrl string, hostGUID string, tlsConfig *tls.Config, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil || u == nil {
		return nil, fmt.Errorf("url is not valid : %v", err)
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	client := Client{
		HostGUID:   hostGUID,
		BaseUrl:    u,
		HttpClient: &http.Client{Transport: tr, Timeout: timeout},
	}

	return &client, nil
}

func GetAuthCli(idpURL string, guid string, caCertPool *x509.CertPool) (*Client, error) {
	baseEndpoint := fmt.Sprintf("https://%s", idpURL)
	tlsConfig := tls.Config{
		RootCAs: caCertPool,
	}
	authCli, err := new(baseEndpoint, guid, &tlsConfig, TIMEOUT)
	return authCli, err
}

func (cli *Client) ProvisionAccessToken(ctx context.Context, authConf config.ConfigAuth) (oauth2.Token, error) {
	var token oauth2.Token
	clientIdFile := filepath.Join(authConf.ClientCredsPath, "client_id")
	clientId, err := utils.ReadFileNoLinks(clientIdFile)
	if err != nil {
		return token, fmt.Errorf("failed to read client id file: %v", err)
	}
	clientSecretFile := filepath.Join(authConf.ClientCredsPath, "client_secret")
	secret, err := utils.ReadFileNoLinks(clientSecretFile)
	if err != nil {
		return token, fmt.Errorf("failed to read client secret file: %v", err)
	}
	endpoint := cli.BaseUrl.JoinPath("/realms/master/protocol/openid-connect/token")

	payload := url.Values{}
	payload.Set("grant_type", "client_credentials")
	payload.Set("client_id", strings.TrimSpace(string(clientId)))
	payload.Set("client_secret", strings.TrimSpace(string(secret)))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader([]byte(payload.Encode())))
	if err != nil {
		return token, fmt.Errorf("http request creation failed: %v", err)
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := cli.HttpClient.Do(request)
	if err != nil {
		log.Error("failed to get token from IDP service")
		return token, err
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return token, fmt.Errorf("failed to read resp.Body:%v", err)
	}
	var tokenR map[string]interface{}
	err = json.Unmarshal(bodyBytes, &tokenR)
	if err != nil {
		return token, fmt.Errorf("failed to unmarshal response:%v", err)
	}
	at := tokenR[config.AccessToken].(string)
	exp, err := auth.GetExpiryFromJWT(at)
	if err != nil {
		return token, fmt.Errorf("failed to get expiry from token:%v", err)
	}

	log.Infof("token retrieved from IDP successfully")
	return oauth2.Token{AccessToken: at, Expiry: exp}, nil
}

func (cli *Client) ProvisionReleaseServiceToken(ctx context.Context, authConf config.ConfigAuth, accessToken string) (oauth2.Token, error) {
	var token oauth2.Token
	endpoint := cli.BaseUrl.JoinPath("/token")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return token, fmt.Errorf("http request creation failed: %v", err)
	}

	// Add access token in request header
	var bearer = "Bearer " + accessToken

	request.Header.Add("Authorization", bearer)
	response, err := cli.HttpClient.Do(request)
	if err != nil {
		log.Error("failed to get release service token")
		return token, err
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return token, fmt.Errorf("failed to read resp.Body:%v", err)
	}
	if response.StatusCode != http.StatusOK {
		return token, fmt.Errorf("response code: %s", response.Status)
	}
	relToken := string(bodyBytes)

	log.Infoln("release service token retrieved successfully")
	var expiry time.Time
	if relToken == "anonymous" {
		currentTime := time.Now()
		// Add a very long time after first check to avoid checking again
		expiry = currentTime.AddDate(10, 0, 0)
	} else {
		expiry, err = auth.GetExpiryFromJWT(relToken)
		if err != nil {
			return token, fmt.Errorf("failed to parse jwt release token to get expiry :%v", err)
		}
	}
	return oauth2.Token{AccessToken: relToken, Expiry: expiry}, nil
}

func (cli *Client) ProvisionToken(ctx context.Context, authConf config.ConfigAuth, tknClient auth.ClientAuthToken) (oauth2.Token, error) {
	var token oauth2.Token
	var err error
	// provision release service token
	if tknClient.ClientName == "release-service" {
		token, err = cli.ProvisionReleaseServiceToken(ctx, authConf, auth.GetNodeAgentToken(authConf))
		if err != nil {
			log.Errorf("failed to get release service token: %d", err)
			return token, err
		}
	} else {
		// provision orchestrator service token
		token, err = cli.ProvisionAccessToken(ctx, authConf)
		if err != nil {
			log.Errorf("failed to get access token from IDP: %v", err)
			return token, err
		}
	}

	log.Infof("JWT token generated for client %s", tknClient.ClientName)
	err = auth.PersistToken(token.AccessToken, filepath.Join(authConf.AccessTokenPath, tknClient.ClientName, config.AccessToken))
	if err != nil {
		log.Errorf("failed to Persist token to file: %v", err)
		return token, err
	}
	return token, nil
}
