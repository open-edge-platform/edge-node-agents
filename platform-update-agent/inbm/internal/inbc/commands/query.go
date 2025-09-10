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

	pb "github.com/open-edge-platform/edge-node-agents/platform-update-agent/inbm/pkg/api/inbd/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// QueryCmd returns the 'query' command.
func QueryCmd() *cobra.Command {
	var socket string
	var option string
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query system information",
		Long: `Query system information including hardware, firmware, OS, software BOM, and version details.

Available options:
  hw      - Hardware information (manufacturer, product, CPU, memory, disk)
  fw      - Firmware information (BIOS vendor, version, release date)
  os      - Operating system information (type, version, release date)
  swbom   - Software Bill of Materials (installed packages)
  version - Version information (INBM version, build date, git commit)
  all     - All available information`,
		Example: `  inbc query
  inbc query --option hw
  inbc query --option fw
  inbc query --option os
  inbc query --option swbom
  inbc query --option version
  inbc query --option all`,
		RunE: handleQueryCmd(&socket, &option, Dial),
	}

	cmd.Flags().StringVar(&socket, "socket", "/var/run/inbd.sock", "UNIX domain socket path")
	cmd.Flags().StringVarP(&option, "option", "o", "all", "Query option (hw, fw, os, swbom, version, all)")

	return cmd
}

