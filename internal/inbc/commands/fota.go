/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"fmt"
	"regexp"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FOTACmd returns a cobra command for the FOTA command
func FOTACmd() *cobra.Command {
	var socket string
	var url string
	var releaseDate string
	var toolOptions string
	var reboot bool
	var userName string
	var signature string

	cmd := &cobra.Command{
		Use:   "fota",
		Short: "Performs Firmware Update",
		Long:  `Updates the firmware on the device.`,
		RunE:  handleFOTA(&socket, &url, &releaseDate, &toolOptions, &reboot, &userName, &signature, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVar(&url, "uri", "", "URI from which to remotely retrieve the package")
	cmd.Flags().StringVar(&releaseDate, "releasedate", "", "Release date of the new SW update (RFC3339 format)")
	cmd.Flags().StringVar(&toolOptions, "tooloptions", "", "Mode for installing the software update (full, no-download, download-only)")
	cmd.Flags().BoolVar(&reboot, "reboot", true, "Whether to reboot after the software update attempt")
	cmd.Flags().StringVar(&userName, "username", "", "Username if authentication is required for the package source")
	cmd.Flags().StringVar(&signature, "signature", "", "Signature of the package")

	return cmd
}

// handleFOTA is a helper function to handle the FOTA command
func handleFOTA(
	socket *string,
	url *string,
	releaseDate *string,
	toolOptions *string,
	reboot *bool,
	username *string,
	signature *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("FOTA INBC Command was invoked.\n")

		// Validate and parse the release date
		var releaseDateProto *timestamppb.Timestamp
		if *releaseDate != "" {
			parsedDate, err := time.Parse(time.RFC3339, *releaseDate)
			if err != nil {
				return fmt.Errorf("error parsing release date: %v", err)
			}
			releaseDateProto = timestamppb.New(parsedDate)
		}
		// Validate signature against expected format
		if *signature != "" {
			matched, err := regexp.MatchString("^[a-fA-F0-9]{64}$", *signature)
			if err != nil {
				return fmt.Errorf("error validating signature format: %v", err)
			}
			if !matched {
				return fmt.Errorf("signature does not match expected format (64 hex characters)")
			}
		}

		request := &pb.UpdateFirmwareRequest{
			Url:         *url,
			ReleaseDate: releaseDateProto,
			ToolOptions: *toolOptions,
			DoNotReboot: !*reboot,
			Username:    *username,
			Signature:   *signature,
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

		ctx, cancel = context.WithTimeout(context.Background(), firmwareUpdateTimerInSeconds*time.Second)
		defer cancel()

		resp, err := client.UpdateFirmware(ctx, request)
		if err != nil {
			return fmt.Errorf("error updating firmware: %v", err)
		}

		fmt.Printf("FOTA Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}
