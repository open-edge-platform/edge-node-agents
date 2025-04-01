// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func SetNonRootUser(t *testing.T) int {

	currenteUid := syscall.Geteuid()

	if syscall.Geteuid() < 1000 {
		err := syscall.Seteuid(1000)
		assert.Nil(t, err, fmt.Sprintf("Could not set non root user %v", err))
	}
	return currenteUid
}

func ResetUser(t *testing.T, originalUid int) {

	if syscall.Geteuid() != originalUid {
		err := syscall.Seteuid(originalUid)
		assert.Nil(t, err, fmt.Sprintf("Could not reset user configuration %v", err))
	}
}

func CreateTestCertWithValidity(t *testing.T, start time.Time, end time.Time) string {

	testPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	assert.Nil(t, err)

	testCa := &x509.Certificate{
		SerialNumber: big.NewInt(2023),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		NotBefore:             start,
		NotAfter:              end,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	testCaBytes, err := x509.CreateCertificate(rand.Reader, testCa, testCa, &testPrivKey.PublicKey, testPrivKey)
	assert.Nil(t, err)

	testCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaBytes,
	}

	testCertContents := pem.EncodeToMemory(testCert)
	assert.Nil(t, err)

	return string(testCertContents)
}

func GenerateJWT() (string, error) {
	var jwtSecretKey = []byte("TestJWT")
	token := jwt.New(jwt.SigningMethodHS512)
	claims := token.Claims.(jwt.MapClaims)

	// Add claims for jwt token
	claims["iss"] = "test-na"
	claims["sub"] = "6a0c3a46-8dfb-48f6-94b1-bde83118aa65"
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(10 * time.Minute).Unix()
	claims["typ"] = "bearer"
	tokenStr, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}
