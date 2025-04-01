// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package clientapi

import (
	"context"
	"crypto/tls"
	"net"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logger"

	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
)

var log = logger.Logger
var LastResp *pb.GetTelemetryConfigResponse
var LastProfile []ProfileGroup

const CONN_TIMEOUT = 5 * time.Second

type Client struct {
	ServerAddr       string
	Dialer           grpc.DialOption
	Transport        grpc.DialOption
	GrpcConn         *grpc.ClientConn
	SouthboundClient pb.TelemetryMgrClient
}

type ProfileGroup struct {
	collectorKind pb.CollectorKind
	resourceKind  pb.TelemetryResourceKind
}

func (s ProfileGroup) isEqual(t ProfileGroup) bool {
	return s.collectorKind == t.collectorKind && s.resourceKind == t.resourceKind
}

// Define enum constants with specific values
const (
	MetricHost    int = 0
	MetricCluster int = 1
	LogHost       int = 2
	LogCluster    int = 3
)

func WithNetworkDialer(serverAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", serverAddr)
		})
	}
}

// NewClient creates grpc client to server southbound API
// by default it uses tcp network dialer
func NewClient(serverAddr string, tlsConfig *tls.Config, devMode bool, options ...func(*Client)) *Client {
	cli := &Client{}
	cli.ServerAddr = serverAddr
	if !devMode {
		cli.Transport = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		cli.Transport = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	WithNetworkDialer(serverAddr)(cli)

	// options can be used to override default values, e.g. from unit tests
	for _, o := range options {
		o(cli)
	}
	return cli
}

// Connect client method establishes GRPC connection with server.
// In case of an error the function will return the error.
func (cli *Client) Connect() (err error) {
	cli.GrpcConn, err = grpc.NewClient(cli.ServerAddr, cli.Transport, cli.Dialer,
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(CONN_TIMEOUT)))
	if err != nil {
		log.Errorf("Connection to server failed : %v", err)
		return err
	}
	cli.SouthboundClient = pb.NewTelemetryMgrClient(cli.GrpcConn)
	return nil
}

// ConnectToTelemetryManager function uses comms API's and the Telemetry Manager address to connect with GRPC the Telemetry Manager server.
// The function will return the Client struct
// In case of error the function will print the error message, will sleep for some period time and will try again.
func ConnectToTelemetryManager(ctx context.Context, serverAddr string, devMode bool) (*Client, error) {
	tlsConfig, err := utils.GetAuthConfig(ctx, nil)
	if err != nil {
		log.Fatalf("TLS configuration creation failed! Error: %v", err)
	}
	server := NewClient(serverAddr, tlsConfig, devMode)

	err = server.Connect()

	if err != nil {
		return nil, err
	}
	return server, nil
}

func GetConfig(ctx context.Context, cli pb.TelemetryMgrClient, nodeId string, jwtTokenPath string) (*pb.GetTelemetryConfigResponse, error) {
	if _, err := os.Stat(jwtTokenPath); os.IsNotExist(err) {
		return nil, err
	}
	ctxAuth := utils.GetAuthContext(ctx, jwtTokenPath)
	resp, err := cli.GetTelemetryConfigByGUID(ctxAuth, &pb.GetTelemetryConfigByGuidRequest{Guid: nodeId})
	return resp, err
}

func CheckIfChanged(latestCfg *pb.GetTelemetryConfigResponse) ([4]bool, [4]bool) {

	var maskDirty [4]bool
	var maskInit [4]bool

	// Check if latestCfg is nil
	if latestCfg == nil {
		return maskDirty, maskInit
	}

	var cfgPrev []*pb.GetTelemetryConfigResponse_TelemetryCfg
	if LastResp != nil && LastResp.Cfg != nil {
		cfgPrev = LastResp.Cfg
	}

	cfgNew := latestCfg.Cfg
	isChange := false

	// check if any interval changes
	for _, itxNew := range cfgNew {
		isChange = true //assumed there is a change by default
		for _, itxPrev := range cfgPrev {
			//if no change detected
			if itxNew.Input == itxPrev.Input && itxNew.Interval == itxPrev.Interval && itxNew.Level == itxPrev.Level {
				isChange = false
				break
			}
		}
		if isChange {
			mIdx := getMaskIdx(itxNew)
			maskDirty[mIdx] = true
		}
	}

	//check if items have been removed from previous
	for _, itxPrev := range cfgPrev {
		isRemoved := true
		for _, itxNew := range cfgNew {
			if itxNew.Input == itxPrev.Input {
				isRemoved = false
			}
		}
		if isRemoved { //found an item has been remove
			mIdx := getMaskIdx(itxPrev)
			maskDirty[mIdx] = true
		}
	}
	// convert config list to profile list
	var profileList []ProfileGroup
	for _, itxNew := range cfgNew {
		profileList = append(profileList, convertConfig(itxNew))
	}
	newProfile := removeDuplicates(profileList)
	// check item removed
	for _, itxPrev := range LastProfile {
		isRemoved := true
		for _, itxNew := range newProfile {
			if itxNew.isEqual(itxPrev) {
				isRemoved = false
			}
		}
		if isRemoved { //found an item has been remove
			missingprofile := &pb.GetTelemetryConfigResponse_TelemetryCfg{
				Kind: itxPrev.collectorKind,
				Type: itxPrev.resourceKind,
			}
			mIdx := getMaskIdx(missingprofile)
			maskInit[mIdx] = true
			maskDirty[mIdx] = true
		}
	}

	LastProfile = assignArray(newProfile)
	LastResp = latestCfg
	return maskDirty, maskInit
}

func convertConfig(source *pb.GetTelemetryConfigResponse_TelemetryCfg) ProfileGroup {
	return ProfileGroup{
		collectorKind: source.Kind,
		resourceKind:  source.Type,
	}
}

func removeDuplicates(arr []ProfileGroup) []ProfileGroup {
	result := []ProfileGroup{}
	for i := 0; i < len(arr); i++ {
		exists := false
		for j := 0; j < len(result); j++ {
			if arr[i].isEqual(result[j]) {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, arr[i])
		}
	}
	return result
}

func assignArray(src []ProfileGroup) []ProfileGroup {
	dest := make([]ProfileGroup, len(src))
	copy(dest, src)
	return dest
}

func getMaskIdx(itx *pb.GetTelemetryConfigResponse_TelemetryCfg) int {

	if itx.Kind == pb.CollectorKind_COLLECTOR_KIND_HOST && itx.Type == pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS {
		return MetricHost
	} else if itx.Kind == pb.CollectorKind_COLLECTOR_KIND_CLUSTER && itx.Type == pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS {
		return MetricCluster
	} else if itx.Kind == pb.CollectorKind_COLLECTOR_KIND_HOST && itx.Type == pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS {
		return LogHost
	} else {
		return LogCluster
	}

}
