/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"fmt"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// RestartCmd returns a cobra command for the Restart command
func RestartCmd() *cobra.Command {
	var socket string

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restarts the device",
		Long:  `Restarts the device.`,
		RunE:  handleRestart(&socket, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")

	return cmd
}

// ShutdownCmd returns a cobra command for the Shutdown command
func ShutdownCmd() *cobra.Command {
	var socket string

	cmd := &cobra.Command{
		Use:   "shutdown",
		Short: "Shuts down the device",
		Long:  `Shuts down the device.`,
		RunE:  handleShutdown(&socket, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")

	return cmd
}

// handleRestart is a helper function to handle the Restart command
func handleRestart(
	socket *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Restart INBC Command was invoked.\n")

		ctx, cancel := context.WithTimeout(context.Background(), clientDialTimeoutInSeconds*time.Second)
		defer cancel()

		client, conn, err := dialer(ctx, *socket)
		if err != nil {
			return fmt.Errorf("error setting up new gRPC client: %v", err)
		}
		defer func() {
			if c, ok := conn.(*grpc.ClientConn); ok {
				if err := c.Close(); err != nil {
					fmt.Printf("Warning: failed to close gRPC connection: %v\n", err)
				}
			}
		}()

		ctx, cancel = context.WithTimeout(context.Background(), firmwareUpdateTimerInSeconds*time.Second)
		defer cancel()

		resp, err := client.SetPowerState(ctx, &pb.SetPowerStateRequest{
			Action: pb.SetPowerStateRequest_POWER_ACTION_CYCLE,
		})
		if err != nil {
			return fmt.Errorf("error restarting device: %v", err)
		}

		fmt.Printf("Restart Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}

// handleShutdown is a helper function to handle the Shutdown command
func handleShutdown(
	socket *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Shutdown INBC Command was invoked.\n")

		ctx, cancel := context.WithTimeout(context.Background(), clientDialTimeoutInSeconds*time.Second)
		defer cancel()

		client, conn, err := dialer(ctx, *socket)
		if err != nil {
			return fmt.Errorf("error setting up new gRPC client: %v", err)
		}
		defer func() {
			if c, ok := conn.(*grpc.ClientConn); ok {
				if err := c.Close(); err != nil {
					fmt.Printf("Warning: failed to close gRPC connection: %v\n", err)
				}
			}
		}()

		ctx, cancel = context.WithTimeout(context.Background(), firmwareUpdateTimerInSeconds*time.Second)
		defer cancel()

		resp, err := client.SetPowerState(ctx, &pb.SetPowerStateRequest{
			Action: *pb.SetPowerStateRequest_POWER_ACTION_OFF.Enum(),
		})
		if err != nil {
			return fmt.Errorf("error shutting down device: %v", err)
		}

		fmt.Printf("Shutdown Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}
