/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FOTACmd returns a cobra command for the FOTA command
func FOTACmd() *cobra.Command {
	var socket string
	var uri string
	var releaseDate string
	var reboot bool
	var userName string
	var signature string
	var hashAlgorithm string

	cmd := &cobra.Command{
		Use:   "fota",
		Short: "Performs Firmware Update",
		Long:  `Updates the firmware on the device.`,
		RunE:  handleFOTA(&socket, &uri, &releaseDate, &reboot, &userName, &signature, &hashAlgorithm, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVar(&uri, "uri", "", "URI from which to remotely retrieve the package")
	err := cmd.MarkFlagRequired("uri")
	if err != nil {
		fmt.Printf("Error marking 'uri' flag as required: %v\n", err)
		return nil
	}
	cmd.Flags().StringVar(&releaseDate, "releasedate", "", "Release date of the new firmware update in YYYY-MM-DD format (required)")
	err = cmd.MarkFlagRequired("releasedate")
	if err != nil {
		fmt.Printf("Error marking 'releasedate' flag as required: %v\n", err)
		return nil
	}
	cmd.Flags().BoolVar(&reboot, "reboot", true, "Whether to reboot after the software update attempt")
	cmd.Flags().StringVar(&userName, "username", "", "Username if authentication is required for the package source")
	cmd.Flags().StringVar(&signature, "signature", "", "Signature of the package")
	cmd.Flags().StringVar(&hashAlgorithm, "hash_algorithm", "", "Hash algorithm to use for signature verification (sha256, sha384, sha512). Default is sha384.")

	return cmd
}

// handleFOTA is a helper function to handle the FOTA command
func handleFOTA(
	socket *string,
	url *string,
	releaseDate *string,
	reboot *bool,
	username *string,
	signature *string,
	hashAlgorithm *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("FOTA INBC Command was invoked.\n")

		// Validate and parse the release date
		var releaseDateProto *timestamppb.Timestamp
		if *releaseDate != "" {
			var parsedDate time.Time
			var err error

			// Try parsing as YYYY-MM-DD format first
			parsedDate, err = time.Parse(time.DateOnly, *releaseDate)
			if err != nil {
				// If that fails, try parsing as RFC3339 (ISO 8601) format
				parsedDate, err = time.Parse(time.RFC3339, *releaseDate)
				if err != nil {
					return fmt.Errorf("error parsing release date (expected YYYY-MM-DD or RFC3339 format): %v", err)
				}
			}
			releaseDateProto = timestamppb.New(parsedDate)
		}
		// TODO: Validate signature against expected format
		// TODO: Add unittest test case for invalid signature format

		// Default to sha384 if not provided
		finalHashAlgorithm := "sha384"
		if hashAlgorithm != nil && *hashAlgorithm != "" {
			switch strings.ToLower(*hashAlgorithm) {
			case "sha256", "sha384", "sha512":
				finalHashAlgorithm = strings.ToLower(*hashAlgorithm)
			default:
				return fmt.Errorf("invalid hash algorithm: must be 'sha256', 'sha384', or 'sha512'")
			}
		}

		request := &pb.UpdateFirmwareRequest{
			Url:           *url,
			ReleaseDate:   releaseDateProto,
			DoNotReboot:   !*reboot,
			Username:      *username,
			Signature:     *signature,
			HashAlgorithm: finalHashAlgorithm,
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
