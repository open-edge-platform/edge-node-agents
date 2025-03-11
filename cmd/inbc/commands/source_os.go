/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
 package commands

 import (
	 "github.com/spf13/cobra"
 )
 
 // SourceOSCmd returns a cobra command for the Source OS command
 func SourceOSCmd() *cobra.Command {
	 cmd := &cobra.Command{
		 Use:   "os",
		 Short: "Modifies the source files for OS Updates",
		 Long:  "Source command is used to creates a new /etc/apt/sources.list file with only the sources provided.",
	 }
 
	 // Add subcommands to source OS command
	 cmd.AddCommand(UpdateOSSourceCmd())
 
	 return cmd
 }
 