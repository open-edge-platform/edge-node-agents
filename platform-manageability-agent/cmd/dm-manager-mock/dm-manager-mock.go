// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	proto "github.com/open-edge-platform/infra-external/dm-manager/pkg/api/dm-manager"
)

var version string // injected at build time
var commit string  // injected at build time

var log = logrus.New()

type Server struct {
	proto.DeviceManagementServer
}

type PlatformManageabilityAgent struct {
	mu sync.Mutex
}

var platformManageabilityAgent PlatformManageabilityAgent

func (srv *Server) ReportAMTStatus(ctx context.Context, req *proto.AMTStatusRequest) (*proto.AMTStatusResponse, error) {
	platformManageabilityAgent.mu.Lock()
	defer platformManageabilityAgent.mu.Unlock()
	// Check request details
	if req.HostId == "" {
		return nil, fmt.Errorf("host ID not provided")
	}
	if req.Status != proto.AMTStatus_ENABLED && req.Status != proto.AMTStatus_DISABLED {
		return nil, fmt.Errorf("incorrect status provided")
	}
	log.Printf("AMTStatusRequest: %v/n", req)
	return &proto.AMTStatusResponse{}, nil
}

func (srv *Server) RetrieveActivationDetails(ctx context.Context, req *proto.ActivationRequest) (*proto.ActivationDetailsResponse, error) {
	platformManageabilityAgent.mu.Lock()
	defer platformManageabilityAgent.mu.Unlock()
	// Check request details
	if req.HostId == "" {
		return nil, fmt.Errorf("host ID not provided")
	}
	activationDetailsResponse := proto.ActivationDetailsResponse{
		HostId:      req.HostId,
		Operation:   proto.OperationType_ACTIVATE,
		ProfileName: "test",
	}
	log.Printf("ActivationRequest: %v\n", req)
	log.Printf("ActivationDetailsResponse: Host Id: %s Operation: %v Profile Name: %s\n",
		activationDetailsResponse.HostId, activationDetailsResponse.Operation, activationDetailsResponse.ProfileName)
	return &activationDetailsResponse, nil
}

func (srv *Server) ReportActivationResults(ctx context.Context, req *proto.ActivationResultRequest) (*proto.ActivationResultResponse, error) {
	platformManageabilityAgent.mu.Lock()
	defer platformManageabilityAgent.mu.Unlock()
	// Check request details
	if req.HostId == "" {
		return nil, fmt.Errorf("host ID not provided")
	}
	if req.ActivationStatus != proto.ActivationStatus_ACTIVATING && req.ActivationStatus != proto.ActivationStatus_ACTIVATED &&
		req.ActivationStatus != proto.ActivationStatus_ACTIVATION_FAILED {
		return nil, fmt.Errorf("invalid activation status received")
	}
	log.Printf("ActivationResultRequest: %v\n", req)
	return &proto.ActivationResultResponse{}, nil
}

func usage() {
	log.Printf(`Usage example:
sudo ./dm-manager-mock \
-certPath path/to/certificate \
-keyPath path/to/key \
-logLevel DEBUG \
-address localhost:8080 \
`)
}

func main() {
	log.Printf("Device Manager Mock %s-%v\n", version, commit)

	if len(os.Args) < 3 {
		usage()
		log.Fatal("error: not enough parameters")
	}

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

	proto.RegisterDeviceManagementServer(s, &srv)

	log.Println("Listening")
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error: %v", err)
	}
}
