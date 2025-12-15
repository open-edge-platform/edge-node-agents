// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

// FuzzPackageValidation fuzzes package name validation logic
func FuzzPackageValidation(f *testing.F) {
	// Valid package names
	f.Add("vim")
	f.Add("package-name-123")
	f.Add("lib_test_99")
	f.Add("test")

	// Invalid package names - basic
	f.Add("package name")
	f.Add("package$invalid")

	// Path traversal attempts (from production fuzzing - INBM_fuzz_results)
	f.Add("../../../etc/passwd")
	f.Add("../../../../etc/shadow")
	f.Add("..\\..\\..\\windows\\system32")

	// Injection attempts (from production fuzzing)
	f.Add("'; DROP TABLE users; --")
	f.Add("package\x00null")
	f.Add("{{{{")
	f.Add("[[[[")
	f.Add("]]]]")
	f.Add("''''")
	f.Add("////")
	f.Add(strings.Repeat("\\", 50))

	// Boundary testing (from production fuzzing)
	f.Add(strings.Repeat("9", 100))
	f.Add(strings.Repeat("X", 255))
	f.Add(strings.Repeat("A", 300))
	f.Add(strings.Repeat("Y", 50))
	f.Add("-1")
	f.Add("-1" + strings.Repeat("X", 100))

	// Type confusion (from production fuzzing)
	f.Add("TRUE")
	f.Add("truetruetrue")
	f.Add("falsefalsefalse")
	f.Add("null")
	f.Add("nullnullnull")

	// Common exploit patterns (from production fuzzing)
	f.Add("admin")
	f.Add("adminadminadmin")
	f.Add("999999999999")

	f.Fuzz(func(t *testing.T, packageName string) {
		isValid := isValidPackageName(packageName)
		hasInvalidChars := strings.ContainsAny(packageName, " $\x00/\\")
		if hasInvalidChars && isValid {
			t.Errorf("Expected validation to reject invalid package: %q", packageName)
		}
	})
}

func isValidPackageName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\ \x00$") {
		return false
	}
	return true
}

// FuzzURLValidation fuzzes URL validation
func FuzzURLValidation(f *testing.F) {
	// Valid URLs
	f.Add("https://example.com/file.tar")
	f.Add("https://example.com/firmware.bin")
	f.Add("http://mirror.example.com/packages")

	// Dangerous URL schemes
	f.Add("javascript:alert(1)")
	f.Add("file:///etc/passwd")
	f.Add("data:text/html,<script>alert(1)</script>")

	// Production-tested patterns from INBM_fuzz_results
	f.Add("////")
	f.Add("\\N\\R\\N")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("nullnullnull")
	f.Add("falsefalsefalse")
	f.Add("[[[[")
	f.Add("]]]]")
	f.Add("''''")
	f.Add("admin")
	f.Add("adminadminadmin")
	f.Add("-1")
	f.Add("999999999999")
	f.Add("TRUE")

	// Boundary testing (from production fuzzing)
	f.Add(strings.Repeat("Y", 50))
	f.Add(strings.Repeat("A", 200))
	f.Add(strings.Repeat("X", 100))
	f.Add("999999999999" + strings.Repeat("X", 100))

	f.Fuzz(func(t *testing.T, url string) {
		isValid := isValidURL(url)
		isDangerous := strings.HasPrefix(url, "javascript:") || strings.HasPrefix(url, "file:")
		if isDangerous && isValid {
			t.Errorf("Expected validation to reject dangerous URL: %q", url)
		}
	})
}

func isValidURL(url string) bool {
	if url == "" {
		return true
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}
	return true
}
