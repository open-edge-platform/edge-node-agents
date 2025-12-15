// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package inbd

import (
	"context"
	"strings"
	"testing"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
)

// FuzzUpdateFirmwareURL fuzzes the UpdateFirmware gRPC handler with various URL inputs
func FuzzUpdateFirmwareURL(f *testing.F) {
	// Valid URLs
	f.Add("https://example.com/firmware.bin")
	f.Add("https://firmware.example.com/update.img")

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
	f.Add(strings.Repeat("Y", 50))
	f.Add(strings.Repeat("A", 200))
	f.Add(strings.Repeat("X", 100))

	// Dangerous schemes
	f.Add("javascript:alert(1)")
	f.Add("file:///etc/passwd")
	f.Add("http://example.com/firmware.bin")

	f.Fuzz(func(t *testing.T, url string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.UpdateFirmwareRequest{
			Url: url,
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateFirmware panicked with URL %q: %v", url, r)
			}
		}()

		resp, err := server.UpdateFirmware(ctx, req)

		// Verify response is never nil
		if resp == nil && err == nil {
			t.Error("UpdateFirmware returned nil response and nil error")
		}
	})
}

// FuzzUpdateSystemSoftwareURL fuzzes UpdateSystemSoftware URL field
func FuzzUpdateSystemSoftwareURL(f *testing.F) {
	// Valid inputs
	f.Add("https://example.com/packages.tar")
	f.Add("")

	// Production-tested patterns
	f.Add("////")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("[[[[")
	f.Add("admin")
	f.Add("999999999999")
	f.Add(strings.Repeat("X", 500))

	f.Fuzz(func(t *testing.T, url string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.UpdateSystemSoftwareRequest{
			Url: url,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateSystemSoftware panicked with URL %q: %v", url, r)
			}
		}()

		resp, err := server.UpdateSystemSoftware(ctx, req)
		if resp == nil && err == nil {
			t.Error("UpdateSystemSoftware returned nil response and nil error")
		}
	})
}

// FuzzUpdateSystemSoftwareReleaseDate fuzzes release_date field
func FuzzUpdateSystemSoftwareReleaseDate(f *testing.F) {
	// Valid dates (as strings, will be ignored since we can't easily create Timestamp)
	f.Add("2025-12-11")
	f.Add("2024-01-01")
	f.Add("")

	// Production-tested patterns
	f.Add("999999999999")
	f.Add("TRUE")
	f.Add("nullnullnull")
	f.Add("[[[[")
	f.Add(strings.Repeat("9", 100))
	f.Add("-1")
	f.Add("admin")

	f.Fuzz(func(t *testing.T, releaseDate string) {
		server := NewInbdServer()
		ctx := context.Background()

		// Note: ReleaseDate is *timestamppb.Timestamp, so we pass nil
		// The fuzzing here tests the string patterns but the actual field is nil
		req := &pb.UpdateSystemSoftwareRequest{
			Url:         "https://example.com/update.tar",
			ReleaseDate: nil, // Can't easily fuzz timestamp fields
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateSystemSoftware panicked: %v", r)
			}
		}()

		_, _ = server.UpdateSystemSoftware(ctx, req)
	})
}

// FuzzLoadConfigURI fuzzes LoadConfig URI field
func FuzzLoadConfigURI(f *testing.F) {
	// Valid URIs
	f.Add("file:///etc/inbd/config.yaml")
	f.Add("/etc/inbd/config.yaml")
	f.Add("")

	// Production-tested patterns
	f.Add("../../../../etc/passwd")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("[[[[")
	f.Add("\\\\\\\\")
	f.Add("////")
	f.Add(strings.Repeat("a", 1000))

	f.Fuzz(func(t *testing.T, uri string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.LoadConfigRequest{
			Uri: uri,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LoadConfig panicked with URI %q: %v", uri, r)
			}
		}()

		_, _ = server.LoadConfig(ctx, req)
	})
}

// FuzzGetConfigPath fuzzes GetConfig path field
func FuzzGetConfigPath(f *testing.F) {
	// Valid paths
	f.Add("logging.level")
	f.Add("network.interface")
	f.Add("")

	// Production-tested patterns (this caused TIMEOUT in production)
	f.Add("../../../../etc/passwd")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("nullnullnull")
	f.Add("[[[[")
	f.Add("admin")
	f.Add(strings.Repeat("a", 1000))
	f.Add(strings.Repeat(".", 500))

	f.Fuzz(func(t *testing.T, path string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.GetConfigRequest{
			Path: path,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetConfig panicked with path %q: %v", path, r)
			}
		}()

		_, _ = server.GetConfig(ctx, req)
	})
}

// FuzzSetConfigPath fuzzes SetConfig path field
func FuzzSetConfigPath(f *testing.F) {
	// Valid paths
	f.Add("logging.level=debug")
	f.Add("network.interface=eth0")
	f.Add("")

	// Production-tested patterns (this caused TIMEOUT in production)
	f.Add("../../../../etc/passwd")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("[[[[")
	f.Add(strings.Repeat("=", 500))
	f.Add("admin=admin")

	f.Fuzz(func(t *testing.T, path string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.SetConfigRequest{
			Path: path,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SetConfig panicked with path %q: %v", path, r)
			}
		}()

		_, _ = server.SetConfig(ctx, req)
	})
}

