// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package comms

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	proto "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/cpu"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/disk"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/gpu"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/memory"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/network"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/system"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/usb"
	"github.com/open-edge-platform/edge-node-agents/hardware-discovery-agent/internal/utils"
)

const connTimeout = 5 * time.Second

var log = logger.Logger

type Client struct {
	ServerAddr       string
	Dialer           grpc.DialOption
	Transport        grpc.DialOption
	GrpcConn         *grpc.ClientConn
	SouthboundClient proto.HostmgrClient
}

func WithNetworkDialer(serverAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return net.Dial("tcp", serverAddr)
		})
	}
}

// NewClient creates grpc client to Edge Infrastructure Manager southbound API
// by default it uses tcp network dialer.
func NewClient(serverAddr string, tlsConfig *tls.Config, options ...func(*Client)) *Client {
	cli := &Client{}
	cli.ServerAddr = serverAddr
	cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	WithNetworkDialer(serverAddr)(cli)

	// options can be used to override default values, e.g. from unit tests
	for _, o := range options {
		o(cli)
	}
	return cli
}

// FIXME: SA1019: grpc.Dial is deprecated: use NewClient instead.
// Connect client method establishes GRPC connection with Edge Infrastructure Manager.
// In case of an error the function will return the error.
func (cli *Client) Connect() (err error) {
	cli.GrpcConn, err = grpc.Dial(cli.ServerAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(connTimeout)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		log.Errorf("Connection to Edge Infrastructure Manager failed : %v", err)
		return err
	}
	cli.SouthboundClient = proto.NewHostmgrClient(cli.GrpcConn)
	return nil
}

// UpdateHostSystemInfoByGuid client method sends UpdateHostSystemInfoByGuidRequest message to the server. It receives UpdateHostSystemInfoResponse message.
// The message will be an empty string if successful.
// In case of an error, the function will return an error.
func (cli *Client) UpdateHostSystemInfoByGUID(ctx context.Context, guid string, systemInfo *proto.SystemInfo) (*proto.UpdateHostSystemInfoByGUIDResponse, error) {
	log.Debugf("Sending System info: %+v", systemInfo)
	updateHostSystemInfoByGUIDRequest := proto.UpdateHostSystemInfoByGUIDRequest{HostGuid: guid, SystemInfo: systemInfo}
	updateHostSystemInfoByGUIDResponsePtr, err := cli.SouthboundClient.UpdateHostSystemInfoByGUID(ctx, &updateHostSystemInfoByGUIDRequest)
	if err != nil {
		log.Errorf("The protobuf UpdateHostSystemInfoByGUID function failed! : %v", err)
		return nil, err
	}
	log.Infof("HW Discovery Agent comms: UpdateHostSystemInfoByGUIDRequest sent successfully")
	return updateHostSystemInfoByGUIDResponsePtr, nil
}

// ConnectToEdgeInfrastructureManager function uses comms API's and the Edge Infrastructure Manager address to connect with GRPC the Edge Infrastructure Manager server.
// The function will return the Client struct
// In case of error the function will print the error message, will sleep for some period time and will try again.
func ConnectToEdgeInfrastructureManager(serverAddr string, tlsConfig *tls.Config) (*Client, error) {
	hostManager := NewClient(serverAddr, tlsConfig)

	err := hostManager.Connect()
	if err != nil {
		return nil, err
	}
	return hostManager, nil
}

