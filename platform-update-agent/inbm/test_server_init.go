package main

import (
	"context"
	"fmt"
	"log"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/internal/inbd"
	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
)

func main() {
	// Test that NewInbdServer properly initializes the powerManager
	server := inbd.NewInbdServer()

	fmt.Printf("Server created: %+v\n", server)

	// Test that we can call SetPowerState without panicking
	req := &pb.SetPowerStateRequest{
		Action: pb.SetPowerStateRequest_POWER_ACTION_UNSPECIFIED,
	}

	resp, err := server.SetPowerState(context.Background(), req)
	if err != nil {
		log.Fatalf("SetPowerState failed: %v", err)
	}

	fmt.Printf("SetPowerState response: StatusCode=%d, Error=%s\n", resp.StatusCode, resp.Error)
	fmt.Println("Test completed successfully - no panic!")
}
