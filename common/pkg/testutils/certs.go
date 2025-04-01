// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func CreateCertificateAndKey() ([]byte, []byte, error) {
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
	testCertContents := pem.EncodeToMemory(testCert)

	testKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testPrivKey),
	}
	testKeyContents := pem.EncodeToMemory(testKey)

	return testCertContents, testKeyContents, nil
}

func generateCertificateAndKey() (*x509.Certificate, *rsa.PrivateKey) {
	testCertContents, testKeyContents, _ := CreateCertificateAndKey()

	certBlock, _ := pem.Decode(testCertContents)
	keyBlock, _ := pem.Decode(testKeyContents)

	cert, _ := x509.ParseCertificate(certBlock.Bytes)
	key, _ := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)

	return cert, key
}

func CreateTestCertificate() (string, string, string) {
	testCaCert, testCaKey := generateCertificateAndKey()
	testCaBytes, _ := x509.CreateCertificate(rand.Reader, testCaCert, testCaCert, &testCaKey.PublicKey, testCaKey)
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

	certFile, _ := os.CreateTemp("", "cert.pem")
	defer certFile.Close()
	_, _ = certFile.Write(testCaCertContents)
	keyFile, _ := os.CreateTemp("", "key.pem")
	defer keyFile.Close()
	_, _ = keyFile.Write(testCaKeyContents)

	caCertBytes := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: testCaCert.Raw,
	}
	pemData := pem.EncodeToMemory(caCertBytes)
	caFile, _ := os.CreateTemp("", "cacert.pem")
	defer caFile.Close()
	_, _ = caFile.Write(pemData)

	return certFile.Name(), keyFile.Name(), caFile.Name()
}

func createCertFiles() (string, string, error) {
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

func CreateTestCertificates() (string, string, error) {
	return createCertFiles()
}

func CreateTestCerts() (string, string) {
	testCertContents, testKeyContents, err := createCertFiles()
	if err != nil {
		return "", ""
	}
	return testCertContents, testKeyContents
}

func createTestCrts() (string, string, []byte) {
	testCa, testPrivKey := generateCertificateAndKey()

	testCaBytes, _ := x509.CreateCertificate(rand.Reader, testCa, testCa, &testPrivKey.PublicKey, testPrivKey)

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

	certFile, _ := os.CreateTemp("", "cert.pem")
	_, _ = certFile.Write(testCertContents)
	_ = certFile.Close()

	keyFile, _ := os.CreateTemp("", "key.pem")
	_, _ = keyFile.Write(testKeyContents)
	_ = keyFile.Close()

	return certFile.Name(), keyFile.Name(), testCertContents
}

func CreateTestCrt() (string, string, []byte) {
	certPath, keyPath, certContents := createTestCrts()
	return certPath, keyPath, certContents
}
