/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"context"
	"fmt"
	"log"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
)

// RemoveApplicationSourceCmd returns a cobra command for the RemoveApplicationSource command
func RemoveApplicationSourceCmd() *cobra.Command {
	var socket string
	var filename string
	var gpgKeyName string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Removes the application source file",
		Long:  "Remove command is used to remove the source file from under /etc/apt/sources.list.d/.",
		RunE:  handleRemoveApplicationSource(&socket, &filename, &gpgKeyName),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVar(&filename, "filename", "", "Filename of the source")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().StringVar(&gpgKeyName, "gpg-key-name", "", "GPG key name")

	return cmd
}

// handleRemoveApplicationSource is a helper function to handle the RemoveApplicationSource command
func handleRemoveApplicationSource(socket *string, filename *string, gpgKeyName *string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SOURCE APPLICATION REMOVE INBC Command was invoked.\n")

		request := &pb.RemoveApplicationSourceRequest{
			Filename:   *filename,
			GpgKeyName: *gpgKeyName,
		}

		client, conn, err := Dial(context.Background(), *socket)
		if err != nil {
			log.Fatalf("Error setting up new gRPC client: %v", err)
		}
		defer conn.Close()

		resp, err := client.RemoveApplicationSource(context.Background(), request)
		if err != nil {
			log.Fatalf("error removing application source: %v", err)
		}

		fmt.Printf("SOURCE APPLICATION REMOVE Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}
