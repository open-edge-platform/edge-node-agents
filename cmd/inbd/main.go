package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"syscall"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"google.golang.org/grpc"
)

type inbdServer struct {
	pb.UnimplementedInbServiceServer
}

func (s *inbdServer) UpdateSystemSoftware(ctx context.Context, req *pb.UpdateSystemSoftwareRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateSystemSoftware request")
	return &pb.UpdateResponse{StatusCode: 501, Error: "Not implemented"}, nil
}

func (s *inbdServer) UpdateOSSource(ctx context.Context, req *pb.UpdateOSSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received UpdateOSSource request")
	return &pb.UpdateResponse{StatusCode: 501, Error: "Not implemented"}, nil
}

func (s *inbdServer) AddApplicationSource(ctx context.Context, req *pb.AddApplicationSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received AddApplicationSource request")
	return &pb.UpdateResponse{StatusCode: 501, Error: "Not implemented"}, nil
}

func (s *inbdServer) RemoveApplicationSource(ctx context.Context, req *pb.RemoveApplicationSourceRequest) (*pb.UpdateResponse, error) {
	log.Printf("Received RemoveApplicationSource request")
	return &pb.UpdateResponse{StatusCode: 501, Error: "Not implemented"}, nil
}

func main() {
	var socket = flag.String("s", "/var/run/inbd.sock", "UNIX domain socket path")
	flag.Parse()

	if _, err := os.Stat(*socket); err == nil {
		err := os.Remove(*socket)
		if err != nil {
			log.Fatalf("Error removing %s", *socket)
		}
	}

	// when creating the socket, we need it to be with 0600 permissions, atomically
	oldUmask := syscall.Umask(0177)
	lis, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("Error listening to %s: %v", *socket, err)
	}
	syscall.Umask(oldUmask)

	grpcServer := grpc.NewServer()

	pb.RegisterInbServiceServer(grpcServer, &inbdServer{})
	log.Printf("Server listening on %s", *socket)

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
