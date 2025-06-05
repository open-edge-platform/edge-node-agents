// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// This main package implements the code for a Mock Server. It is a GRPC server and it uses a protobuf
// interface with definitions of GRPC requests and responses.
// The mock server is receives a UpdateHostHWInfoByGuidRequest message and send back UpdateHostHWInfoByGuidResponse
// to the GRPC client with the relevant HWInfo.
// Currently the UpdateHostHWInfoByGuid will be created with HWInfo gathered from the machine it's running on.
package main

import (
	"context"
	"flag"
	"net"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
)

var version string // injected at build time
var commit string  // injected at build time

var log = logrus.New()

type Server struct {
	proto.HostmgrServer
}

type HdAgent struct {
	mu sync.Mutex
}

var hdAgent HdAgent

func (srv *Server) UpdateHostSystemInfoByGUID(ctx context.Context, req *proto.UpdateHostSystemInfoByGUIDRequest) (*proto.UpdateHostSystemInfoByGUIDResponse, error) {
	log.Printf("UpdateSystemInfoByGUIDRequest: %v\n", req)

	var resp proto.UpdateHostSystemInfoByGUIDResponse

	hdAgent.mu.Lock()
	defer hdAgent.mu.Unlock()

	log.Printf("UpdateSystemInfoByGUIDResponse: %v\n", &resp)
	return &resp, nil
}

func usage() {
	log.Printf(`Usage example:
sudo ./hostmgr-mock \
-certPath path/to/certificate \
-keyPath path/to/key \
-logLevel DEBUG \
-address localhost:8080 \
`)
}

func main() {
	log.Printf("Host Manager Mock %s-%v\n", version, commit)

	if len(os.Args) < 3 {
		usage()
		log.Fatalln("error: not enough parameters")
	}

	//TODO: Validate the input parameters of the Mock Server
	logLevel := flag.String("logLevel", "INFO", "Set logging level for logrus (optional)")
	certPath := flag.String("certPath", "", "Path to TLS certificate")
	keyPath := flag.String("keyPath", "", "Path to TLS key")
	address := flag.String("address", "", "Address on which mock is listening")
	flag.Parse()

	if *address == "" {
		flag.Usage()
		log.Fatal("Error: Address not specified.")
	}

	if *certPath == "" || *keyPath == "" {
		flag.Usage()
		log.Fatal("Error: TLS certificate and key not provided.")
	}

	// only INFO and DEBUG are supported
	switch *logLevel {
	case "INFO":
		log.SetLevel(logrus.InfoLevel)
	case "DEBUG":
		log.SetLevel(logrus.DebugLevel)
	}

	log.Printf("listeningAddr: %s\n", *address)

	//updateSystemInfoByGUID
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	creds, err := credentials.NewServerTLSFromFile(*certPath, *keyPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	s := grpc.NewServer(grpc.Creds(creds))

	srv := Server{}

	proto.RegisterHostmgrServer(s, &srv)

	//TODO: Logging + Debug messages
	log.Println("Listening...")
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error: %v", err)
	}

}
