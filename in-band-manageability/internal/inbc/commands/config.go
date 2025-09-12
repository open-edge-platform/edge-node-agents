/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// Package commands are the commands that are used by the INBC tool.
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/in-band-manageability/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// ConfigLoadCmd returns the 'load' subcommand.
func ConfigLoadCmd() *cobra.Command {
	var socket string
	var uri, signature, hashAlgorithm string
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load a new configuration file",
		RunE:  handleConfigLoadCmd(&socket, &uri, &signature, &hashAlgorithm, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI to config file")
	cmd.Flags().StringVarP(&signature, "signature", "s", "", "Signature for config file")
	cmd.Flags().StringVar(&hashAlgorithm, "hash_algorithm", "", "Hash algorithm to use for signature verification (sha256, sha384, sha512). Default is sha384.")
	must(cmd.MarkFlagRequired("uri"))

	return cmd
}

// handleConfigLoadCmd is a helper function to handle the ConfigLoadCmd
func handleConfigLoadCmd(
	socket *string,
	uri *string,
	signature *string,
	hashAlgorithm *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("CONFIG LOAD command invoked.")

		if uri == nil || *uri == "" {
			return errors.New("uri is required")
		}

		// TODO: Validate signature against expected format
		// TODO: Add unittest test case for invalid signature format

		// Default to sha384 if not provided
		finalHashAlgorithm := "sha384"
		if hashAlgorithm != nil && *hashAlgorithm != "" {
			finalHashAlgorithm = *hashAlgorithm
		}

		request := &pb.LoadConfigRequest{
			Uri:           *uri,
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

		ctx, cancel = context.WithTimeout(context.Background(), configTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.LoadConfig(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing config load: %v", err)
		}
		if resp.StatusCode != 200 || resp.Error != "" {
			return fmt.Errorf("config load failed: %s", resp.Error)
		}

		fmt.Printf("CONFIG LOAD Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())
		return nil
	}
}

// ConfigGetCmd returns the 'get' subcommand.
func ConfigGetCmd() *cobra.Command {
	var socket string
	var path string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get key/value pairs from configuration file",
		RunE:  handleConfigGetCmd(&socket, &path, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Key path")
	must(cmd.MarkFlagRequired("path"))

	return cmd
}

// handleConfigGetCmd is a helper function to handle the ConfigGetCmd
func handleConfigGetCmd(
	socket *string,
	path *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("CONFIG GET command invoked.")

		if path == nil || *path == "" {
			return errors.New("path is required")
		}

		request := &pb.GetConfigRequest{
			Path: *path,
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

		ctx, cancel = context.WithTimeout(context.Background(), configTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.GetConfig(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing config get: %v", err)
		}

		fmt.Printf("CONFIG GET Response: %d-%s, value: %s\n", resp.GetStatusCode(), resp.GetError(), resp.GetValue())
		return nil
	}
}

// ConfigSetCmd returns the 'set' subcommand.
func ConfigSetCmd() *cobra.Command {
	var socket string
	var path string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set key/value pairs in configuration file",
		RunE:  handleConfigSetCmd(&socket, &path, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Key path and value (e.g. key:value)")
	must(cmd.MarkFlagRequired("path"))

	return cmd
}

// handleConfigSetCmd is a helper function to handle the ConfigSetCmd
func handleConfigSetCmd(
	socket *string,
	path *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("CONFIG SET command invoked.")

		if path == nil || *path == "" {
			return errors.New("path is required")
		}

		// Validate the path format (e.g., key:value)
		if !strings.Contains(*path, ":") {
			return fmt.Errorf("path must be in the format 'key:value', got: %s", *path)
		}

		request := &pb.SetConfigRequest{
			Path: *path,
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

		ctx, cancel = context.WithTimeout(context.Background(), configTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.SetConfig(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing config set: %v", err)
		}

		fmt.Printf("CONFIG SET Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())
		return nil
	}
}

// ConfigAppendCmd returns the 'append' subcommand.
func ConfigAppendCmd() *cobra.Command {
	var socket string
	var path string
	cmd := &cobra.Command{
		Use:   "append",
		Short: "Append to trustedRepositories",
		RunE:  handleConfigAppendCmd(&socket, &path, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Key path and value to append")
	must(cmd.MarkFlagRequired("path"))

	return cmd
}

// handleConfigAppendCmd is a helper function to handle the ConfigAppendCmd
func handleConfigAppendCmd(
	socket *string,
	path *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("CONFIG APPEND command invoked.")

		if path == nil || *path == "" {
			return errors.New("path is required")
		}

		// Validate the path format (e.g., key:value)
		if !strings.Contains(*path, ":") {
			return fmt.Errorf("path must be in the format 'key:value', got: %s", *path)
		}

		request := &pb.AppendConfigRequest{
			Path: *path,
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

		ctx, cancel = context.WithTimeout(context.Background(), configTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.AppendConfig(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing config append: %v", err)
		}

		fmt.Printf("CONFIG APPEND Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())
		return nil
	}
}

// ConfigRemoveCmd returns the 'remove' subcommand.
func ConfigRemoveCmd() *cobra.Command {
	var socket string
	var path string
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove from trustedRepositories",
		RunE:  handleConfigRemoveCmd(&socket, &path, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Key path and value to remove")
	must(cmd.MarkFlagRequired("path"))

	return cmd
}

// handleConfigRemoveCmd is a helper function to handle the ConfigRemoveCmd
func handleConfigRemoveCmd(
	socket *string,
	path *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("CONFIG REMOVE command invoked.")

		if path == nil || *path == "" {
			return errors.New("path is required")
		}

		// Validate the path format (e.g., key:value)
		if !strings.Contains(*path, ":") {
			return fmt.Errorf("path must be in the format 'key:value', got: %s", *path)
		}

		request := &pb.RemoveConfigRequest{
			Path: *path,
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

		ctx, cancel = context.WithTimeout(context.Background(), configTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.RemoveConfig(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing config remove: %v", err)
		}

		fmt.Printf("CONFIG REMOVE Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())
		return nil
	}
}