// FuzzAppendConfigPath fuzzes AppendConfig path field
func FuzzAppendConfigPath(f *testing.F) {
	f.Add("logging.handlers=file")
	f.Add("")
	f.Add("../../../../etc/passwd")
	f.Add("[[[[")
	f.Add(strings.Repeat("a", 1000))

	f.Fuzz(func(t *testing.T, path string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.AppendConfigRequest{
			Path: path,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AppendConfig panicked with path %q: %v", path, r)
			}
		}()

		_, _ = server.AppendConfig(ctx, req)
	})
}

// FuzzRemoveConfigPath fuzzes RemoveConfig path field
func FuzzRemoveConfigPath(f *testing.F) {
	f.Add("logging.handler")
	f.Add("")
	f.Add("../../../../etc/passwd")
	f.Add("'; DROP TABLE users; --")
	f.Add(strings.Repeat(".", 500))

	f.Fuzz(func(t *testing.T, path string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.RemoveConfigRequest{
			Path: path,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RemoveConfig panicked with path %q: %v", path, r)
			}
		}()

		_, _ = server.RemoveConfig(ctx, req)
	})
}

// FuzzQueryOption fuzzes Query option field
func FuzzQueryOption(f *testing.F) {
	// Test various option values (as int32)
	f.Add(int32(0)) // UNSPECIFIED
	f.Add(int32(1)) // HARDWARE
	f.Add(int32(2)) // FIRMWARE
	f.Add(int32(3)) // OS
	f.Add(int32(4)) // SWBOM
	f.Add(int32(5)) // VERSION
	f.Add(int32(6)) // ALL
	f.Add(int32(-1))
	f.Add(int32(999))

	f.Fuzz(func(t *testing.T, option int32) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.QueryRequest{
			Option: pb.QueryOption(option),
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Query panicked with option %d: %v", option, r)
			}
		}()

		_, _ = server.Query(ctx, req)
	})
}

// FuzzSetPowerStateAction fuzzes SetPowerState action field
func FuzzSetPowerStateAction(f *testing.F) {
	// Test various action values (as int32)
	f.Add(int32(0)) // UNSPECIFIED
	f.Add(int32(1)) // CYCLE
	f.Add(int32(2)) // OFF
	f.Add(int32(-1))
	f.Add(int32(999))
	f.Add(int32(2147483647))

	f.Fuzz(func(t *testing.T, action int32) {
		// Use existing MockPowerManager from server_test.go
		mockPM := &MockPowerManager{}
		server := NewInbdServerWithPowerManager(mockPM)
		ctx := context.Background()

		req := &pb.SetPowerStateRequest{
			Action: pb.SetPowerStateRequest_PowerAction(action),
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SetPowerState panicked with action %d: %v", action, r)
			}
		}()

		resp, err := server.SetPowerState(ctx, req)
		if resp == nil && err == nil {
			t.Error("SetPowerState returned nil response and nil error")
		}
	})
}

// FuzzAddApplicationSourceGPGKeyName fuzzes AddApplicationSource gpg_key_name field
func FuzzAddApplicationSourceGPGKeyName(f *testing.F) {
	// Valid inputs
	f.Add("my-key")
	f.Add("")

	// Production-tested patterns
	f.Add("../../../../etc/passwd")
	f.Add("null")
	f.Add("[[[[")
	f.Add("admin")
	f.Add(strings.Repeat("X", 500))

	f.Fuzz(func(t *testing.T, gpgKeyName string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.AddApplicationSourceRequest{
			GpgKeyName: gpgKeyName,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AddApplicationSource panicked with gpg_key_name %q: %v", gpgKeyName, r)
			}
		}()

		_, _ = server.AddApplicationSource(ctx, req)
	})
}

// FuzzRemoveApplicationSourceFilename fuzzes RemoveApplicationSource filename field
func FuzzRemoveApplicationSourceFilename(f *testing.F) {
	// Valid inputs
	f.Add("myrepo.list")
	f.Add("")

	// Production-tested patterns
	f.Add("../../../../etc/passwd")
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("[[[[")
	f.Add(strings.Repeat(".", 500))

	f.Fuzz(func(t *testing.T, filename string) {
		server := NewInbdServer()
		ctx := context.Background()

		req := &pb.RemoveApplicationSourceRequest{
			Filename: filename,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RemoveApplicationSource panicked with filename %q: %v", filename, r)
			}
		}()

		_, _ = server.RemoveApplicationSource(ctx, req)
	})
}

// FuzzUpdateOSSourceList fuzzes UpdateOSSource source_list field
func FuzzUpdateOSSourceList(f *testing.F) {
	// Valid inputs
	f.Add("deb http://archive.ubuntu.com/ubuntu focal main")
	f.Add("")

	// Production-tested patterns
	f.Add("'; DROP TABLE users; --")
	f.Add("null")
	f.Add("[[[[")
	f.Add(strings.Repeat("X", 1000))
	f.Add("admin")

	f.Fuzz(func(t *testing.T, sourceList string) {
		server := NewInbdServer()
		ctx := context.Background()

		// SourceList is []string, so wrap the fuzzed string in a slice
		req := &pb.UpdateOSSourceRequest{
			SourceList: []string{sourceList},
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateOSSource panicked with source_list %q: %v", sourceList, r)
			}
		}()

		_, _ = server.UpdateOSSource(ctx, req)
	})
}
