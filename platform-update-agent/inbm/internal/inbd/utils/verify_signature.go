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
	// File size limits to prevent integer overflow and resource exhaustion
	MaxAllowedFileSize    = 100 * 1024 * 1024 // 100 MB for general files
	MaxAllowedPEMSize     = 64 * 1024         // 64 KB for PEM certificates
	MaxAllowedTarSize     = 500 * 1024 * 1024 // 500 MB for entire tar archive
	MaxAllowedContentSize = 10 * 1024 * 1024  // 10 MB for in-memory content processing
	MaxFilenameLength     = 255               // Maximum filename length
	MaxTarEntries         = 1000              // Maximum number of entries in tar
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
// Generic function that supports configuration, and FOTA packages with embedded PEM certificates
func VerifySignature(signature, pathToFile string, hashAlgorithm *HashAlgorithm) error {
	fs := afero.NewOsFs()

	if err := isFilePathAbsolute(pathToFile); err != nil {
		return fmt.Errorf("signature check failed: invalid path: %w", err)
	}

	if err := isFilePathSymLink(pathToFile); err != nil {
		return fmt.Errorf("signature check failed: path is symlink: %w", err)
	}

	// Get file extension
	extension := strings.TrimPrefix(filepath.Ext(pathToFile), ".")
	var certPath string

	// Handle tar files (FOTA/config packages)
	if strings.ToLower(extension) == "tar" {
		tarContents, err := extractAndValidateTarContents(fs, pathToFile)
		if err != nil {
			return fmt.Errorf("signature check failed: %w", err)
		}

		// Use embedded PEM certificate if available, else system cert
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
		// Single file validation (config, FOTA)
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
	if err := isFilePathAbsolute(tarPath); err != nil {
		return nil, fmt.Errorf("tar path validation failed: invalid path: %w", err)
	}

	if err := isFilePathSymLink(tarPath); err != nil {
		return nil, fmt.Errorf("tar path validation failed: path is symlink: %w", err)
	}

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

		// Only process regular files, reject symlinks and hardlinks
		switch header.Typeflag {
		case tar.TypeReg:
			// Regular file - safe to process
		case tar.TypeSymlink, tar.TypeLink:
			return nil, fmt.Errorf("tar contains symlink/hardlink: %s", header.Name)
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			// Skip header entries
			continue
		default:
			return nil, fmt.Errorf("tar contains unsupported file type %d: %s", header.Typeflag, header.Name)
		}

		filename := header.Name

		if err := validateTarFilename(filename); err != nil {
			return nil, fmt.Errorf("invalid TarFilename '%s': %w", filename, err)
		}

		// Get safe directory for path validation
		tarDir := filepath.Dir(tarPath)

		// Validate path to prevent path traversal attacks
		if err := validateTarPath(fs, filename, tarDir); err != nil {
			return nil, fmt.Errorf("path traversal detected in '%s': %w", filename, err)
		}

		// Validate file size
		if err := validateTarFileSize(header); err != nil {
			return nil, fmt.Errorf("file too large '%s': %w", filename, err)
		}

		basename := filepath.Base(filename)

		// Check file types (currently config focused)
		switch {
		case isPEMFile(basename):
			if !contents.HasPEM {
				// Read PEM content
				pemContent, err := readTarContentWithLimit(tarReader, header.Size, 64*1024)
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
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	return ext == "pem" || ext == "crt" || ext == "cert"
}

// validatePEMContent validates PEM certificate content
func validatePEMContent(pemContent string) error {
	if strings.TrimSpace(pemContent) == "" {
		log.Printf("PEM validation failed: empty PEM content")
		return fmt.Errorf("empty PEM content")
	}
	block, _ := pem.Decode([]byte(pemContent))
	if block == nil {
		log.Printf("PEM validation failed: could not decode PEM content")
		return fmt.Errorf("failed to decode PEM content")
	}
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("PEM validation failed: could not parse X.509 certificate: %v", err)
		return fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}
	return nil
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
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(basename), "."))
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
		log.Printf("Certificate file does not exist: %s", certPath)
		return fmt.Errorf("certificate file does not exist: %s", certPath)
	}
	certContent, err := ReadFile(fs, certPath)
	if err != nil {
		log.Printf("Could not load certificate from %s: %v", certPath, err)
		return fmt.Errorf("could not load certificate: %w", err)
	}

	// Parse PEM certificate
	block, _ := pem.Decode(certContent)
	if block == nil {
		log.Printf("Failed to parse PEM certificate at %s: PEM decode failed", certPath)
		return fmt.Errorf("failed to parse PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse certificate at %s: %v", certPath, err)
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Get public key
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		log.Printf("Certificate at %s does not contain RSA public key", certPath)
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

// Helper to convert string to *HashAlgorithm
func ParseHashAlgorithm(algo string) *HashAlgorithm {
	switch strings.ToLower(algo) {
	case "sha256":
		a := SHA256
		return &a
	case "sha512":
		a := SHA512
		return &a
	default:
		a := SHA384
		return &a
	}
}

// validateTarFilename - validates filename safety
func validateTarFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	if strings.Contains(filename, "\x00") {
		return fmt.Errorf("filename contains null bytes")
	}
	if len(filename) > 255 {
		return fmt.Errorf("filename too long: %d characters", len(filename))
	}
	dangerousChars := []string{"\r", "\n", "\t"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains dangerous character: %q", char)
		}
	}
	return nil
}

