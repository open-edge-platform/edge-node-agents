package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "example.com/tcv5/pkg/api/inbd/v1"
	"google.golang.org/grpc"
)

type inbdServer struct {
	pb.UnimplementedInbServiceServer
}

func (s *inbdServer) GetVersion(ctx context.Context, req *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	log.Printf("Received GetVersion request")
	return &pb.GetVersionResponse{
		Version: "1.0.0",
	}, nil
}

func main() {
	// remove sock if exists
	if _, err := os.Stat("/tmp/inbd.sock"); err == nil {
		err := os.Remove("/tmp/inbd.sock")
		if err != nil {
			log.Fatalf("Error removing /tmp/inbd.sock")
		}
	}

	lis, err := net.Listen("unix", "/tmp/inbd.sock")
	if err != nil {
		log.Fatalf("Error listening to /tmp/inbd.sock: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterInbServiceServer(grpcServer, &inbdServer{})
	log.Println("Server listening on /tmp/inbd.sock")

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
