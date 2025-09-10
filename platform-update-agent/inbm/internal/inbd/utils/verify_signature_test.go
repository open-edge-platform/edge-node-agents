/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to generate a test RSA key and self-signed cert
func generateTestCertAndKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	// Create test certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Intel Test"},
			Country:      []string{"US"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	// Save certificate using file_service.go function
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return priv, certPEM
}

// Helper to generate a signature for a file (SHA384)
func generateSignature(t *testing.T, priv *rsa.PrivateKey, content []byte) string {
	hasher := sha512.New384()
	hasher.Write(content)
	checksum := hasher.Sum(nil)
	checksumHex := hex.EncodeToString(checksum)
	checksumBytes := []byte(checksumHex)

	signHasher := sha512.New384()
	signHasher.Write(checksumBytes)
	hashed := signHasher.Sum(nil)

	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA384, hashed, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto})
	require.NoError(t, err)
	return hex.EncodeToString(sig)
}

func createTestTarFile(fs afero.Fs, tarPath string, files map[string][]byte) error {
	tarFile, err := fs.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	for fileName, content := range files {
		// Check for potential int64 overflow
		if uint64(len(content)) > uint64(math.MaxInt64) {
			return fmt.Errorf("file %s is too large to be added to tar", fileName)
		}
		header := &tar.Header{
			Name: fileName,
			Mode: 0644,
			Size: int64(len(content)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if _, err := tarWriter.Write(content); err != nil {
			return err
		}
	}

	return nil
}

// Test cases
func TestIsValidPackageFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"package.rpm", true},
		{"package.deb", true},
		{"firmware.fv", true},
		{"capsule.cap", true},
		{"bios.bio", true},
		{"binary.bin", true},
		{"config.conf", true},
		{"update.mender", true},
		{"intel_manageability.conf", true}, // Exact match
		{"PACKAGE.RPM", true},              // Case insensitive
		{"invalid.txt", false},
		{"malicious.exe", false},
		{"", false},
		{"no-extension", false},
		{"my_intel_manageability.conf", false}, // Should be blocked
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := isValidPackageFile(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	fs := afero.NewMemMapFs()
	testContent := []byte("test file content")
	testFile := "/tmp/test_file.conf"

	err := WriteFile(fs, testFile, testContent, 0644)
	require.NoError(t, err)

	t.Run("Default SHA384", func(t *testing.T) {
		checksum, err := calculateFileChecksum(fs, testFile, nil)
		require.NoError(t, err)
		assert.Equal(t, 96, len(checksum)) // SHA384 hex = 96 chars
	})

	t.Run("SHA256", func(t *testing.T) {
		alg := SHA256
		checksum, err := calculateFileChecksum(fs, testFile, &alg)
		require.NoError(t, err)
		assert.Equal(t, 64, len(checksum)) // SHA256 hex = 64 chars
	})

	t.Run("SHA384", func(t *testing.T) {
		alg := SHA384
		checksum, err := calculateFileChecksum(fs, testFile, &alg)
		require.NoError(t, err)
		assert.Equal(t, 96, len(checksum)) // SHA384 hex = 96 chars
	})

	t.Run("SHA512", func(t *testing.T) {
		alg := SHA512
		checksum, err := calculateFileChecksum(fs, testFile, &alg)
		require.NoError(t, err)
		assert.Equal(t, 128, len(checksum)) // SHA512 hex = 128 chars
	})

	t.Run("File not found", func(t *testing.T) {
		_, err := calculateFileChecksum(fs, "/nonexistent/file", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not read file content")
	})
}

func TestValidateTarFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("Valid tar file", func(t *testing.T) {
		tarPath := "/tmp/valid.tar"
		files := map[string][]byte{
			"intel_manageability.conf": []byte("config content"),
			"update.rpm":               []byte("rpm content"),
			"update.deb":               []byte("deb content"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.NoError(t, err)
	})

	t.Run("Invalid file in tar", func(t *testing.T) {
		tarPath := "/tmp/invalid.tar"
		files := map[string][]byte{
			"intel_manageability.conf": []byte("config content"),
			"malicious.exe":            []byte("exe content"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file in tarball: 'malicious.exe'. only valid configuration files, PEM certificates, and package files are allowed")
	})

	t.Run("Multiple config files", func(t *testing.T) {
		tarPath := "/tmp/multi_conf.tar"
		files := map[string][]byte{
			"intel_manageability.conf":        []byte("config1"),
			"subdir/intel_manageability.conf": []byte("config2"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "multiple configuration files found in tarball")
	})

	t.Run("Only PEM file in tar", func(t *testing.T) {
		tarPath := "/tmp/onlypem.tar"
		files := map[string][]byte{
			"cert.pem": []byte("-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqh..."),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PEM certificate: failed to decode PEM content")
	})

	t.Run("Empty tar file", func(t *testing.T) {
		tarPath := "/tmp/empty.tar"
		files := map[string][]byte{}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid package found in tarball")
	})

	t.Run("Tar file does not exist", func(t *testing.T) {
		err := validateTarFile(fs, "/tmp/nonexistent/file.tar")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tar file does not exist")
	})

	t.Run("Config file with wrong name", func(t *testing.T) {
		tarPath := "/tmp/wrongname.tar"
		files := map[string][]byte{
			"my_intel_manageability.conf": []byte("bad config"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file in tarball: 'my_intel_manageability.conf'. only valid configuration files, PEM certificates, and package files are allowed")
	})

	t.Run("Config file in subdir only", func(t *testing.T) {
		tarPath := "/tmp/subdir_conf.tar"
		files := map[string][]byte{
			"subdir/intel_manageability.conf": []byte("config content"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.NoError(t, err)
	})

	t.Run("Config file and unrelated .conf file", func(t *testing.T) {
		tarPath := "/tmp/unrelated_conf.tar"
		files := map[string][]byte{
			"intel_manageability.conf": []byte("config content"),
			"other.conf":               []byte("other config"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.NoError(t, err)
	})

	t.Run("Config file and invalid extension", func(t *testing.T) {
		tarPath := "/tmp/invalid_ext.tar"
		files := map[string][]byte{
			"intel_manageability.conf": []byte("config content"),
			"malicious.exe":            []byte("bad"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file in tarball: 'malicious.exe'. only valid configuration files, PEM certificates, and package files are allowed")
	})

	t.Run("Path traversal attack", func(t *testing.T) {
		tarPath := "/tmp/traversal.tar"
		files := map[string][]byte{
			"../../../etc/passwd": []byte("hacked"),
		}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		err = validateTarFile(fs, tarPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal detected")
	})
}

func TestVerifyChecksumWithKey(t *testing.T) {
	// Generate test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	testContent := []byte("test content for signing")

	t.Run("Valid signature", func(t *testing.T) {
		signature := generateSignature(t, privateKey, testContent)

		// Calculate checksum as the verification function expects
		hasher := sha512.New384()
		hasher.Write(testContent)
		checksum := hasher.Sum(nil)
		checksumHex := hex.EncodeToString(checksum)
		checksumBytes := []byte(checksumHex)

		err := verifyChecksumWithKey(context.Background(), &privateKey.PublicKey, signature, checksumBytes)
		assert.NoError(t, err)
	})

	t.Run("Invalid signature", func(t *testing.T) {
		invalidSignature := "invalid_hex_signature"
		checksum := []byte("test checksum")

		err := verifyChecksumWithKey(context.Background(), &privateKey.PublicKey, invalidSignature, checksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature format")
	})

	t.Run("Empty checksum", func(t *testing.T) {
		err := verifyChecksumWithKey(context.Background(), &privateKey.PublicKey, "validhex", []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum")
	})

	t.Run("Empty signature", func(t *testing.T) {
		checksum := []byte("test checksum")
		err := verifyChecksumWithKey(context.Background(), &privateKey.PublicKey, "", checksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature")
	})

	t.Run("Small key size", func(t *testing.T) {
		smallKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		checksum := []byte("test checksum")
		signature := "deadbeef"

		err = verifyChecksumWithKey(context.Background(), &smallKey.PublicKey, signature, checksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid key size")
	})
}

func TestVerifySignature_DirectFile(t *testing.T) {
	// Skip test if we can't create system directories (running without root)
	if os.Getuid() != 0 {
		t.Skip("Skipping test that requires root access to create /etc/intel-manageability")
	}

	// Use the real OS filesystem because VerifySignature uses afero.NewOsFs()
	fs := afero.NewOsFs()
	priv, certPEM := generateTestCertAndKey(t)
	err := MkdirAll(fs, "/etc/intel-manageability/public", 0755)
	require.NoError(t, err)
	err = WriteFile(fs, OTAPackageCertPath, certPEM, 0644)
	require.NoError(t, err)
	content := []byte("test config content")
	confPath := "/tmp/intel_manageability.conf"
	err = WriteFile(fs, confPath, content, 0644)
	require.NoError(t, err)
	sig := generateSignature(t, priv, content)

	err = VerifySignature(sig, confPath, nil)
	assert.NoError(t, err)
	// Cleanup
	_ = fs.Remove(confPath)
	_ = fs.Remove(OTAPackageCertPath)
}

func TestVerifySignature_TarWithPEM(t *testing.T) {
	fs := afero.NewOsFs()
	priv, certPEM := generateTestCertAndKey(t)

	content := []byte("tar config content")
	confName := ConfigFileName

	tarPath := "/tmp/test_package.tar"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: confName, Mode: 0644, Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "testcert.pem", Mode: 0644, Size: int64(len(certPEM)),
	}))
	_, err = tw.Write(certPEM)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	// Signature must be over the tarball bytes
	sig := generateSignature(t, priv, buf.Bytes())

	require.NoError(t, WriteFile(fs, tarPath, buf.Bytes(), 0644))

	err = VerifySignature(sig, tarPath, nil)
	assert.NoError(t, err)
	// Cleanup
	_ = fs.Remove(tarPath)
}

func TestVerifySignature_TarWithSystemPEM(t *testing.T) {
	// Skip test if we can't create system directories (running without root)
	if os.Getuid() != 0 {
		t.Skip("Skipping test that requires root access to create /etc/intel-manageability")
	}

	fs := afero.NewOsFs()
	priv, certPEM := generateTestCertAndKey(t)

	// Write OTA cert to system path
	err := MkdirAll(fs, "/etc/intel-manageability/public", 0755)
	require.NoError(t, err)
	err = WriteFile(fs, OTAPackageCertPath, certPEM, 0644)
	require.NoError(t, err)

	content := []byte("tar config content 2")
	confName := ConfigFileName

	tarPath := "/tmp/test_package2.tar"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: confName, Mode: 0644, Size: int64(len(content)),
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	// Signature must be over the tarball bytes
	sig := generateSignature(t, priv, buf.Bytes())

	require.NoError(t, WriteFile(fs, tarPath, buf.Bytes(), 0644))

	err = VerifySignature(sig, tarPath, nil)
	assert.NoError(t, err)
	// Cleanup
	_ = fs.Remove(tarPath)
	_ = fs.Remove(OTAPackageCertPath)
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	// Skip test if we can't create system directories (running without root)
	if os.Getuid() != 0 {
		t.Skip("Skipping test that requires root access to create /etc/intel-manageability")
	}

	fs := afero.NewOsFs()
	_, certPEM := generateTestCertAndKey(t)

	err := MkdirAll(fs, "/etc/intel-manageability/public", 0755)
	require.NoError(t, err)
	err = WriteFile(fs, OTAPackageCertPath, certPEM, 0644)
	require.NoError(t, err)

	content := []byte("test config content")
	confPath := "/tmp/intel_manageability.conf"
	err = WriteFile(fs, confPath, content, 0644)
	require.NoError(t, err)

	badSig := "deadbeef"
	err = VerifySignature(badSig, confPath, nil)
	assert.Error(t, err)
	// Cleanup
	_ = fs.Remove(confPath)
	_ = fs.Remove(OTAPackageCertPath)
}

func TestVerifySignature_EmptySignatureWhenRequired(t *testing.T) {
	// Skip test if we can't create system directories (running without root)
	if os.Getuid() != 0 {
		t.Skip("Skipping test that requires root access to create /etc/intel-manageability")
	}

	fs := afero.NewOsFs()
	_, certPEM := generateTestCertAndKey(t)
	err := MkdirAll(fs, "/etc/intel-manageability/public", 0755)
	require.NoError(t, err)
	err = WriteFile(fs, OTAPackageCertPath, certPEM, 0644)
	require.NoError(t, err)
	content := []byte("test config content")
	confPath := "/tmp/intel_manageability.conf"
	err = WriteFile(fs, confPath, content, 0644)
	require.NoError(t, err)

	// Empty signature, but cert exists, so signature is required
	err = VerifySignature("", confPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature is required")
	// Cleanup
	_ = fs.Remove(confPath)
	_ = fs.Remove(OTAPackageCertPath)
}

func TestVerifySignature_EmptySignatureWhenNotRequired(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("test config content")
	confPath := "/tmp/intel_manageability.conf"
	err := WriteFile(fs, confPath, content, 0644)
	require.NoError(t, err)

	// Remove OTA package cert if it exists
	_ = fs.Remove(OTAPackageCertPath)

	// Empty signature, and no OTA cert, so signature is not required
	err = VerifySignature("", confPath, nil)
	assert.NoError(t, err)
	_ = fs.Remove(confPath)
}

func TestVerifySignature_UnsupportedFileFormat(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("test content")
	badPath := "/tmp/file.exe"
	err := WriteFile(fs, badPath, content, 0644)
	require.NoError(t, err)

	// Even with a signature, should fail due to unsupported file format
	priv, _ := generateTestCertAndKey(t)
	sig := generateSignature(t, priv, content)
	err = VerifySignature(sig, badPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file format")
	_ = fs.Remove(badPath)
}

func TestVerifySignature_MissingCertificateFile(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("test config content")
	confPath := "/tmp/intel_manageability.conf"
	err := WriteFile(fs, confPath, content, 0644)
	require.NoError(t, err)

	// Remove OTA package cert if it exists
	_ = fs.Remove(OTAPackageCertPath)

	priv, _ := generateTestCertAndKey(t)
	sig := generateSignature(t, priv, content)
	err = VerifySignature(sig, confPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate file does not exist")
	_ = fs.Remove(confPath)
}

func TestVerifySignature_TarWithInvalidPEM(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("tar config content")
	confName := ConfigFileName
	badPEM := []byte("not a valid pem")

	tarPath := "/tmp/test_badpem.tar"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: confName, Mode: 0644, Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "badcert.pem", Mode: 0644, Size: int64(len(badPEM)),
	}))
	_, err = tw.Write(badPEM)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	require.NoError(t, WriteFile(fs, tarPath, buf.Bytes(), 0644))

	priv, _ := generateTestCertAndKey(t)
	// Generate signature over the entire tarball, not just the config content
	sig := generateSignature(t, priv, buf.Bytes())

	err = VerifySignature(sig, tarPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM certificate")
	_ = fs.Remove(tarPath)
}

func TestVerifySignature_TarWithMultipleConfigs(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("tar config content")
	confName := ConfigFileName

	tarPath := "/tmp/test_multiconf.tar"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: confName, Mode: 0644, Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "subdir/" + confName, Mode: 0644, Size: int64(len(content)),
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	require.NoError(t, WriteFile(fs, tarPath, buf.Bytes(), 0644))

	priv, _ := generateTestCertAndKey(t)
	// Generate signature over the entire tarball, not just the config content
	sig := generateSignature(t, priv, buf.Bytes())

	err = VerifySignature(sig, tarPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple configuration files found in tarball")
	_ = fs.Remove(tarPath)
}

func TestPathValidation(t *testing.T) {
	t.Run("Path validation tests", func(t *testing.T) {
		// Test absolute paths
		absolutePaths := []string{
			"/tmp/file.rpm",
			"/home/user/package.deb",
			"/tmp/config.conf",
		}

		for _, path := range absolutePaths {
			assert.True(t, filepath.IsAbs(path), "Path should be absolute: %s", path)
		}

		// Test relative paths
		relativePaths := []string{
			"relative/file.rpm",
			"./local/package.deb",
			"../parent/config.conf",
		}

		for _, path := range relativePaths {
			assert.False(t, filepath.IsAbs(path), "Path should be relative: %s", path)
		}
	})
}

func TestHashAlgorithmConstants(t *testing.T) {
	assert.Equal(t, HashAlgorithm(256), SHA256)
	assert.Equal(t, HashAlgorithm(384), SHA384)
	assert.Equal(t, HashAlgorithm(512), SHA512)
}

func TestSignatureGeneration(t *testing.T) {
	// Test the signature generation logic used in tests
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	testContent := []byte("test content")

	signature1 := generateSignature(t, privateKey, testContent)
	assert.NotEmpty(t, signature1)

	signature2 := generateSignature(t, privateKey, testContent)
	assert.NotEmpty(t, signature2)

	// Signatures should be different due to random salt in PSS
	assert.NotEqual(t, signature1, signature2, "PSS signatures should be different due to random salt")

	// But both should be valid hex strings
	_, err = hex.DecodeString(signature1)
	assert.NoError(t, err, "Signature 1 should be valid hex")

	_, err = hex.DecodeString(signature2)
	assert.NoError(t, err, "Signature 2 should be valid hex")
}

func TestParseHashAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected HashAlgorithm
	}{
		{"sha256", SHA256},
		{"SHA256", SHA256},
		{"sha512", SHA512},
		{"SHA512", SHA512},
		{"sha384", SHA384},
		{"SHA384", SHA384},
		{"unknown", SHA384}, // Default
		{"", SHA384},        // Default
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ParseHashAlgorithm(test.input)
			assert.Equal(t, test.expected, *result)
		})
	}
}

func TestIsPEMFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"cert.pem", true},
		{"certificate.crt", true},
		{"public.cert", true},
		{"CERT.PEM", true}, // Case insensitive
		{"config.conf", false},
		{"package.rpm", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := isPEMFile(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestValidateTarFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{"valid filename", "config.conf", false, ""},
		{"empty filename", "", true, "empty filename"},
		{"null bytes", "bad\x00file.conf", true, "filename contains null bytes"},
		{"too long filename", strings.Repeat("a", 256) + ".conf", true, "filename too long"},
		{"carriage return", "bad\rfile.conf", true, "filename contains dangerous character"},
		{"newline", "bad\nfile.conf", true, "filename contains dangerous character"},
		{"tab", "bad\tfile.conf", true, "filename contains dangerous character"},
		{"normal path", "subdir/file.conf", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTarFilename(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTarPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	safeDir := "/safe/dir"

	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{"valid relative path", "config.conf", false, ""},
		{"valid subdirectory", "subdir/config.conf", false, ""},
		{"absolute path", "/etc/passwd", true, "absolute paths not allowed"},
		{"path traversal dots", "../../../etc/passwd", true, "path traversal detected"},
		{"path traversal mixed", "good/../../../bad", true, "path traversal detected"},
		{"current directory", "./config.conf", false, ""},
		{"empty filename", "", false, ""}, // Empty gets cleaned to "."
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTarPath(fs, tt.filename, safeDir)
			if tt.wantErr {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTarFileSize(t *testing.T) {
	tests := []struct {
		name    string
		header  *tar.Header
		wantErr bool
		errMsg  string
	}{
		{
			"valid size",
			&tar.Header{Name: "file.conf", Size: 1024},
			false, "",
		},
		{
			"negative size",
			&tar.Header{Name: "file.conf", Size: -1},
			true, "invalid file size",
		},
		{
			"too large file",
			&tar.Header{Name: "huge.bin", Size: 200 * 1024 * 1024}, // 200MB
			true, "file too large",
		},
		{
			"large PEM file",
			&tar.Header{Name: "huge.pem", Size: 128 * 1024}, // 128KB PEM
			true, "PEM file too large",
		},
		{
			"valid PEM file",
			&tar.Header{Name: "cert.pem", Size: 4096}, // 4KB PEM
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTarFileSize(tt.header)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePEMContent(t *testing.T) {
	_, validPEM := generateTestCertAndKey(t)

	// Create a valid PEM structure but with invalid certificate data
	invalidCertPEM := `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAinvalidcertificatedata==
-----END CERTIFICATE-----`

	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{"valid PEM", string(validPEM), false, ""},
		{"empty content", "", true, "empty PEM content"},
		{"invalid PEM structure", "not a pem certificate", true, "failed to decode PEM content"},
		{"invalid certificate data", invalidCertPEM, true, "failed to parse X.509 certificate"},
		{"missing BEGIN block", "-----END CERTIFICATE-----\n", true, "failed to decode PEM content"},
		{"missing END block", "-----BEGIN CERTIFICATE-----\ndata\n", true, "failed to decode PEM content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePEMContent(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReadTarContentWithLimit(t *testing.T) {
	// Create a tar with content to test reading limits
	fs := afero.NewMemMapFs()

	t.Run("content within limit", func(t *testing.T) {
		content := []byte("small content")
		tarPath := "/tmp/small.tar"
		files := map[string][]byte{"file.txt": content}

		err := createTestTarFile(fs, tarPath, files)
		require.NoError(t, err)

		tarFile, err := fs.Open(tarPath)
		require.NoError(t, err)
		defer tarFile.Close()

		tarReader := tar.NewReader(tarFile)
		header, err := tarReader.Next()
		require.NoError(t, err)

		result, err := readTarContentWithLimit(tarReader, header.Size, 1024)
		assert.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("declared size exceeds limit", func(t *testing.T) {
		// Test when declared size is larger than limit
		var mockReader tar.Reader // Won't actually read, just test validation
		_, err := readTarContentWithLimit(&mockReader, 2048, 1024)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "declared size 2048 exceeds limit 1024")
	})
}
