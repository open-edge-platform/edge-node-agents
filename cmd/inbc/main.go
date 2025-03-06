package main

import (
	"context"
	"log"
	"net"

	pb "example.com/tcv5/pkg/api/inbd/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		// cut off the unix:// part
		addr = addr[7:]
		return net.Dial("unix", addr)
	}

	conn, err := grpc.NewClient("unix:///tmp/inbd.sock", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer))
	if err != nil {
		log.Fatalf("Error setting up new grpc client: %v", err)
	}
	defer conn.Close()

	client := pb.NewInbServiceClient(conn)
	version, err := client.GetVersion(context.Background(), &pb.GetVersionRequest{})
	if err != nil {
		log.Fatalf("error getting server version: %v", err)
	}

	log.Printf("Got version from server: %s", version.GetVersion())
}
