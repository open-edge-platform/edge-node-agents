// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// This main package implements the code for a Mock Server. It is a GRPC server and it uses a protobuf
// interface with definitions of GRPC requests and responses.
// The mock server is receives a GetTelemetryConfigByGuidRequest message and send back GetTelemetryConfigResponse
// to the GRPC client with the relevant TelemetryConfig.
package main

import (
	"context"
	"flag"
	"net"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	proto "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
)

var version string // injected at build time
var commit string  // injected at build time

var log = logrus.New()

type Server struct {
	proto.TelemetryMgrServer
}

type PtAgent struct {
	mu sync.Mutex
}

var ptAgent PtAgent

func (srv *Server) GetTelemetryConfigByGUID(ctx context.Context, req *proto.GetTelemetryConfigByGuidRequest) (*proto.GetTelemetryConfigResponse, error) {
	log.Printf("GetTelemetryConfigByGUID: %v\n", req)

	var resp proto.GetTelemetryConfigResponse

	ptAgent.mu.Lock()
	defer ptAgent.mu.Unlock()

	resp = proto.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*proto.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     proto.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     proto.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    proto.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	log.Printf("GetTelemetryConfigResponse: %v\n", &resp)
	return &resp, nil
}

func usage() {
	log.Printf(`Usage example:
sudo ./telemetrymgr-mock \
-logLevel DEBUG \
-address localhost:8080 \
`)
}

func main() {
	log.Printf("Telemetry Manager Mock %s-%v\n", version, commit)

	if len(os.Args) < 3 {
		usage()
		log.Fatalln("error: not enough parameters")
	}

	logLevel := flag.String("logLevel", "INFO", "Set logging level for logrus (optional)")
	address := flag.String("address", "", "Address on which mock is listening")
	flag.Parse()

	if *address == "" {
		flag.Usage()
		log.Fatal("Error: Address not specified.")
	}

	// only INFO and DEBUG are supported
	if *logLevel == "INFO" {
		log.SetLevel(logrus.InfoLevel)
	} else if *logLevel == "DEBUG" {
		log.SetLevel(logrus.DebugLevel)
	}

	log.Printf("listeningAddr: %s\n", *address)

	//GetTelemetryConfigByGUID
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	s := grpc.NewServer()

	srv := Server{}

	proto.RegisterTelemetryMgrServer(s, &srv)

	log.Println("Listening...")
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error: %v", err)
	}

}
