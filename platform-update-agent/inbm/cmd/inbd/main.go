/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
package main

import (
	"flag"
	"log"
	"net"
	"os"
	"syscall"

	"google.golang.org/grpc"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd/utils"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

func main() {
	// Check if running as root
	if os.Geteuid() != 0 {
		log.Fatal("inbd must be run as root (use sudo)")
	}

	socket := flag.String("s", "/var/run/inbd.sock", "UNIX domain socket path")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Build our dependency struct using real functions.
	deps := inbd.ServerDeps{
		Socket:        *socket,
		Stat:          os.Stat,
		Remove:        os.Remove,
		NetListen:     net.Listen,
		Umask:         syscall.Umask,
		NewGRPCServer: grpc.NewServer,
		RegisterService: func(gs *grpc.Server) {
			// Register our inbdServer implementation.
			pb.RegisterInbServiceServer(gs, inbd.NewInbdServer())
		},
		ServeFunc: func(gs *grpc.Server, lis net.Listener) error {
			log.Printf("Server listening on %s", *socket)
			return gs.Serve(lis)
		},
		IsValidJSON: func(fs afero.Afero, filePath string, schemaPath string) (bool, error) {
			return utils.IsValidJSON(afero.Afero{Fs: afero.NewOsFs()}, filePath, schemaPath)
		},
		GetInbcGroupID: func() (int, error) {
			return inbd.GetInbcGroupID()
		},
		Chown: func(path string, uid, gid int) error {
			return os.Chown(path, uid, gid)
		},
		Chmod: func(path string, mode os.FileMode) error {
			return os.Chmod(path, mode)
		},
	}

	// Run the server (returning an error instead of calling log.Fatal internally).
	if err := inbd.RunServer(deps); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