func parseSystemInfo(serialNumber string, productName string, bmcAddr string, osInfo *system.Os, biosInfo *system.Bios, cpu *cpu.CPU,
	storage []*disk.Disk, gpu []*gpu.Gpu, mem uint64, networks []*network.Network, bmType proto.BmInfo_BmType, usbInfo []*usb.Usb) *proto.SystemInfo {

	gpuList := []*proto.SystemGPU{}
	for _, gpuDetails := range gpu {
		gpuList = append(gpuList, &proto.SystemGPU{
			PciId:       gpuDetails.PciID,
			Product:     gpuDetails.Product,
			Vendor:      gpuDetails.Vendor,
			Name:        gpuDetails.Name,
			Description: gpuDetails.Description,
			Features:    gpuDetails.Features,
		})
	}

	diskList := []*proto.SystemDisk{}
	for _, diskDetails := range storage {
		diskList = append(diskList, &proto.SystemDisk{
			SerialNumber: diskDetails.SerialNum,
			Name:         diskDetails.Name,
			Vendor:       diskDetails.Vendor,
			Model:        diskDetails.Model,
			Size:         diskDetails.Size,
			Wwid:         diskDetails.Wwid,
		})
	}

	networkList := []*proto.SystemNetwork{}
	for _, networkDetails := range networks {
		ipAddressList := []*proto.IPAddress{}
		for _, ipAddress := range networkDetails.IPAddresses {
			ipAddressList = append(ipAddressList, &proto.IPAddress{
				IpAddress:         ipAddress.IPAddress,
				NetworkPrefixBits: ipAddress.NetPrefBits,
				ConfigMode:        ipAddress.ConfigMode,
			})
		}
		networkList = append(networkList, &proto.SystemNetwork{
			Name:                networkDetails.Name,
			PciId:               networkDetails.PciID,
			Mac:                 networkDetails.Mac,
			LinkState:           networkDetails.LinkState,
			CurrentSpeed:        networkDetails.CurrentSpeed,
			CurrentDuplex:       networkDetails.CurrentDuplex,
			SupportedLinkMode:   networkDetails.SupportedLinkMode,
			AdvertisingLinkMode: networkDetails.AdvertisingLinkMode,
			Features:            networkDetails.Features,
			Sriovenabled:        networkDetails.SriovEnabled,
			Sriovnumvfs:         networkDetails.SriovNumVfs,
			SriovVfsTotal:       networkDetails.SriovVfsTotal,
			PeerName:            networkDetails.PeerName,
			PeerDescription:     networkDetails.PeerDescription,
			PeerMac:             networkDetails.PeerMac,
			PeerMgmtIp:          networkDetails.PeerManagementIP,
			PeerPort:            networkDetails.PeerPort,
			IpAddresses:         ipAddressList,
			Mtu:                 networkDetails.Mtu,
			BmcNet:              networkDetails.BmcNet,
		})
	}

	osKern := proto.OsKernel{}
	if osInfo.Kernel != nil {
		kernConfig := []*proto.Config{}
		for _, config := range osInfo.Kernel.Config {
			kernConfig = append(kernConfig, &proto.Config{
				Key:   config.Key,
				Value: config.Value,
			})
		}
		osKern = proto.OsKernel{
			Version: osInfo.Kernel.Version,
			Config:  kernConfig,
		}
	}

	osRelease := proto.OsRelease{}
	if osInfo.Release != nil {
		relMetadata := []*proto.Metadata{}
		for _, metadata := range osInfo.Release.Metadata {
			relMetadata = append(relMetadata, &proto.Metadata{
				Key:   metadata.Key,
				Value: metadata.Value,
			})
		}
		osRelease = proto.OsRelease{
			Id:       osInfo.Release.ID,
			Version:  osInfo.Release.Version,
			Metadata: relMetadata,
		}
	}

	usbList := []*proto.SystemUSB{}
	for _, usbDetails := range usbInfo {
		interfacesList := []*proto.Interfaces{}
		for _, interfaces := range usbDetails.Interfaces {
			interfacesList = append(interfacesList, &proto.Interfaces{Class: interfaces.Class})
		}
		usbList = append(usbList, &proto.SystemUSB{
			Class:       usbDetails.Class,
			Idvendor:    usbDetails.VendorId,
			Idproduct:   usbDetails.ProductId,
			Bus:         usbDetails.Bus,
			Addr:        usbDetails.Address,
			Description: usbDetails.Description,
			Serial:      usbDetails.Serial,
			Interfaces:  interfacesList,
		})
	}

	cpuInfo := proto.SystemCPU{}
	if cpu != nil {
		if cpu.Topology != nil {
			sockets := []*proto.Socket{}
			for _, socket := range cpu.Topology.Sockets {
				coreGroups := []*proto.CoreGroup{}
				for _, coreGroup := range socket.CoreGroups {
					coreGroups = append(coreGroups, &proto.CoreGroup{
						CoreType: coreGroup.Type,
						CoreList: coreGroup.List,
					})
				}
				sockets = append(sockets, &proto.Socket{
					SocketId:   socket.SocketID,
					CoreGroups: coreGroups,
				})
			}
			cpuInfo = proto.SystemCPU{
				Arch:        cpu.Arch,
				Vendor:      cpu.Vendor,
				Model:       cpu.Model,
				Sockets:     cpu.Sockets,
				Cores:       cpu.Cores,
				Threads:     cpu.Threads,
				Features:    cpu.Features,
				CpuTopology: &proto.CPUTopology{Sockets: sockets},
			}
		} else {
			cpuInfo = proto.SystemCPU{
				Arch:     cpu.Arch,
				Vendor:   cpu.Vendor,
				Model:    cpu.Model,
				Sockets:  cpu.Sockets,
				Cores:    cpu.Cores,
				Threads:  cpu.Threads,
				Features: cpu.Features,
			}
		}
	}

	systemInfo := &proto.SystemInfo{
		HwInfo: &proto.HWInfo{
			SerialNum:   serialNumber,
			ProductName: productName,
			Cpu:         &cpuInfo,
			Memory:      &proto.SystemMemory{Size: mem},
			Storage:     &proto.Storage{Disk: diskList},
			Gpu:         gpuList,
			Network:     networkList,
			Usb:         usbList,
		},
		OsInfo: &proto.OsInfo{
			Kernel:  &osKern,
			Release: &osRelease,
		},
		BmCtlInfo: &proto.BmInfo{
			BmType: bmType,
			BmcInfo: &proto.BmcInfo{
				BmIp: bmcAddr,
			},
		},
		BiosInfo: &proto.BiosInfo{
			Version:     biosInfo.Version,
			ReleaseDate: biosInfo.RelDate,
			Vendor:      biosInfo.Vendor,
		},
	}

	return systemInfo
}

