// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// This main package implements the code for a Mock Server. It is a GRPC server and it uses a protobuf
// interface with definitions of GRPC requests and responses.
// The mock server is receives a RegisterClusterRequest message and send back RegisterClusterResponse
// to the GRPC client with the relevant RegisterClusterCommand.
// Currently the RegisterClusterCommand will be created with an input parameters of the process.
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

	proto "github.com/open-edge-platform/cluster-api-provider-intel/pkg/api/proto"
)

var version string // injected at build time
var commit string  // injected at build time

var log = logrus.New()

type Server struct {
	proto.ClusterOrchestratorSouthboundServer
}

var installCmd, uninstallCmd *string

type ClusterAgent struct {
	mu                  sync.Mutex
	clusterRegistered   bool
	clusterDeregistered bool
}

var clusterAgent ClusterAgent
var integrationTestMode bool

func (srv *Server) RegisterCluster(ctx context.Context, req *proto.RegisterClusterRequest) (*proto.RegisterClusterResponse, error) {
	registerClusterResponse := proto.RegisterClusterResponse{
		InstallCmd:   &proto.ShellScriptCommand{Command: *installCmd},
		UninstallCmd: &proto.ShellScriptCommand{Command: *uninstallCmd}}
	log.Printf("RegisterClusterRequest: %v\n", req)
	log.Printf("RegisterClusterResponse: %v\n", &registerClusterResponse)
	return &registerClusterResponse, nil
}

func (srv *Server) UpdateClusterStatus(ctx context.Context, req *proto.UpdateClusterStatusRequest) (*proto.UpdateClusterStatusResponse, error) {
	log.Printf("UpdateClusterStatusRequest: %v\n", req)

	var resp proto.UpdateClusterStatusResponse

	// if running in regular mode always return REGISTER
	if !integrationTestMode {
		resp.ActionRequest = proto.UpdateClusterStatusResponse_REGISTER
		log.Printf("UpdateClusterStatusResponse: %v\n", &resp)
		return &resp, nil
	}

	// if running in integration test mode return REGISTER -> UNREGISTER -> NONE.
	clusterAgent.mu.Lock()
	defer clusterAgent.mu.Unlock()

	if !clusterAgent.clusterRegistered && req.Code == proto.UpdateClusterStatusRequest_ACTIVE {
		clusterAgent.clusterRegistered = true
	} else if clusterAgent.clusterRegistered && req.Code == proto.UpdateClusterStatusRequest_INACTIVE {
		clusterAgent.clusterDeregistered = true
	}

	if !clusterAgent.clusterRegistered {
		resp.ActionRequest = proto.UpdateClusterStatusResponse_REGISTER
	} else if !clusterAgent.clusterDeregistered {
		resp.ActionRequest = proto.UpdateClusterStatusResponse_DEREGISTER
	} else {
		resp.ActionRequest = proto.UpdateClusterStatusResponse_NONE
	}

	log.Printf("UpdateClusterStatusResponse: %v\n", &resp)
	return &resp, nil
}

func usage() {
	log.Printf(`Usage example:
sudo ./cluster-orch-mock \
-certPath path/to/certificate \
-keyPath path/to/key \
-logLevel DEBUG \
-address localhost:8080 \
-installCmd="curl -fL https://mock.example.intel.com/system-agent-install.sh | sudo  sh -s - --server https://mock.example.intel.com --label 'cattle.io/os=linux' --token 8h5fskvwq5lw8js4488c8h87djxd9ltmdb9tqj86x5mj6njhdc7km6 --ca-checksum b50da8bfa2cbcc13e209b9ffbab4b39c699e0aa2b3fe50f44ec4477c54725ea3 --etcd --controlplane --worker" \
-uninstallCmd="/usr/local/bin/rancher-system-agent-uninstall.sh; /usr/local/bin/rke2-uninstall.sh"
`)
}

func main() {
	log.Printf("Cluster Orchestration Mock %s-%v\n", version, commit)

	if len(os.Args) < 3 {
		usage()
		log.Fatalln("error: not enough parameters")
	}

	//TODO: Validate the input parameters of the Mock Server
	logLevel := flag.String("logLevel", "INFO", "Set logging level for logrus (optional)")
	certPath := flag.String("certPath", "", "Path to TLS certificate")
	keyPath := flag.String("keyPath", "", "Path to TLS key")
	address := flag.String("address", "", "Address on which mock is listening")
	installCmd = flag.String("installCmd", "", "Install command")
	uninstallCmd = flag.String("uninstallCmd", "", "Uninstall command")
	flag.BoolVar(&integrationTestMode, "integrationTestMode", false, "Mode used for integration testing. Cluster Orchestration Mock will send uninstall cmd immediately after cluster is ACTIVE")
	flag.Parse()

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
	log.Printf("installCmd: %s\n", *installCmd)
	log.Printf("uninstallCmd: %s\n", *uninstallCmd)

	//registerClusterCommand
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

	proto.RegisterClusterOrchestratorSouthboundServer(s, &srv)

	//TODO: Logging + Debug messages
	log.Println("Listening...")
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error: %v", err)
	}

}
