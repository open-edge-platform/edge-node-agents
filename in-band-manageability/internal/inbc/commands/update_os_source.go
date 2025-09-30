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

// UpdateOSSourceCmd returns a cobra command for the Update OS Source command
func UpdateOSSourceCmd() *cobra.Command {
	var socket string
	var sources []string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Creates a new /etc/apt/sources.list file",
		Long:  "Update command is used to creates a new /etc/apt/sources.list file with only the sources provided.",
		RunE:  handleUpdateOSSource(&socket, &sources, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "List of sources to add")
	must(cmd.MarkFlagRequired("sources"))

	return cmd
}

// handleUpdateOSSource is a helper function to handle the UpdateOSSource command
func handleUpdateOSSource(
	socket *string,
	sources *[]string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SOURCE OS UPDATE INBC Command was invoked.\n")

		// Validate and parse the package list
		sourcesSet := make(map[string]struct{})
		for _, source := range *sources {
			if _, exists := sourcesSet[source]; exists {
				return fmt.Errorf("duplicate source in the sources list: %s", source)
			}
			sourcesSet[source] = struct{}{}
		}

		request := &pb.UpdateOSSourceRequest{
			SourceList: *sources,
		}

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

		ctx, cancel = context.WithTimeout(context.Background(), sourceTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.UpdateOSSource(ctx, request)
		if err != nil {
			return fmt.Errorf("error updating OS sources: %v", err)
		}

		fmt.Printf("SOURCE OS UPDATE Command Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())

		return nil
	}
}
