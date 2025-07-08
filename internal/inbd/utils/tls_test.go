/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package utils provides utility functions.
package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// cleanupFiles removes multiple files and logs any errors
func cleanupFiles(t *testing.T, fs afero.Fs, files []string) {
	for _, file := range files {
		if err := RemoveFile(fs, file); err != nil {
			t.Logf("Warning: failed to remove file %s: %v", file, err)
		}
	}
}

func TestTLSDirConstants(t *testing.T) {
	if TLSDirSecret == "" || TLSDirPublic == "" {
		t.Error("TLSDirSecret or TLSDirPublic should not be empty")
	}
}

func TestGenerateLocalCA(t *testing.T) {
	caCert, caKey, err := GenerateLocalCA()
	if err != nil {
		t.Fatalf("GenerateLocalCA failed: %v", err)
	}
	if caCert == nil || caKey == nil {
		t.Error("CA cert or key is nil")
	}
	// cleanup
	fs := afero.NewOsFs()
	cleanupFiles(t, fs, []string{
		filepath.Join(TLSDirSecret, "ca.key"),
		filepath.Join(TLSDirSecret, "ca.crt"),
		filepath.Join(TLSDirPublic, "ca.pub"),
	})
}

func TestGenerateAndSignCert(t *testing.T) {
	caCert, caKey, err := GenerateLocalCA()
	if err != nil {
		t.Fatalf("GenerateLocalCA failed: %v", err)
	}
	err = GenerateAndSignCert("testsvc", caCert, caKey)
	if err != nil {
		t.Fatalf("GenerateAndSignCert failed: %v", err)
	}
	// cleanup
	fs := afero.NewOsFs()
	cleanupFiles(t, fs, []string{
		filepath.Join(TLSDirSecret, "testsvc.key"),
		filepath.Join(TLSDirSecret, "testsvc.crt"),
		filepath.Join(TLSDirPublic, "testsvc.pub"),
		filepath.Join(TLSDirSecret, "ca.key"),
		filepath.Join(TLSDirSecret, "ca.crt"),
		filepath.Join(TLSDirPublic, "ca.pub"),
	})
}

func TestSetupTLSCertificates(t *testing.T) {
	// Remove ca.crt if exists to force regeneration
	caCrtPath := filepath.Join(TLSDirSecret, "ca.crt")
	fs := afero.NewOsFs()
	if err := RemoveFile(fs, caCrtPath); err != nil {
		t.Logf("Warning: failed to remove ca.crt: %v", err)
	}

	// Test successful SetupTLSCertificates execution
	err := SetupTLSCertificates()
	if err != nil {
		t.Fatalf("SetupTLSCertificates failed: %v", err)
	}
	if _, err := os.Stat(caCrtPath); err != nil {
		t.Errorf("ca.crt not created: %v", err)
	}

	// Test early return when ca.crt already exists
	err = SetupTLSCertificates()
	if err != nil {
		t.Fatalf("SetupTLSCertificates failed on early return: %v", err)
	}

	// cleanup
	cleanupFiles(t, fs, []string{
		filepath.Join(TLSDirSecret, "ca.key"),
		filepath.Join(TLSDirSecret, "ca.crt"),
		filepath.Join(TLSDirSecret, "inbc.key"),
		filepath.Join(TLSDirSecret, "inbc.crt"),
		filepath.Join(TLSDirSecret, "inbd.key"),
		filepath.Join(TLSDirSecret, "inbd.crt"),
		filepath.Join(TLSDirPublic, "ca.pub"),
		filepath.Join(TLSDirPublic, "inbc.pub"),
		filepath.Join(TLSDirPublic, "inbd.pub"),
	})
}
