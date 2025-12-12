// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"strings"
	"testing"
)

// FuzzParsePackageOutput fuzzes the Debian package output parser
func FuzzParsePackageOutput(f *testing.F) {
	// Seed with valid inputs
	f.Add("vim 2:8.2.3995-1ubuntu2.17 amd64")
	f.Add("nano 6.2-1 amd64")
	f.Add("")
	f.Add("package-name 1.0.0 all")

	// Production-tested patterns from INBM_fuzz_results
	f.Add("test")
	f.Add("../../../../etc/passwd")
	f.Add("999999999999")
	f.Add("TRUE")
	f.Add(strings.Repeat("\\", 50))
	f.Add("////")
	f.Add("{{{{")
	f.Add("nullnullnull")
	f.Add("'; DROP TABLE users; --")
	f.Add("adminadminadmin")
	f.Add(strings.Repeat("X", 300))
	f.Add(strings.Repeat("A", 500))
	f.Add("[[[[")
	f.Add("]]]]")
	f.Add("-1" + strings.Repeat("X", 100))

	f.Fuzz(func(t *testing.T, output string) {
		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parsePackageOutput panicked: %v", r)
			}
		}()

		packages := parsePackageOutput(output)

		// Basic sanity checks
		if packages == nil {
			t.Error("parsePackageOutput returned nil")
		}
	})
}

// FuzzParseRPMOutput fuzzes the RPM package output parser
func FuzzParseRPMOutput(f *testing.F) {
	// Seed with valid inputs
	f.Add("vim-enhanced-8.2.2637-20.el9_1.x86_64")
	f.Add("kernel-5.14.0-162.6.1.el9_1.x86_64")
	f.Add("")
	f.Add("package-1.0-1.noarch")

	// Production-tested patterns from INBM_fuzz_results (EMT system)
	f.Add("test")
	f.Add("999999999999")
	f.Add("TRUE")
	f.Add("truetruetrue")
	f.Add(strings.Repeat("\\", 50))
	f.Add("////")
	f.Add("{{{{")
	f.Add("[[[[")
	f.Add("]]]]")
	f.Add("nullnullnull")
	f.Add("admin")
	f.Add(strings.Repeat("Y", 50))
	f.Add(strings.Repeat("A", 500))
	f.Add("-1")
	f.Add("'; DROP TABLE users; --")

	f.Fuzz(func(t *testing.T, output string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parseRPMOutput panicked: %v", r)
			}
		}()

		packages := parseRPMOutput(output)

		if packages == nil {
			t.Error("parseRPMOutput returned nil")
		}
	})
}

// FuzzParseRPMPackageName fuzzes the RPM package name parser
func FuzzParseRPMPackageName(f *testing.F) {
	// Seed with valid inputs
	f.Add("vim-enhanced-8.2.2637-20.el9_1.x86_64")
	f.Add("kernel-5.14.0-162.6.1.el9_1.x86_64")
	f.Add("")
	f.Add("package-1.0-1.noarch")
	f.Add("invalid-package")

	// Production-tested patterns from INBM_fuzz_results
	f.Add("test")
	f.Add("../../../../etc/passwd")
	f.Add("999999999999")
	f.Add("TRUE")
	f.Add("falsefalsefalse")
	f.Add(strings.Repeat("\\", 50))
	f.Add("////")
	f.Add("{{{{")
	f.Add("[[[[")
	f.Add("nullnullnull")
	f.Add("admin")
	f.Add(strings.Repeat("a", 1000))
	f.Add(strings.Repeat("X", 500))
	f.Add("-1" + strings.Repeat("X", 100))
	f.Add("'; DROP TABLE users; --")

	f.Fuzz(func(t *testing.T, packageName string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parseRPMPackageName panicked: %v", r)
			}
		}()

		pkg := parseRPMPackageName(packageName)

		// If input is empty, package should be nil or have empty fields
		if packageName == "" && pkg != nil {
			if pkg.Name != "" {
				t.Error("Expected empty package name for empty input")
			}
		}
	})
}
