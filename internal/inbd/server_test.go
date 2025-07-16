/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
package inbd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

// MockPowerManager implements PowerManager for fast testing without delays
type MockPowerManager struct {
	RebootError    error
	ShutdownError  error
	RebootCalled   bool
	ShutdownCalled bool
}

func (m *MockPowerManager) Reboot() error {
	m.RebootCalled = true
	return m.RebootError
}

func (m *MockPowerManager) Shutdown() error {
	m.ShutdownCalled = true
	return m.ShutdownError
}

// dummyFileInfo implements os.FileInfo for testing.
type dummyFileInfo struct{}

func (d dummyFileInfo) Name() string       { return "dummy.sock" }
func (d dummyFileInfo) Size() int64        { return 0 }
func (d dummyFileInfo) Mode() os.FileMode  { return 0600 }
func (d dummyFileInfo) ModTime() time.Time { return time.Time{} }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() interface{}   { return nil }

// fakeListener is a dummy net.Listener.
type fakeListener struct {
	closed bool
}

func (fl *fakeListener) Accept() (net.Conn, error) {
	return nil, errors.New("fake listener: no connection")
}

func (fl *fakeListener) Close() error {
	fl.closed = true
	return nil
}

func (fl *fakeListener) Addr() net.Addr {
	return fakeAddr("fakeAddr")
}

type fakeAddr string

func (a fakeAddr) Network() string { return string(a) }
func (a fakeAddr) String() string  { return string(a) }

// createBaseTLSServerDeps returns a ServerDeps with default TLS mocks configured.
func createBaseTLSServerDeps() ServerDeps {
	return ServerDeps{
		Socket: "dummy.sock",
		Stat: func(name string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Remove: os.Remove,
		NetListen: func(network, address string) (net.Listener, error) {
			return &fakeListener{}, nil
		},
		Umask: func(mask int) int {
			return 0
		},
		NewGRPCServer: func(opts ...grpc.ServerOption) *grpc.Server {
			return grpc.NewServer(opts...)
		},
		RegisterService: func(gs *grpc.Server) {},
		ServeFunc: func(gs *grpc.Server, lis net.Listener) error {
			return nil
		},
		IsValidJSON: func(afero.Afero, string, string) (bool, error) {
			return true, nil
		},
		GetInbcGroupID: func() (int, error) {
			return 1000, nil
		},
		Chown: func(path string, uid, gid int) error {
			return nil
		},
		Chmod: func(path string, mode os.FileMode) error {
			return nil
		},
		SetupTLSCertificates: func() error {
			return nil
		},
		LoadX509KeyPair: func(certFile, keyFile string) (tls.Certificate, error) {
			return tls.Certificate{}, nil
		},
		ReadFile: func(fs afero.Fs, filename string) ([]byte, error) {
			return []byte("mock CA certificate"), nil
		},
		NewOsFs: func() afero.Fs {
			return afero.NewMemMapFs()
		},
		AppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
			return true
		},
	}
}

// TestRunServer_Success verifies that when no socket file exists RunServer succeeds.
func TestRunServer_Success(t *testing.T) {
	fl := &fakeListener{}
	removeCalled := false

	deps := createBaseTLSServerDeps()
	deps.Remove = func(name string) error {
		removeCalled = true
		return nil
	}
	deps.NetListen = func(network, address string) (net.Listener, error) {
		if network != "unix" || address != "dummy.sock" {
			return nil, errors.New("unexpected parameters")
		}
		return fl, nil
	}
	deps.ServeFunc = func(gs *grpc.Server, lis net.Listener) error {
		if lis != fl {
			return errors.New("listener mismatch")
		}
		return nil
	}

	err := RunServer(deps)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if removeCalled {
		t.Errorf("Remove should not be called because Stat returned os.ErrNotExist")
	}
}

func TestRunServer_ConfigValidationFails(t *testing.T) {
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		if network != "unix" || address != "dummy.sock" {
			return nil, errors.New("unexpected parameters")
		}
		return fl, nil
	}
	deps.ServeFunc = func(gs *grpc.Server, lis net.Listener) error {
		if lis != fl {
			return errors.New("listener mismatch")
		}
		return nil
	}
	deps.IsValidJSON = func(afero.Afero, string, string) (bool, error) {
		return true, errors.New("invalid JSON")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "error validating INBD Configuration file: invalid JSON" {
		t.Errorf("Expected error on configuration file JSON validation, got %v", err)
	}
}

func TestRunServer_ConfigValidationInvalid(t *testing.T) {
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		if network != "unix" || address != "dummy.sock" {
			return nil, errors.New("unexpected parameters")
		}
		return fl, nil
	}
	deps.ServeFunc = func(gs *grpc.Server, lis net.Listener) error {
		if lis != fl {
			return errors.New("listener mismatch")
		}
		return nil
	}
	deps.IsValidJSON = func(afero.Afero, string, string) (bool, error) {
		return false, nil
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "INBD Configuration file is not valid" {
		t.Errorf("Expected error on configuration file JSON validation, got %v", err)
	}
}