// handleQueryCmd is a helper function to handle the QueryCmd
func handleQueryCmd(
	socket *string,
	option *string,
	dialer func(context.Context, string) (pb.InbServiceClient, grpc.ClientConnInterface, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("QUERY command invoked.")

		// Use default value if option is empty
		var optionValue string
		if option == nil {
			optionValue = "all"
		} else if *option == "" {
			optionValue = "all"
		} else {
			optionValue = *option
		}

		// Parse and validate query option
		queryOption, err := parseQueryOption(optionValue)
		if err != nil {
			return fmt.Errorf("invalid query option '%s': %v", optionValue, err)
		}

		request := &pb.QueryRequest{
			Option: queryOption,
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

		ctx, cancel = context.WithTimeout(context.Background(), queryTimeoutInSeconds*time.Second)
		defer cancel()

		resp, err := client.Query(ctx, request)
		if err != nil {
			return fmt.Errorf("error performing query: %v", err)
		}

		if resp == nil {
			return fmt.Errorf("received nil response from server")
		}

		// Display response
		displayQueryResponse(resp, optionValue)
		return nil
	}
}

// parseQueryOption converts string to QueryOption enum
func parseQueryOption(option string) (pb.QueryOption, error) {
	option = strings.ToLower(option)

	// Check for empty string after trimming
	if option == "" {
		return pb.QueryOption_QUERY_OPTION_UNSPECIFIED, fmt.Errorf("query option cannot be empty")
	}

	switch option {
	case "hw", "hardware":
		return pb.QueryOption_QUERY_OPTION_HARDWARE, nil
	case "fw", "firmware":
		return pb.QueryOption_QUERY_OPTION_FIRMWARE, nil
	case "os", "operating-system":
		return pb.QueryOption_QUERY_OPTION_OS, nil
	case "swbom", "software-bom":
		return pb.QueryOption_QUERY_OPTION_SWBOM, nil
	case "version", "ver":
		return pb.QueryOption_QUERY_OPTION_VERSION, nil
	case "all":
		return pb.QueryOption_QUERY_OPTION_ALL, nil
	default:
		return pb.QueryOption_QUERY_OPTION_UNSPECIFIED, fmt.Errorf("invalid query option '%s'. Valid options: hw, fw, os, swbom, version, all", option)
	}
}

// displayQueryResponse formats and displays the query response
func displayQueryResponse(resp *pb.QueryResponse, option string) {
	fmt.Printf("QUERY Response: %d-%s\n", resp.GetStatusCode(), resp.GetError())

	if resp.GetSuccess() && resp.GetData() != nil {
		fmt.Printf("Query Type: %s\n", option)
		fmt.Printf("Data Type: %s\n", resp.GetData().GetType())

		if resp.GetData().GetTimestamp() != nil {
			fmt.Printf("Timestamp: %s\n", resp.GetData().GetTimestamp().AsTime().Format(time.RFC3339))
		}

		// Display specific data based on the values oneof
		switch values := resp.GetData().GetValues().(type) {
		case *pb.QueryData_Hardware:
			displayHardwareInfo(values.Hardware)
		case *pb.QueryData_Firmware:
			displayFirmwareInfo(values.Firmware)
		case *pb.QueryData_OsInfo:
			displayOSInfo(values.OsInfo)
		case *pb.QueryData_Swbom:
			displaySWBOMInfo(values.Swbom)
		case *pb.QueryData_Version:
			displayVersionInfo(values.Version)
		case *pb.QueryData_AllInfo:
			displayAllInfo(values.AllInfo)
		default:
			fmt.Println("No specific data available")
		}
	} else {
		// Show error details for failed responses
		if !resp.GetSuccess() {
			fmt.Printf("Query failed: %s\n", resp.GetError())
		}
	}
}

// displayHardwareInfo displays hardware information
func displayHardwareInfo(hw *pb.HardwareInfo) {
	if hw == nil {
		return
	}

	fmt.Println("\n=== Hardware Information ===")
	if hw.GetCpuId() != "" {
		fmt.Printf("CPU ID: %s\n", hw.GetCpuId())
	}
	if hw.GetTotalPhysicalMemory() != "" {
		fmt.Printf("Total Physical Memory: %s\n", hw.GetTotalPhysicalMemory())
	}
	if hw.GetDiskInformation() != "" {
		fmt.Printf("Disk Information: %s\n", hw.GetDiskInformation())
	}
	if hw.GetSystemManufacturer() != "" {
		fmt.Printf("System Manufacturer: %s\n", hw.GetSystemManufacturer())
	}
	if hw.GetSystemProductName() != "" {
		fmt.Printf("System Product Name: %s\n", hw.GetSystemProductName())
	}
}

// displayFirmwareInfo displays firmware information
func displayFirmwareInfo(fw *pb.FirmwareInfo) {
	if fw == nil {
		return
	}

	fmt.Println("\n=== Firmware Information ===")
	if fw.GetBiosVendor() != "" {
		fmt.Printf("BIOS Vendor: %s\n", fw.GetBiosVendor())
	}
	if fw.GetBiosVersion() != "" {
		fmt.Printf("BIOS Version: %s\n", fw.GetBiosVersion())
	}
	if fw.GetBiosReleaseDate() != nil {
		fmt.Printf("BIOS Release Date: %s\n", fw.GetBiosReleaseDate().AsTime().Format(time.RFC3339))
	}
}

// displayOSInfo displays operating system information
func displayOSInfo(os *pb.OSInfo) {
	if os == nil {
		return
	}

	fmt.Println("\n=== Operating System Information ===")
	if os.GetOsInformation() != "" {
		fmt.Printf("OS Information: %s\n", os.GetOsInformation())
	}
}

// displaySWBOMInfo displays software BOM information with ALL fields
func displaySWBOMInfo(swbom *pb.SWBOMInfo) {
	if swbom == nil {
		return
	}

	fmt.Println("\n=== Software Bill of Materials ===")

	// Collection timestamp
	if swbom.GetCollectionTimestamp() != nil {
		fmt.Printf("Collection Timestamp: %s\n", swbom.GetCollectionTimestamp().AsTime().Format(time.RFC3339))
	}

	// Collection method
	if swbom.GetCollectionMethod() != "" {
		fmt.Printf("Collection Method: %s\n", swbom.GetCollectionMethod())
	}

	// Packages with ALL fields
	packages := swbom.GetPackages()
	if len(packages) > 0 {
		fmt.Printf("Total Packages: %d\n", len(packages))
		fmt.Println("\nPackages:")
		for i, pkg := range packages {
			if i >= 10 { // Limit display to first 10 packages
				fmt.Printf("... and %d more packages\n", len(packages)-10)
				break
			}

			// Package name (required field)
			fmt.Printf("  - %s", pkg.GetName())

			// Package version
			if pkg.GetVersion() != "" {
				fmt.Printf(" (%s)", pkg.GetVersion())
			}

			// Package vendor
			if pkg.GetVendor() != "" {
				fmt.Printf(" by %s", pkg.GetVendor())
			}

			// Package type
			if pkg.GetType() != "" {
				fmt.Printf(" [%s]", pkg.GetType())
			}

			// Package architecture
			if pkg.GetArchitecture() != "" {
				fmt.Printf(" (%s)", pkg.GetArchitecture())
			}

			fmt.Println()

			// Additional details (indented)
			if pkg.GetDescription() != "" {
				fmt.Printf("    Description: %s\n", pkg.GetDescription())
			}
			if pkg.GetLicense() != "" {
				fmt.Printf("    License: %s\n", pkg.GetLicense())
			}
			if pkg.GetInstallDate() != nil {
				fmt.Printf("    Install Date: %s\n", pkg.GetInstallDate().AsTime().Format(time.RFC3339))
			}
		}
	}
}

// displayVersionInfo displays version information with ALL fields
func displayVersionInfo(version *pb.VersionInfo) {
	if version == nil {
		return
	}

	fmt.Println("\n=== Version Information ===")
	if version.GetVersion() != "" {
		fmt.Printf("Version: %s\n", version.GetVersion())
	}
	if version.GetInbmVersionCommit() != "" {
		fmt.Printf("INBM Version Commit: %s\n", version.GetInbmVersionCommit())
	}
	if version.GetGitCommit() != "" {
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
	}
	if version.GetBuildDate() != nil {
		fmt.Printf("Build Date: %s\n", version.GetBuildDate().AsTime().Format(time.RFC3339))
	}
}

// displayPowerCapabilities displays power capabilities with ALL fields
func displayPowerCapabilities(power *pb.PowerCapabilitiesInfo) {
	if power == nil {
		return
	}

	fmt.Println("\n=== Power Capabilities ===")
	fmt.Printf("Shutdown: %t\n", power.GetShutdown())
	fmt.Printf("Reboot: %t\n", power.GetReboot())
	fmt.Printf("Suspend: %t\n", power.GetSuspend())
	fmt.Printf("Hibernate: %t\n", power.GetHibernate())
	if power.GetCapabilitiesJson() != "" {
		fmt.Printf("Capabilities JSON: %s\n", power.GetCapabilitiesJson())
	}
}

// displayAllInfo displays all system information with ALL fields
func displayAllInfo(all *pb.AllInfo) {
	if all == nil {
		return
	}

	fmt.Println("\n=== All System Information ===")

	// Hardware information
	if all.GetHardware() != nil {
		displayHardwareInfo(all.GetHardware())
	}

	// Firmware information
	if all.GetFirmware() != nil {
		displayFirmwareInfo(all.GetFirmware())
	}

	// OS information
	if all.GetOsInfo() != nil {
		displayOSInfo(all.GetOsInfo())
	}

	// Version information
	if all.GetVersion() != nil {
		displayVersionInfo(all.GetVersion())
	}

	// Power capabilities
	if all.GetPowerCapabilities() != nil {
		displayPowerCapabilities(all.GetPowerCapabilities())
	}

	// Software BOM
	if all.GetSwbom() != nil {
		displaySWBOMInfo(all.GetSwbom())
	}

	// Additional information
	additionalInfo := all.GetAdditionalInfo()
	if len(additionalInfo) > 0 {
		fmt.Println("\n=== Additional Information ===")
		for _, info := range additionalInfo {
			fmt.Printf("  - %s\n", info)
		}
	}
}
