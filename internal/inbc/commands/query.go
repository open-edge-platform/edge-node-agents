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
		if option == nil || *option == "" {
			*option = "all"
		}

		// Parse and validate query option
		queryOption, err := parseQueryOption(*option)
		if err != nil {
			return fmt.Errorf("invalid query option '%s': %v", *option, err)
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

		// Display response
		displayQueryResponse(resp, *option)
		return nil
	}
}

// parseQueryOption converts string to QueryOption enum
func parseQueryOption(option string) (pb.QueryOption, error) {
	switch strings.ToLower(option) {
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
		return pb.QueryOption_QUERY_OPTION_UNSPECIFIED, fmt.Errorf("valid options are: hw, fw, os, swbom, version, all")
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
	}
}

// displayHardwareInfo displays hardware information
func displayHardwareInfo(hw *pb.HardwareInfo) {
	if hw == nil {
		return
	}

	fmt.Println("\n=== Hardware Information ===")
	if hw.GetManufacturer() != "" {
		fmt.Printf("Manufacturer: %s\n", hw.GetManufacturer())
	}
	if hw.GetProduct() != "" {
		fmt.Printf("Product: %s\n", hw.GetProduct())
	}
	if hw.GetStepping() != "" {
		fmt.Printf("Stepping: %s\n", hw.GetStepping())
	}
	if hw.GetSku() != "" {
		fmt.Printf("SKU: %s\n", hw.GetSku())
	}
	if hw.GetModel() != "" {
		fmt.Printf("Model: %s\n", hw.GetModel())
	}
	if hw.GetSerialSum() != "" {
		fmt.Printf("Serial Sum: %s\n", hw.GetSerialSum())
	}
	if hw.GetSystemManufacturer() != "" {
		fmt.Printf("System Manufacturer: %s\n", hw.GetSystemManufacturer())
	}
	if hw.GetSystemProductName() != "" {
		fmt.Printf("System Product Name: %s\n", hw.GetSystemProductName())
	}
	if hw.GetCpuId() != "" {
		fmt.Printf("CPU ID: %s\n", hw.GetCpuId())
	}
	if hw.GetTotalPhysicalMemory() != "" {
		fmt.Printf("Total Physical Memory: %s\n", hw.GetTotalPhysicalMemory())
	}
	if hw.GetDiskInformation() != "" {
		fmt.Printf("Disk Information: %s\n", hw.GetDiskInformation())
	}
}

// displayFirmwareInfo displays firmware information
func displayFirmwareInfo(fw *pb.FirmwareInfo) {
	if fw == nil {
		return
	}

	fmt.Println("\n=== Firmware Information ===")
	if fw.GetBootFwDate() != nil {
		fmt.Printf("Boot Firmware Date: %s\n", fw.GetBootFwDate().AsTime().Format(time.RFC3339))
	}
	if fw.GetBootFwVendor() != "" {
		fmt.Printf("Boot Firmware Vendor: %s\n", fw.GetBootFwVendor())
	}
	if fw.GetBootFwVersion() != "" {
		fmt.Printf("Boot Firmware Version: %s\n", fw.GetBootFwVersion())
	}
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
	if os.GetOsType() != "" {
		fmt.Printf("OS Type: %s\n", os.GetOsType())
	}
	if os.GetOsVersion() != "" {
		fmt.Printf("OS Version: %s\n", os.GetOsVersion())
	}
	if os.GetOsReleaseDate() != nil {
		fmt.Printf("OS Release Date: %s\n", os.GetOsReleaseDate().AsTime().Format(time.RFC3339))
	}
	if os.GetOsInformation() != "" {
		fmt.Printf("OS Information: %s\n", os.GetOsInformation())
	}
}

// displaySWBOMInfo displays software BOM information
func displaySWBOMInfo(swbom *pb.SWBOMInfo) {
	if swbom == nil {
		return
	}

	fmt.Println("\n=== Software Bill of Materials ===")
	if swbom.GetCollectionTimestamp() != nil {
		fmt.Printf("Collection Timestamp: %s\n", swbom.GetCollectionTimestamp().AsTime().Format(time.RFC3339))
	}
	if swbom.GetCollectionMethod() != "" {
		fmt.Printf("Collection Method: %s\n", swbom.GetCollectionMethod())
	}

	packages := swbom.GetPackages()
	if len(packages) > 0 {
		fmt.Printf("Total Packages: %d\n", len(packages))
		fmt.Println("\nPackages:")
		for i, pkg := range packages {
			if i >= 10 { // Limit display to first 10 packages
				fmt.Printf("... and %d more packages\n", len(packages)-10)
				break
			}
			fmt.Printf("  - %s", pkg.GetName())
			if pkg.GetVersion() != "" {
				fmt.Printf(" (%s)", pkg.GetVersion())
			}
			if pkg.GetVendor() != "" {
				fmt.Printf(" by %s", pkg.GetVendor())
			}
			fmt.Println()
		}
	}
}

// displayVersionInfo displays version information
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
	if version.GetBuildDate() != nil {
		fmt.Printf("Build Date: %s\n", version.GetBuildDate().AsTime().Format(time.RFC3339))
	}
	if version.GetGitCommit() != "" {
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
	}
}

// displayAllInfo displays all system information
func displayAllInfo(all *pb.AllInfo) {
	if all == nil {
		return
	}

	fmt.Println("\n=== All System Information ===")

	if all.GetHardware() != nil {
		displayHardwareInfo(all.GetHardware())
	}

	if all.GetFirmware() != nil {
		displayFirmwareInfo(all.GetFirmware())
	}

	if all.GetOsInfo() != nil {
		displayOSInfo(all.GetOsInfo())
	}

	if all.GetVersion() != nil {
		displayVersionInfo(all.GetVersion())
	}

	if all.GetPowerCapabilities() != "" {
		fmt.Printf("\nPower Capabilities: %s\n", all.GetPowerCapabilities())
	}

	additionalInfo := all.GetAdditionalInfo()
	if len(additionalInfo) > 0 {
		fmt.Println("\nAdditional Information:")
		for _, info := range additionalInfo {
			fmt.Printf("  - %s\n", info)
		}
	}
}