// TestRunServer_RemoveError simulates a failure removing the existing socket.
func TestRunServer_RemoveError(t *testing.T) {
	deps := createBaseTLSServerDeps()
	deps.Stat = func(name string) (os.FileInfo, error) {
		return dummyFileInfo{}, nil
	}
	deps.Remove = func(name string) error {
		return errors.New("remove failed")
	}
	// Other functions are not invoked.
	deps.NetListen = nil

	err := RunServer(deps)
	if err == nil || err.Error() != "error removing socket: remove failed" {
		t.Errorf("Expected error on remove, got %v", err)
	}
}

// TestRunServer_NetListenError simulates a failure when creating the listener.
func TestRunServer_NetListenError(t *testing.T) {
	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return nil, errors.New("netListen failed")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "error listening on socket dummy.sock: netListen failed" {
		t.Errorf("Expected netListen error, got %v", err)
	}
}

// TestRunServer_ServeError simulates an error during serving.
func TestRunServer_ServeError(t *testing.T) {
	fl := &fakeListener{}
	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.ServeFunc = func(gs *grpc.Server, lis net.Listener) error {
		return errors.New("serve error")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "serve error" {
		t.Errorf("Expected serve error, got %v", err)
	}
}

// TestRunServer_RegisterCalled verifies that RegisterService is invoked.
func TestRunServer_RegisterCalled(t *testing.T) {
	registerCalled := false
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.RegisterService = func(gs *grpc.Server) {
		registerCalled = true
	}

	err := RunServer(deps)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !registerCalled {
		t.Errorf("Expected RegisterService to be called")
	}
}

// TestRunServer_UmaskRestoration verifies that Umask is called to set and then restore.
func TestRunServer_UmaskRestoration(t *testing.T) {
	var maskSet []int
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.Umask = func(mask int) int {
		maskSet = append(maskSet, mask)
		return 0 // old umask
	}

	err := RunServer(deps)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(maskSet) != 2 {
		t.Errorf("Expected Umask to be called twice, got %d", len(maskSet))
	}
	if maskSet[0] != 0177 {
		t.Errorf("Expected first Umask call with 0177, got %o", maskSet[0])
	}
	if maskSet[1] != 0 {
		t.Errorf("Expected second Umask call with 0, got %o", maskSet[1])
	}

}

// TestRunServer_TLSSetupError simulates an error during TLS certificate setup.
func TestRunServer_TLSSetupError(t *testing.T) {
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.SetupTLSCertificates = func() error {
		return errors.New("TLS setup failed")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "failed to set up TLS certificates: TLS setup failed" {
		t.Errorf("Expected TLS setup error, got %v", err)
	}
}

// TestRunServer_LoadX509KeyPairError simulates an error during certificate loading.
func TestRunServer_LoadX509KeyPairError(t *testing.T) {
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.LoadX509KeyPair = func(certFile, keyFile string) (tls.Certificate, error) {
		return tls.Certificate{}, errors.New("failed to load certificate")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "failed to load server certificate: failed to load certificate" {
		t.Errorf("Expected certificate load error, got %v", err)
	}
}

// TestRunServer_ReadFileError simulates an error during CA certificate reading.
func TestRunServer_ReadFileError(t *testing.T) {
	fl := &fakeListener{}

	deps := createBaseTLSServerDeps()
	deps.NetListen = func(network, address string) (net.Listener, error) {
		return fl, nil
	}
	deps.ReadFile = func(fs afero.Fs, filename string) ([]byte, error) {
		return nil, errors.New("failed to read CA certificate")
	}

	err := RunServer(deps)
	if err == nil || err.Error() != "failed to read CA certificate: failed to read CA certificate" {
		t.Errorf("Expected CA certificate read error, got %v", err)
	}
}

// TestSetPowerState_UnspecifiedAction tests SetPowerState with unspecified power action
func TestSetPowerState_UnspecifiedAction(t *testing.T) {
	mockPowerManager := &MockPowerManager{}
	server := NewInbdServerWithPowerManager(mockPowerManager)
	ctx := context.Background()

	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_UNSPECIFIED,
	}

	resp, err := server.SetPowerState(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}
	if resp.Error != "Power action is required" {
		t.Errorf("Expected error 'Power action is required', got %s", resp.Error)
	}

	// Verify no power operations were called
	if mockPowerManager.RebootCalled || mockPowerManager.ShutdownCalled {
		t.Errorf("Expected no power operations to be called for unspecified action")
	}
}

// TestSetPowerState_CycleAction tests SetPowerState with cycle power action
func TestSetPowerState_CycleAction(t *testing.T) {
	mockPowerManager := &MockPowerManager{}
	server := NewInbdServerWithPowerManager(mockPowerManager)
	ctx := context.Background()

	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_CYCLE,
	}

	resp, err := server.SetPowerState(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
	if resp.Error != "SUCCESS" {
		t.Errorf("Expected error 'SUCCESS', got %s", resp.Error)
	}

	// Verify reboot was called
	if !mockPowerManager.RebootCalled {
		t.Errorf("Expected Reboot to be called")
	}
	if mockPowerManager.ShutdownCalled {
		t.Errorf("Expected Shutdown not to be called")
	}
}

// TestSetPowerState_OffAction tests SetPowerState with power off action
func TestSetPowerState_OffAction(t *testing.T) {
	mockPowerManager := &MockPowerManager{}
	server := NewInbdServerWithPowerManager(mockPowerManager)
	ctx := context.Background()

	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_OFF,
	}

	resp, err := server.SetPowerState(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
	if resp.Error != "SUCCESS" {
		t.Errorf("Expected error 'SUCCESS', got %s", resp.Error)
	}

	// Verify shutdown was called
	if !mockPowerManager.ShutdownCalled {
		t.Errorf("Expected Shutdown to be called")
	}
	if mockPowerManager.RebootCalled {
		t.Errorf("Expected Reboot not to be called")
	}
}

// TestSetPowerState_CycleActionError tests SetPowerState with cycle power action that fails
func TestSetPowerState_CycleActionError(t *testing.T) {
	mockPowerManager := &MockPowerManager{
		RebootError: errors.New("reboot failed"),
	}
	server := NewInbdServerWithPowerManager(mockPowerManager)
	ctx := context.Background()

	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_CYCLE,
	}

	resp, err := server.SetPowerState(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}
	if resp.Error != "reboot failed" {
		t.Errorf("Expected error 'reboot failed', got %s", resp.Error)
	}

	// Verify reboot was called
	if !mockPowerManager.RebootCalled {
		t.Errorf("Expected Reboot to be called")
	}
	if mockPowerManager.ShutdownCalled {
		t.Errorf("Expected Shutdown not to be called")
	}
}

// TestSetPowerState_OffActionError tests SetPowerState with power off action that fails
func TestSetPowerState_OffActionError(t *testing.T) {
	mockPowerManager := &MockPowerManager{
		ShutdownError: errors.New("shutdown failed"),
	}
	server := NewInbdServerWithPowerManager(mockPowerManager)
	ctx := context.Background()

	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_OFF,
	}

	resp, err := server.SetPowerState(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}
	if resp.Error != "shutdown failed: shutdown failed" {
		t.Errorf("Expected error 'shutdown failed: shutdown failed', got %s", resp.Error)
	}

	// Verify shutdown was called
	if !mockPowerManager.ShutdownCalled {
		t.Errorf("Expected Shutdown to be called")
	}
	if mockPowerManager.RebootCalled {
		t.Errorf("Expected Reboot not to be called")
	}
}

