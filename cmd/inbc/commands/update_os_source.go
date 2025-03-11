/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
)

// UpdateOSSourceCmd returns a cobra command for the Update OS Source command
func UpdateOSSourceCmd() *cobra.Command {
	var sources []string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Creates a new /etc/apt/sources.list file",
		Long:  `Update command is used to creates a new /etc/apt/sources.list file with only the sources provided.`,
		RunE:  handleUpdateOSSource(&sources),
	}

	cmd.MarkFlagRequired("sources")
	cmd.Flags().StringSliceVar(&sources, "sources", []string{}, "List of sources to add")

	return cmd
}

// handleUpdateOSSource is a helper function to handle the UpdateOSSource command
func handleUpdateOSSource(sources *[]string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SOURCE OS UPDATE INBC Command was invoked.\n")

		// Validate and parse the package list
		sourcesSet := make(map[string]struct{})
		for _, source := range *sources {
			if _, exists := sourcesSet[source]; exists {
				fmt.Println("Duplicate source in the sources list.")
				os.Exit(1)
			}
			sourcesSet[source] = struct{}{}
		}

		request := &pb.UpdateOSSourceRequest{
			SourceList: *sources,
		}

		client, conn, err := Dial(context.Background(), "unix:///tmp/inbd.sock")
		if err != nil {
			log.Fatalf("Error setting up new grpc client: %v", err)
		}
		defer conn.Close()

		resp, err := client.UpdateOSSource(context.Background(), request)
		if err != nil {
			log.Fatalf("error updating OS sources: %v", err)
		}

		fmt.Printf("SOURCE OS UPDATE Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}
