/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"archive/tar"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// HashAlgorithm represents the supported hash algorithms
type HashAlgorithm int

const (
	SHA256 HashAlgorithm = 256
	SHA384 HashAlgorithm = 384
	SHA512 HashAlgorithm = 512
)

// TarContents holds extracted tar file contents
type TarContents struct {
	ConfigFile   string
	PEMCert      string
	HasConfig    bool
	HasPEM       bool
	PackageFiles []string
}

// VerifySignature verifies that the signed checksum of the package matches the package received
// Generic function that supports configuration, FOTA, and SOTA packages with embedded PEM certificates
func VerifySignature(signature, pathToFile string, hashAlgorithm *HashAlgorithm) error {
	fs := afero.NewOsFs()

	if err := isFilePathAbsolute(pathToFile); err != nil {
		return fmt.Errorf("signature check failed: invalid path: %w", err)
	}

	if err := isFilePathSymLink(pathToFile); err != nil {
		return fmt.Errorf("signature check failed: path is symlink: %w", err)
	}

	// Get file extension
	extension := getFileExtension(pathToFile)
	var certPath string

	// Handle tar files containing packages and optional PEM certificates
	if strings.ToLower(extension) == "tar" {
		tarContents, err := extractAndValidateTarContents(fs, pathToFile)
		if err != nil {
			return fmt.Errorf("signature check failed: %w", err)
		}

		// Use embedded PEM certificate if available (priority over system cert)
		if tarContents.HasPEM {
			// Create temporary certificate file
			tempCertFile := filepath.Join(filepath.Dir(pathToFile), "temp_cert.pem")
			if err := WriteFile(fs, tempCertFile, []byte(tarContents.PEMCert), 0644); err != nil {
				return fmt.Errorf("signature check failed: could not write temporary certificate: %w", err)
			}
			defer func() {
				if err := fs.Remove(tempCertFile); err != nil {
					log.Printf("Warning: failed to remove temporary certificate file %s: %v", tempCertFile, err)
				}
			}()
			certPath = tempCertFile
		} else {
			// Fallback to system certificate
			if !IsFileExist(fs, OTAPackageCertPath) {
				return fmt.Errorf("signature check failed: system certificate not found at %s", OTAPackageCertPath)
			}
			certPath = OTAPackageCertPath
		}
	} else {
		// Single file validation (currently config focused, expandable for FOTA/SOTA)
		if !isValidPackageFile(pathToFile) {
			return fmt.Errorf("signature check failed: unsupported file format")
		}
		certPath = OTAPackageCertPath
	}

	// Check signature requirements
	if signature == "" {
		if shouldRequireSignature(fs) {
			return fmt.Errorf("signature is required to proceed with the update")
		} else {
			log.Printf("Proceeding without signature check on package.")
			return nil
		}
	}

	// Calculate checksum
	checksum, err := calculateFileChecksum(fs, pathToFile, hashAlgorithm)
	if err != nil {
		return fmt.Errorf("signature check failed: could not create checksum: %w", err)
	}

	// Verify signature with certificate
	if err := verifyChecksumWithCertificateFile(fs, certPath, signature, checksum); err != nil {
		return fmt.Errorf("signature check failed: %w", err)
	}

	log.Printf("Signature verification passed.")
	return nil
}

// shouldRequireSignature checks if signature verification should be required
func shouldRequireSignature(fs afero.Fs) bool {
	return IsFileExist(fs, OTAPackageCertPath)
}

// extractAndValidateTarContents extracts and validates tar contents
func extractAndValidateTarContents(fs afero.Fs, tarPath string) (*TarContents, error) {
	if !IsFileExist(fs, tarPath) {
		return nil, fmt.Errorf("tar file does not exist: %s", tarPath)
	}

	file, err := Open(fs, tarPath)
	if err != nil {
		return nil, fmt.Errorf("could not open tar file: %w", err)
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	contents := &TarContents{
		PackageFiles: make([]string, 0),
	}
	configFiles := []string{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %w", err)
		}

		// Only process regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		filename := header.Name
		basename := filepath.Base(filename)

		// Check file types (currently config focused, expandable for FOTA/SOTA)
		switch {
		case isPEMFile(basename):
			if !contents.HasPEM {
				// Read PEM content
				pemContent, err := io.ReadAll(tarReader)
				if err != nil {
					return nil, fmt.Errorf("could not read PEM file from tar: %w", err)
				}
				// Validate PEM content
				if err := validatePEMContent(string(pemContent)); err != nil {
					return nil, fmt.Errorf("invalid PEM certificate: %w", err)
				}
				contents.PEMCert = string(pemContent)
				contents.HasPEM = true
			}
		case basename == ConfigFileName:
			configFiles = append(configFiles, filename)
			contents.ConfigFile = filename
			contents.HasConfig = true
		case isValidPackageFile(basename):
			contents.PackageFiles = append(contents.PackageFiles, filename)
		default:
			// Currently supports configuration files, PEM certificates, and package files
			return nil, fmt.Errorf("invalid file in tarball: '%s'. only valid configuration files, PEM certificates, and package files are allowed", basename)
		}
	}

	// Check for multiple configuration files
	if len(configFiles) > 1 {
		return nil, fmt.Errorf("multiple configuration files found in tarball: %v", configFiles)
	}

	// Validate contents
	if !contents.HasConfig && len(contents.PackageFiles) == 0 {
		return nil, fmt.Errorf("no valid package found in tarball")
	}
	return contents, nil
}

