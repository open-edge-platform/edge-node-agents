// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
)

const REFRESH_INTERVAL = 10 * time.Minute

type ClientAuthToken struct {
	ClientName  string
	AccessToken string
	Expiry      time.Time
}

type TokenManager struct {
	TokenClients []ClientAuthToken
}

// Initialize logger
var log = logger.Logger

func PersistToken(token string, tokenFile string) error {

	// Only the node-agent user should be able
	// to access then token

	err := os.WriteFile(tokenFile, []byte(token), 0640) // #nosec G306
	if err != nil {
		log.Error("could not persist token")
		return err
	}

	log.Info("persisted token to file")

	return nil
}

func IsTokenRefreshRequired(tokenExpiry time.Time) bool {
	safeInterval := tokenExpiry.Add(-REFRESH_INTERVAL)
	return time.Now().After(safeInterval)
}

func NewTokenManager(conf config.ConfigAuth) *TokenManager {
	tknMgr := TokenManager{TokenClients: make([]ClientAuthToken, len(conf.TokenClients))}

	for i := 0; i < len(conf.TokenClients); i++ {
		tknMgr.TokenClients[i] = ClientAuthToken{ClientName: conf.TokenClients[i]}
	}
	return &tknMgr
}

func (tknMgr *TokenManager) PopulateTokenClients(conf config.ConfigAuth) {
	for i, client := range tknMgr.TokenClients {
		tPath := filepath.Join(conf.AccessTokenPath, client.ClientName, config.AccessToken)
		log.Infof("path %s", tPath)
		tokenData, err := utils.ReadFileNoLinks(tPath)
		if err != nil {
			log.Errorf("failed to read persistent JWT token: %v", err)
			continue
		}
		tknMgr.TokenClients[i].AccessToken = string(tokenData)
		expiry, err := GetExpiryFromJWT(string(tokenData))
		if err != nil {
			log.Errorf("Failed to get expiry from JWT token: %v", err)
			os.Exit(1)
		}
		tknMgr.TokenClients[i].Expiry = expiry
	}
}

func GetExpiryFromJWT(jwtTokenStr string) (time.Time, error) {
	var exp time.Time
	parser := &jwt.Parser{}
	token, _, err := parser.ParseUnverified(jwtTokenStr, jwt.MapClaims{})
	if err != nil {
		fmt.Println(err)
		return exp, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); !ok {
		fmt.Println(err)
		return exp, err
	} else {
		exp = time.Unix(int64(claims["exp"].(float64)), 0)
	}
	return exp, nil
}

func GetNodeAgentToken(confs config.ConfigAuth) string {
	tokenFile := filepath.Join(confs.AccessTokenPath, "node-agent", config.AccessToken)
	tBytes, _ := utils.ReadFileNoLinks(tokenFile)
	return strings.TrimSpace(string(tBytes))
}