// validateTarPath checks for path traversal attacks
func validateTarPath(fs afero.Fs, filename, safeDir string) error {
	// Clean the path to resolve any . and .. elements
	cleanPath := filepath.Clean(filename)

	// Check for absolute paths (should be relative)
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute paths not allowed in tar entries: %s", cleanPath)
	}

	// Check for path traversal patterns
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", cleanPath)
	}

	// ensure resolved path would be within safe directory
	fullPath := filepath.Join(safeDir, cleanPath)

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("could not resolve absolute path: %w", err)
	}

	absSafeDir, err := filepath.Abs(safeDir)
	if err != nil {
		return fmt.Errorf("could not resolve safe directory absolute path: %w", err)
	}

	// Ensure the resolved path is still within the safe directory
	if !strings.HasPrefix(absFullPath, absSafeDir+string(filepath.Separator)) && absFullPath != absSafeDir {
		return fmt.Errorf("path would escape safe directory: %s", filename)
	}

	return nil
}

// validateTarFileSize - prevents resource exhaustion
func validateTarFileSize(header *tar.Header) error {
	// Prevent negative sizes (potential integer underflow)
	if header.Size < 0 {
		return fmt.Errorf("invalid file size: %d (negative sizes not allowed)", header.Size)
	}
	// Prevent integer overflow by checking against max int64
	if header.Size > MaxAllowedFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", header.Size, MaxAllowedFileSize)
	}
	// Special limits for PEM files
	basename := filepath.Base(header.Name)
	if isPEMFile(basename) && header.Size > MaxAllowedPEMSize {
		return fmt.Errorf("PEM file too large: %d bytes (max %d bytes)", header.Size, MaxAllowedPEMSize)
	}
	return nil
}

// readTarContentWithLimit - safe content reading with size validation
func readTarContentWithLimit(reader *tar.Reader, declaredSize int64, maxSize int64) ([]byte, error) {
	// Validate declared size is within limits
	if declaredSize < 0 {
		return nil, fmt.Errorf("invalid declared size: %d (cannot be negative)", declaredSize)
	}

	// Ensure maxSize is reasonable
	if maxSize > MaxAllowedContentSize {
		return nil, fmt.Errorf("max size limit too large: %d bytes (max allowed %d bytes)", maxSize, MaxAllowedContentSize)
	}

	if declaredSize > maxSize {
		return nil, fmt.Errorf("declared size %d exceeds limit %d", declaredSize, maxSize)
	}

	// Prevent overflow in LimitReader calculation
	readerLimit := maxSize + 1 // Always safe due to prior validation
	limitedReader := io.LimitReader(reader, readerLimit)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	actualSize := int64(len(content))

	// Verify content length doesn't exceed safe bounds
	if actualSize > MaxAllowedContentSize {
		return nil, fmt.Errorf("content size exceeds maximum allowed: %d > %d bytes", actualSize, MaxAllowedContentSize)
	}

	// Prevent oversized content attacks
	if actualSize > maxSize {
		return nil, fmt.Errorf("content exceeds size limit: got %d bytes, max allowed %d bytes", actualSize, maxSize)
	}

	// Validate actual size matches declared size
	if actualSize != declaredSize {
		// This could indicate:
		// - Corrupt archive
		// - Malicious content designed to bypass size checks
		// - Archive manipulation attack
		return nil, fmt.Errorf("size mismatch: declared size %d but actual size %d (possible corruption or malicious content)", declaredSize, actualSize)
	}

	return content, nil
}