func GenerateSystemInfoRequest(executor utils.CmdExecutor) *proto.SystemInfo {
	storage, err := disk.GetDiskList(executor)
	if err != nil {
		log.Errorf("unable to get disk description : %v", err)
	}

	sn, err := system.GetSerialNumber(executor)
	if err != nil {
		log.Errorf("unable to get system serial number : %v", err)
	}

	productName, err := system.GetProductName(executor)
	if err != nil {
		log.Errorf("unable to get system product name : %v", err)
	}

	osInfo, err := system.GetOsInfo(executor)
	if err != nil {
		log.Errorf("unable to get system OS information : %v", err)
	}

	biosInfo, err := system.GetBiosInfo(executor)
	if err != nil {
		log.Errorf("unable to get system BIOS information : %v", err)
	}

	cpu, err := cpu.GetCPUList(executor)
	if err != nil {
		log.Errorf("unable to get cpu description : %v", err)
	}

	gpu, err := gpu.GetGpuList(executor)
	if err != nil {
		log.Errorf("unable to get gpu description : %v", err)
	}

	mem, err := memory.GetMemory(executor)
	if err != nil {
		log.Errorf("unable to get memory description : %v", err)
	}

	networkList, bmType, bmcAddr, err := network.GetNICList(executor)
	if err != nil {
		log.Errorf("unable to get network interface description : %v", err)
	}

	usbList, err := usb.GetUsbList(executor)
	if err != nil {
		log.Errorf("unable to get usb description : %v", err)
	}

	return parseSystemInfo(sn, productName, bmcAddr, osInfo, biosInfo, cpu, storage, gpu, mem, networkList, bmType, usbList)
}
