/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/intel/intel-inb-manageability/cmd/inbc/commands"
)

// Version is set with linker flags at build time.
var Version string

func main() {
	// Root command and persistent flags
	rootCmd := &cobra.Command{
		Use:     "inbc",
		Short:   "INBC - CLI for Intel Manageability",
		Version: Version,
		Long:    `INBC is a CLI to access and perform different manageability commands.`,
	}
	verbose := rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(commands.SOTACmd())

	// Execute CLI
	if err := rootCmd.Execute(); err != nil {
		if *verbose {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
