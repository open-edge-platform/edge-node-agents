// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"
)

// CreateCertificateAndKey generates a self-signed test certificate and private key, and returns them in PEM format.
func CreateCertificateAndKey() (testCertContents []byte, testKeyContents []byte, err error) {
	testPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	testCa := &x509.Certificate{
		SerialNumber: big.NewInt(2023),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		DNSNames:              []string{"test.dummy.unit"},
	}

	testCaBytes, err := x509.CreateCertificate(rand.Reader, testCa, testCa, &testPrivKey.PublicKey, testPrivKey)
	if err != nil {
		return nil, nil, err
	}

	testCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaBytes,
	}
	testCertContents = pem.EncodeToMemory(testCert)

	testKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testPrivKey),
	}
	testKeyContents = pem.EncodeToMemory(testKey)

	return testCertContents, testKeyContents, nil
}

// generateCertificateAndKey generates a certificate and key for internal use and returns an error if any step fails.
func generateCertificateAndKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	testCertContents, testKeyContents, err := CreateCertificateAndKey()
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(testCertContents)
	keyBlock, _ := pem.Decode(testKeyContents)
	if certBlock == nil || keyBlock == nil {
		return nil, nil, errors.New("failed to decode PEM block for certificate or key")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// CreateTestCertificate generates a test certificate, key, and CA certificate, writes them to temporary files, and returns their paths.
func CreateTestCertificate() (certPath string, keyPath string, caPath string, err error) {
	testCaCert, testCaKey, err := generateCertificateAndKey()
	if err != nil {
		return "", "", "", err
	}
	testCaBytes, err := x509.CreateCertificate(rand.Reader, testCaCert, testCaCert, &testCaKey.PublicKey, testCaKey)
	if err != nil {
		return "", "", "", err
	}
	testCaCertBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaBytes,
	}
	testCaCertContents := pem.EncodeToMemory(testCaCertBlock)

	testCaKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testCaKey),
	}
	testCaKeyContents := pem.EncodeToMemory(testCaKeyBlock)

	certFile, err := os.CreateTemp("", "cert.pem")
	if err != nil {
		return "", "", "", err
	}
	defer certFile.Close()

	_, err = certFile.Write(testCaCertContents)
	if err != nil {
		return "", "", "", err
	}

	keyFile, err := os.CreateTemp("", "key.pem")
	if err != nil {
		return "", "", "", err
	}
	defer keyFile.Close()

	_, err = keyFile.Write(testCaKeyContents)
	if err != nil {
		return "", "", "", err
	}

	caCertBytes := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaCert.Raw,
	}
	pemData := pem.EncodeToMemory(caCertBytes)
	caFile, err := os.CreateTemp("", "cacert.pem")
	if err != nil {
		return "", "", "", err
	}
	defer caFile.Close()

	_, err = caFile.Write(pemData)
	if err != nil {
		return "", "", "", err
	}

	return certFile.Name(), keyFile.Name(), caFile.Name(), nil
}

// CreateTestCertificates generates a test certificate and key, writes them to temporary files, and returns their paths.
func CreateTestCertificates() (certPath string, keyPath string, err error) {
	testCertContents, testKeyContents, err := CreateCertificateAndKey()
	if err != nil {
		return "", "", err
	}

	certFile, err := os.CreateTemp("", "cert.pem")
	if err != nil {
		return "", "", err
	}
	defer certFile.Close()

	_, err = certFile.Write(testCertContents)
	if err != nil {
		return "", "", err
	}

	keyFile, err := os.CreateTemp("", "key.pem")
	if err != nil {
		return "", "", err
	}
	defer keyFile.Close()

	_, err = keyFile.Write(testKeyContents)
	if err != nil {
		return "", "", err
	}

	return certFile.Name(), keyFile.Name(), nil
}

// CreateTestCrt generates a test certificate and key, writes them to temporary files, and returns their paths and the certificate contents.
func CreateTestCrt() (certPath string, keyPath string, certContents []byte, err error) {
	testCa, testPrivKey, err := generateCertificateAndKey()
	if err != nil {
		return "", "", nil, err
	}

	testCaBytes, err := x509.CreateCertificate(rand.Reader, testCa, testCa, &testPrivKey.PublicKey, testPrivKey)
	if err != nil {
		return "", "", nil, err
	}

	testCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaBytes,
	}

	testCertContents := pem.EncodeToMemory(testCert)

	testKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testPrivKey),
	}

	testKeyContents := pem.EncodeToMemory(testKey)

	certFile, err := os.CreateTemp("", "cert.pem")
	if err != nil {
		return "", "", nil, err
	}
	defer certFile.Close()

	_, err = certFile.Write(testCertContents)
	if err != nil {
		return "", "", nil, err
	}

	keyFile, err := os.CreateTemp("", "key.pem")
	if err != nil {
		return "", "", nil, err
	}
	defer keyFile.Close()

	_, err = keyFile.Write(testKeyContents)
	if err != nil {
		return "", "", nil, err
	}

	return certFile.Name(), keyFile.Name(), testCertContents, nil
}
