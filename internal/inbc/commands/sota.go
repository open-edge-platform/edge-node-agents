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

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/intel/intel-inb-manageability/internal/inbc/utils"
)

// SOTACmd returns a cobra command for the SOTA command
func SOTACmd() *cobra.Command {
	var socket string
	var url string
	var releaseDate string
	var mode string
	var reboot bool
	var packageList []string
	var signature string

	cmd := &cobra.Command{
		Use:   "sota",
		Short: "Performs System Software Update",
		Long:  `Updates the system software on the device.`,
		RunE:  handleSOTA(&socket, &url, &releaseDate, &mode, &reboot, &packageList, &signature, utils.DetectOS, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVar(&url, "uri", "", "URI from which to remotely retrieve the package")
	cmd.Flags().StringVar(&releaseDate, "releasedate", "", "Release date of the new SW update (RFC3339 format)")
	cmd.Flags().StringVar(&mode, "mode", "full", "Mode for installing the software update (full, no-download, download-only)")
	cmd.Flags().BoolVar(&reboot, "reboot", true, "Whether to reboot after the software update attempt")
	cmd.Flags().StringSliceVar(&packageList, "package-list", []string{}, "List of packages to install if whole package update isn't desired")
	cmd.Flags().StringVar(&signature, "signature", "", "Signature of the package")

	return cmd
}

// handleSOTA is a helper function to handle the SOTA command
func handleSOTA(
	socket *string,
	url *string,
	releaseDate *string,
	mode *string,
	reboot *bool,
	packageList *[]string,
	signature *string,
	detectOS func() (string, error),
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SOTA INBC Command was invoked.\n")

		// Validate and parse the release date
		var releaseDateProto *timestamppb.Timestamp
		if *releaseDate != "" {
			parsedDate, err := time.Parse(time.RFC3339, *releaseDate)
			if err != nil {
				return fmt.Errorf("error parsing release date: %v", err)
			}
			releaseDateProto = timestamppb.New(parsedDate)
		}

		// Validate and parse the package list
		packageSet := make(map[string]struct{})
		for _, pkg := range *packageList {
			if _, exists := packageSet[pkg]; exists {
				return fmt.Errorf("duplicate package in the package list: %s", pkg)
			}
			packageSet[pkg] = struct{}{}
		}

		// Validate and parse the mode
		var downloadMode int32
		switch *mode {
		case "full":
			downloadMode = 1
		case "no-download":
			downloadMode = 2
		case "download-only":
			downloadMode = 3
		default:
			return fmt.Errorf("invalid mode. Use one of full, no-download, download-only")
		}

		request := &pb.UpdateSystemSoftwareRequest{
			Url:         *url,
			ReleaseDate: releaseDateProto,
			Mode:        pb.UpdateSystemSoftwareRequest_DownloadMode(downloadMode),
			DoNotReboot: !*reboot,
			PackageList: *packageList,
			Signature:   *signature,
		}

		ctx, cancel := context.WithTimeout(context.Background(), clientDialTimeoutInSeconds * time.Second)
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

		os, err := detectOS()
		if err != nil {
			return fmt.Errorf("error detected OS type: %v", err)
		}

		var timeout time.Duration = defaultSoftwareUpdateTimerInSeconds
		if os == "EMT" {
			timeout = emtSoftwareUpdateTimerInSeconds
		}

		ctx, cancel = context.WithTimeout(context.Background(), timeout * time.Second)
		defer cancel()

		resp, err := client.UpdateSystemSoftware(ctx, request)
		if err != nil {
			return fmt.Errorf("error updating system software: %v", err)
		}

		fmt.Printf("SOTA Command Response: %d-%s\n", resp.GetStatusCode(), string(resp.GetError()))

		return nil
	}
}
