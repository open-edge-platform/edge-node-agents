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

// AddApplicationSourceCmd returns a cobra command for the AddApplicationSource command
func AddApplicationSourceCmd() *cobra.Command {
	var socket string
	var sources []string
	var filename string
	var gpgKeyURI string
	var gpgKeyName string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a new application source",
		Long:  "Add command is used to add a new application source to the list of sources.",
		RunE:  handleAddApplicationSource(&socket, &sources, &filename, &gpgKeyURI, &gpgKeyName, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "List of application sources to add")
	must(cmd.MarkFlagRequired("sources"))
	cmd.Flags().StringVar(&filename, "filename", "", "Filename of the source")
	must(cmd.MarkFlagRequired("filename"))
	cmd.Flags().StringVar(&gpgKeyURI, "gpgKeyUri", "", "GPG key URI")
	cmd.Flags().StringVar(&gpgKeyName, "gpgKeyName", "", "GPG key name")

	return cmd
}

// handleAddApplicationSource is a helper function to handle the AddApplicationSource command
func handleAddApplicationSource(
	socket *string,
	sources *[]string,
	filename *string,
	gpgKeyURI *string,
	gpgKeyName *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SOURCE APPLICATION ADD INBC Command was invoked.\n")

		// Validate and parse the package list
		sourcesSet := make(map[string]struct{})
		for _, source := range *sources {
			if _, exists := sourcesSet[source]; exists {
				return fmt.Errorf("duplicate source in the sources list: %s", source)
			}
			sourcesSet[source] = struct{}{}
		}

		request := &pb.AddApplicationSourceRequest{
			Source:     *sources,
			Filename:   *filename,
			GpgKeyUri:  *gpgKeyURI,
			GpgKeyName: *gpgKeyName,
		}

		ctx, cancel := context.WithTimeout(context.Background(), clientDialTimeoutInSeconds*time.Second)
		defer cancel()

		inbdClient, conn, err := dialer(ctx, *socket)
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

		resp, err := inbdClient.AddApplicationSource(ctx, request)
		if err != nil {
			return fmt.Errorf("error adding application source: %v", err)
		}

		fmt.Printf("SOURCE APPLICATION ADD Command Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())
		return nil
	}
}
