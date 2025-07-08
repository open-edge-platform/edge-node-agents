/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/afero"
)

func main() {
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
			pb.RegisterInbServiceServer(gs, &inbd.InbdServer{})
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
		SetupTLSCertificates: func() error {
			return utils.SetupTLSCertificates()
		},
		LoadX509KeyPair: tls.LoadX509KeyPair,
		ReadFile:        utils.ReadFile,
		NewOsFs:         afero.NewOsFs,
		AppendCertsFromPEM: func(pool *x509.CertPool, pemCerts []byte) bool {
			return pool.AppendCertsFromPEM(pemCerts)
		},
	}

	config, err := utils.LoadConfig(afero.NewOsFs(), utils.ConfigFilePath)
	if err != nil {
		log.Printf("Failed to load config: %s", err)
		os.Exit(1)
	}

	if err := utils.SetupLUKSVolume(afero.NewOsFs(), config); err != nil {
		log.Printf("Failed to set up LUKS volume: %s", err)
	}

	// --- Signal handling for graceful shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	serverErrCh := make(chan error, 1)
	go func() {
		// Run the server in a goroutine
		serverErrCh <- inbd.RunServer(deps)
	}()

	select {
	case sig := <-sigCh:
		log.Printf("Received signal %s, cleaning up...", sig)
		if config != nil {
			if err := utils.RemoveLUKSVolume(config); err != nil {
				log.Printf("Failed to Remove LUKS volume: %s", err)
			}
		}
		os.Exit(0)
	case err := <-serverErrCh:
		if err != nil {
			log.Printf("Server failed: %v", err)
		}
		if config != nil {
			if err := utils.RemoveLUKSVolume(config); err != nil {
				log.Printf("Failed to Remove LUKS volume: %s", err)
			}
		}
		os.Exit(1)
	}
}
