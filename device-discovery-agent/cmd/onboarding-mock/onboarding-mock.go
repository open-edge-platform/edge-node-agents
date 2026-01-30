// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// This package implements a simple Mock Onboarding Server for testing purposes.
// Note: Full implementation requires protobuf definitions from infra-onboarding repo.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var version string // injected at build time
var commit string  // injected at build time

// Server is a placeholder for the onboarding mock server
type Server struct {
	// Add proto server embedding when available
}

func usage() {
	fmt.Printf(`Usage example:
./onboarding-mock \
-certPath path/to/certificate \
-keyPath path/to/key \
-address localhost:8443
`)
}

func main() {
	log.Printf("Onboarding Manager Mock %s-%v\n", version, commit)

	certPath := flag.String("certPath", "", "Path to TLS certificate")
	keyPath := flag.String("keyPath", "", "Path to TLS key")
	address := flag.String("address", "localhost:8443", "Address on which mock is listening")
	flag.Parse()

	if *certPath == "" || *keyPath == "" {
		usage()
		log.Fatal("Error: Certificate and key paths must be specified")
	}

	if *address == "" {
		flag.Usage()
		log.Fatal("Error: Address not specified")
	}

	// Load TLS credentials
	creds, err := credentials.NewServerTLSFromFile(*certPath, *keyPath)
	if err != nil {
		log.Fatalf("Failed to load TLS credentials: %v", err)
	}

	// Create listener
	lis, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *address, err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	// TODO: Register onboarding service when proto definitions are available
	// pb.RegisterOnboardingManagerServer(grpcServer, &Server{})

	log.Printf("Onboarding mock server listening on %s", *address)
	log.Println("Note: Proto definitions needed for full implementation")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
