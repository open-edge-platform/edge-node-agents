/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
package inbd

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"

	"github.com/spf13/afero"
	"google.golang.org/grpc"

	osUpdater "github.com/intel/intel-inb-manageability/internal/os_updater"
)

const configFilePath = "/etc/intel_manageability.conf"
const schemaFilePath = "/usr/share/inbd_schema.json"

// ServerDeps groups the dependencies needed for running the server.
type ServerDeps struct {
	Socket          string
	Stat            func(string) (os.FileInfo, error)
	Remove          func(string) error
	NetListen       func(network, address string) (net.Listener, error)
	Umask           func(int) int
	NewGRPCServer   func(...grpc.ServerOption) *grpc.Server
	RegisterService func(*grpc.Server)
	ServeFunc       func(*grpc.Server, net.Listener) error
	IsValidJSON     func(afero.Afero, string, string) (bool, error)
	GetInbcGroupID  func() (int, error)
	Chown           func(string, int, int) error    // os.Chown
	Chmod           func(string, os.FileMode) error // os.Chmod
}

// RunServer implements the core logic of the server:
// • If a socket file already exists then remove it.
// • Adjust the process umask (saving and then restoring the previous value)
// • Create a UNIX domain listener on Socket.
// • Create a new gRPC server, register the inbd service and then serve.
func RunServer(deps ServerDeps) error {
	// If the socket file exists, try to remove before binding.
	if _, err := deps.Stat(deps.Socket); err == nil {
		if err := deps.Remove(deps.Socket); err != nil {
			return fmt.Errorf("error removing socket: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("unexpected stat error: %w", err)
	}

	// Set the umask to 0177 (so that the created socket has 0600 permissions)
	// and then restore the previous value afterward.
	oldUmask := deps.Umask(0177)
	lis, err := deps.NetListen("unix", deps.Socket)
	if err != nil {
		deps.Umask(oldUmask)
		return fmt.Errorf("error listening on socket %s: %w", deps.Socket, err)
	}

	inbcGroupID, err := deps.GetInbcGroupID()
	if err != nil {
		return fmt.Errorf("failed to get inbc group GID: %w", err)
	}

	if err := deps.Chown(deps.Socket, 0, inbcGroupID); err != nil {
		return fmt.Errorf("could not chown socket: %w", err)
	}
	if err := deps.Chmod(deps.Socket, 0660); err != nil {
		return fmt.Errorf("could not chmod socket: %w", err)
	}

	deps.Umask(oldUmask)
	if err != nil {
		return fmt.Errorf("error listening on socket: %w", err)
	}

	grpcServer := deps.NewGRPCServer()
	deps.RegisterService(grpcServer)

	// VerifyUpdateAfterReboot verifies the update after reboot.
	// It compares the version in dispatcher_state file with the current system version.
	// If the versions are different, the system successfully boots into the new image.

	fs := afero.NewOsFs()

	err = osUpdater.VerifyUpdateAfterReboot(fs)
	if err != nil {
		return fmt.Errorf("[Post verification failed] error verifying update after reboot: %w", err)
	}

	isValidConfig, err := deps.IsValidJSON(afero.Afero{Fs: fs}, schemaFilePath, configFilePath)
	if err != nil {
		return fmt.Errorf("error validating INBD Configuration file: %w", err)
	}
	if !isValidConfig {
		return fmt.Errorf("INBD Configuration file is not valid")
	}
	return deps.ServeFunc(grpcServer, lis)
}

// GetInbcGroupID retrieves the GID of the 'inbc' group.
func GetInbcGroupID() (int, error) {
	grp, err := user.LookupGroup("inbc")
	if err != nil {
		return 0, fmt.Errorf("could not find group 'inbc': %w", err)
	}
	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return 0, fmt.Errorf("could not parse GID: %w", err)
	}
	return gid, nil
}