// TestUpdateFirmware_EmptyURL tests UpdateFirmware with empty URL
func TestUpdateFirmware_EmptyURL(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	req := &pb.UpdateFirmwareRequest{
		Url: "",
	}

	resp, err := server.UpdateFirmware(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}
	if resp.Error != "URL is required" {
		t.Errorf("Expected error 'URL is required', got %s", resp.Error)
	}
}

// TestUpdateFirmware_InvalidURL tests UpdateFirmware with invalid URL
func TestUpdateFirmware_InvalidURL(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	testCases := []struct {
		name        string
		url         string
		expectedErr string
	}{
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectedErr: "URL is not valid:",
		},
		{
			name:        "HTTP instead of HTTPS",
			url:         "http://example.com/firmware.bin",
			expectedErr: "URL must use https scheme",
		},
		{
			name:        "HTTPS URL without host",
			url:         "https://",
			expectedErr: "URL must have a host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &pb.UpdateFirmwareRequest{
				Url: tc.url,
			}

			resp, err := server.UpdateFirmware(ctx, req)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if resp.StatusCode != 400 {
				t.Errorf("Expected status code 400, got %d", resp.StatusCode)
			}
			if !contains(resp.Error, tc.expectedErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErr, resp.Error)
			}
		})
	}
}

// TestUpdateFirmware_ValidURL tests UpdateFirmware with valid URL
func TestUpdateFirmware_ValidURL(t *testing.T) {
	server := &InbdServer{}
	ctx := context.Background()

	req := &pb.UpdateFirmwareRequest{
		Url:         "https://example.com/firmware.bin",
		DoNotReboot: true,
		Username:    "testuser",
		Signature:   "test-signature",
	}

	resp, err := server.UpdateFirmware(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The firmware update will likely fail due to network/download issues in test environment
	// but we test that the initial validation passes
	if resp.StatusCode != 200 && resp.StatusCode != 500 {
		t.Errorf("Expected status code 200 or 500, got %d", resp.StatusCode)
	}

	// If it's a download error (500), that's expected in test environment
	if resp.StatusCode == 500 {
		// This is expected as the download will likely fail
		if resp.Error == "" {
			t.Errorf("Expected error message for status code 500")
		}
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

// findSubstring is a helper to find substring in string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