// isPEMFile checks if file is a PEM certificate
func isPEMFile(filename string) bool {
	ext := strings.ToLower(getFileExtension(filename))
	return ext == "pem" || ext == "crt" || ext == "cert"
}

// validatePEMContent validates PEM certificate content
func validatePEMContent(pemContent string) error {
	if strings.TrimSpace(pemContent) == "" {
		return fmt.Errorf("empty PEM content")
	}
	block, _ := pem.Decode([]byte(pemContent))
	if block == nil {
		return fmt.Errorf("failed to decode PEM content")
	}
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}
	return nil
}

// getFileExtension returns the file extension
func getFileExtension(filename string) string {
	if filename == "" {
		return ""
	}
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[idx+1:]
	}
	return ""
}

// isValidPackageFile checks if the file is a valid package file
func isValidPackageFile(filename string) bool {
	basename := filepath.Base(filename)
	if basename == ConfigFileName {
		return true
	}
	// Prevent files like "my_intel_manageability.conf"
	if strings.Contains(basename, ConfigFileName) && basename != ConfigFileName {
		return false
	}
	validExtensions := []string{"rpm", "deb", "fv", "cap", "bio", "bin", "conf", "mender"}
	ext := strings.ToLower(getFileExtension(basename))
	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

// validateTarFile validates files within a tar archive
func validateTarFile(fs afero.Fs, tarPath string) error {
	_, err := extractAndValidateTarContents(fs, tarPath)
	return err
}

// calculateFileChecksum calculates the checksum of a file
func calculateFileChecksum(fs afero.Fs, filePath string, hashAlgorithm *HashAlgorithm) ([]byte, error) {
	content, err := ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file content: %w", err)
	}

	var hasher hash.Hash
	// Default to SHA384
	switch {
	case hashAlgorithm == nil, *hashAlgorithm == SHA384:
		hasher = sha512.New384()
	case *hashAlgorithm == SHA256:
		hasher = sha256.New()
	case *hashAlgorithm == SHA512:
		hasher = sha512.New()
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %v", hashAlgorithm)
	}

	hasher.Write(content)
	checksum := hasher.Sum(nil)
	checksumHex := hex.EncodeToString(checksum)

	return []byte(checksumHex), nil
}

// verifyChecksumWithCertificateFile verifies signature using specific certificate file
func verifyChecksumWithCertificateFile(fs afero.Fs, certPath, signature string, checksum []byte) error {
	// Create context with timeout for signature verification
	ctx, cancel := context.WithTimeout(context.Background(), SignatureVerificationTimeoutInSeconds*time.Second)
	defer cancel()
	if !IsFileExist(fs, certPath) {
		return fmt.Errorf("certificate file does not exist: %s", certPath)
	}
	certContent, err := ReadFile(fs, certPath)
	if err != nil {
		return fmt.Errorf("could not load certificate: %w", err)
	}

	// Parse PEM certificate
	block, _ := pem.Decode(certContent)
	if block == nil {
		return fmt.Errorf("failed to parse PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Get public key
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate does not contain RSA public key")
	}

	// Verify signature with timeout protection
	return verifyChecksumWithKey(ctx, pubKey, signature, checksum)
}

// verifyChecksumWithKey verifies the checksum with the public key
func verifyChecksumWithKey(ctx context.Context, pubKey *rsa.PublicKey, signature string, checksum []byte) error {
	if len(checksum) == 0 {
		return fmt.Errorf("invalid checksum")
	}
	if signature == "" {
		return fmt.Errorf("invalid signature")
	}
	// Check key size > MinKeySizeBits
	if pubKey.Size()*8 < MinKeySizeBits {
		return fmt.Errorf("invalid key size. update rejected")
	}

	// Decode signature from hex
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	// Try multiple algorithms for robust verification
	algorithms := []struct {
		name       string
		hasher     func() hash.Hash
		cryptoHash crypto.Hash
	}{
		{"SHA384", sha512.New384, crypto.SHA384}, // Primary
		{"SHA256", sha256.New, crypto.SHA256},    // Fallback
		{"SHA512", sha512.New, crypto.SHA512},    // Additional
	}

	var lastErr error
	for _, alg := range algorithms {
		// Check timeout before each verification attempt
		select {
		case <-ctx.Done():
			return fmt.Errorf("signature verification timeout")
		default:
		}
		hasher := alg.hasher()
		hasher.Write(checksum)
		hashed := hasher.Sum(nil)
		err = rsa.VerifyPSS(pubKey, alg.cryptoHash, hashed, sigBytes, &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthAuto,
		})
		if err == nil {
			log.Printf("Signature verified using %s", alg.name)
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("checksum of data does not match signature in manifest: %w", lastErr)
}
