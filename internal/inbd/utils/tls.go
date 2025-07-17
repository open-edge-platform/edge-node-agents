/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/afero"
)

// Directories to store keys/certs
const (
	TLSDirPublic = "/etc/intel-manageability/public"
)

// getTLSDirSecret returns the TLS secret directory path from configuration
var (
	tlsDirSecretCache string
	tlsDirSecretOnce  sync.Once
)

func getTLSDirSecret() string {
	tlsDirSecretOnce.Do(func() {
		config, err := LoadConfig(afero.NewOsFs(), ConfigFilePath)
		if err != nil {
			// Fallback to default if config can't be loaded
			tlsDirSecretCache = "/etc/intel-manageability/secret"
			return
		}
		// Use LUKS mount point if available, otherwise use default
		if config.LUKS.MountPoint != "" {
			tlsDirSecretCache = config.LUKS.MountPoint
		} else {
			tlsDirSecretCache = "/etc/intel-manageability/secret"
		}
	})
	return tlsDirSecretCache
}

// GetTLSDirSecret returns the TLS secret directory path from configuration
func GetTLSDirSecret() string {
	return getTLSDirSecret()
}

// GenerateLocalCA generates a local CA key and certificate.
func GenerateLocalCA() (caCert *x509.Certificate, caKey *rsa.PrivateKey, err error) {
	caKey, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}
	caCertTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"Oregon"},
			Locality:           []string{"Hillsboro"},
			Organization:       []string{"Intel"},
			OrganizationalUnit: []string{"ECG"},
			CommonName:         "INBM-CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SignatureAlgorithm:    x509.SHA384WithRSA, // Enforce SHA-384
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caCertTmpl, caCertTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}
	caCert, err = x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	// Write CA key/cert: cert to secret, public key to public
	fs := afero.NewOsFs()
	if err := MkdirAll(fs, TLSDirPublic, 0755); err != nil {
		return nil, nil, err
	}
	tlsDirSecret := getTLSDirSecret()
	if err := MkdirAll(fs, tlsDirSecret, 0700); err != nil {
		return nil, nil, err
	}

	if err := writePEM(filepath.Join(tlsDirSecret, "ca.key"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caKey)); err != nil {
		return nil, nil, fmt.Errorf("failed to write CA private key: %w", err)
	}
	if err := writePEM(filepath.Join(tlsDirSecret, "ca.crt"), "CERTIFICATE", caCertDER); err != nil {
		return nil, nil, fmt.Errorf("failed to write CA certificate: %w", err)
	}
	if err := writePublicKey(filepath.Join(TLSDirPublic, "ca.pub"), &caKey.PublicKey); err != nil {
		return nil, nil, fmt.Errorf("failed to write CA public key: %w", err)
	}
	return caCert, caKey, nil
}

// GenerateAndSignCert generates a key and CSR for a service, signs it with the CA, and writes files.
func GenerateAndSignCert(name string, caCert *x509.Certificate, caKey *rsa.PrivateKey) error {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate %s private key: %w", name, err)
	}
	certTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"Oregon"},
			Locality:           []string{"Hillsboro"},
			Organization:       []string{"Intel"},
			OrganizationalUnit: []string{"ECG"},
			CommonName:         name,
		},
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:           []string{"localhost"},
		IPAddresses:        []net.IP{net.ParseIP("127.0.0.1")},
		SignatureAlgorithm: x509.SHA384WithRSA, // Enforce SHA-384
	}
	certDER, err := x509.CreateCertificate(rand.Reader, certTmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create %s certificate: %w", name, err)
	}
	fs := afero.NewOsFs()
	tlsDirSecret := getTLSDirSecret()
	if err := MkdirAll(fs, tlsDirSecret, 0700); err != nil {
		return err
	}
	if err := MkdirAll(fs, TLSDirPublic, 0755); err != nil {
		return err
	}

	if err := writePEM(filepath.Join(tlsDirSecret, name+".key"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)); err != nil {
		return fmt.Errorf("failed to write private key for %s: %w", name, err)
	}
	if err := writePEM(filepath.Join(tlsDirSecret, name+".crt"), "CERTIFICATE", certDER); err != nil {
		return fmt.Errorf("failed to write certificate for '%s': %w", name, err)
	}
	if err := writePublicKey(filepath.Join(TLSDirPublic, name+".pub"), &key.PublicKey); err != nil {
		return fmt.Errorf("failed to write public key for '%s': %w", name, err)
	}
	return nil
}

// writePEM writes PEM-encoded data to a file.
func writePEM(filename, typ string, derBytes []byte) error {
	f, err := OpenFile(afero.NewOsFs(), filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: derBytes})
}

// writePublicKey writes the public key in PEM format.
func writePublicKey(filename string, pub *rsa.PublicKey) error {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	return writePEM(filename, "PUBLIC KEY", pubASN1)
}

// SetupTLSCertificates generates CA, inbc, and inbd keys/certs if not present.
func SetupTLSCertificates() error {
	tlsDirSecret := getTLSDirSecret()
	caCrtPath := filepath.Join(tlsDirSecret, "ca.crt")
	if _, err := os.Stat(caCrtPath); err == nil {
		return nil // Return early if CA certificate already exists
	}
	caCert, caKey, err := GenerateLocalCA()
	if err != nil {
		return fmt.Errorf("failed to generate local CA: %w", err)
	}
	for _, name := range []string{"inbc", "inbd"} {
		if err := GenerateAndSignCert(name, caCert, caKey); err != nil {
			return fmt.Errorf("failed to generate and sign cert for %s: %w", name, err)
		}
	}
	return nil // Return success
}
