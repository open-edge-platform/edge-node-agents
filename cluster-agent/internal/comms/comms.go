// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// package comms provides communication with Cluster Orchestrator via its southbound gRPC API.
// The Cluster Agent will be a GRPC client and it uses a protobuf interfaces for the messages:
// It sends RegisterClusterRequest message to the server. It receives RegisterClusterResponse message
// That contains RegisterClusterCommand. After a seccussful connection the function will execute the
// requested command.
//
// In case of any failure the code will print the relevant error.
package comms

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	proto "github.com/open-edge-platform/cluster-api-provider-intel/pkg/api/proto"
	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

const CONN_TIMEOUT = 5 * time.Second

var log = logger.Logger

type Client struct {
	ServerAddr            string
	Dialer                grpc.DialOption
	Transport             grpc.DialOption
	GrpcConn              *grpc.ClientConn
	CoSouthboundClient    proto.ClusterOrchestratorSouthboundClient
	RegisterToClusterOrch func(ctx context.Context, guid string) (installCmd, uninstallCmd string)
}

// Helper function for dailing to the server.
func WithNetworkDialer(serverAddr string) func(*Client) {
	return func(s *Client) {
		s.Dialer = grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", serverAddr)
		})
	}
}

// NewClient creates grpc client to Cluster Orchestrator southbound API
// by default it uses tcp network dialer and insecure transport
func NewClient(serverAddr string, tlsConfig *tls.Config, options ...func(*Client)) *Client {
	cli := &Client{}
	cli.ServerAddr = serverAddr
	cli.Transport = grpc.WithTransportCredentials(insecure.NewCredentials())
	cli.RegisterToClusterOrch = cli.registerToClusterOrch

	WithNetworkDialer(serverAddr)(cli)

	// options can be used to override default values, e.g. from unit tests
	for _, o := range options {
		o(cli)
	}
	return cli
}

func codeToLevel(_ codes.Code) logrus.Level {
	return logrus.DebugLevel
}

// FIXME: SA1019: grpc.Dial is deprecated: use NewClient instead.
// Connect client method establishes GRPC connection with a Cluster Orchestrator Server.
// In case of an error the function will return the error.
func (cli *Client) Connect() (err error) {
	cli.GrpcConn, err = grpc.Dial(cli.ServerAddr, cli.Transport, cli.Dialer, //nolint:staticcheck
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(log, grpc_logrus.WithLevels(codeToLevel))),
		grpc.WithStreamInterceptor(grpc_logrus.StreamClientInterceptor(log, grpc_logrus.WithLevels(codeToLevel))),
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(CONN_TIMEOUT)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connection to %v failed: %v", cli.ServerAddr, err)
	}
	cli.CoSouthboundClient = proto.NewClusterOrchestratorSouthboundClient(cli.GrpcConn)
	return nil
}

// Register client method sends RegisterClusterRequest message to the server. It receives RegisterClusterResponse message
// That contains RegisterClusterCommand.
// In case of an error the function will return the error.
func (cli *Client) Register(ctx context.Context, nodeAuthToken string) (*proto.RegisterClusterResponse, error) {
	registerClusterRequest := proto.RegisterClusterRequest{NodeGuid: nodeAuthToken}

	registerClusterResponsePtr, err := cli.CoSouthboundClient.RegisterCluster(ctx, &registerClusterRequest)
	if err != nil {
		log.Errorf("Register error: %v", err)
		return nil, err
	}
	if registerClusterResponsePtr.GetRes() == proto.RegisterClusterResponse_ERROR {
		err = fmt.Errorf("ERROR response from cluster orchestrator")
		log.Errorf("Register error: %v", err)
		return nil, err
	}

	log.Infof("Register response: %v", registerClusterResponsePtr.String())

	return registerClusterResponsePtr, nil
}

// UpdateClusterStatus client method sends UpdateClusterStatusRequest message to the server. It receives UpdateClusterStatusResponse message
// The message contain enum value - with the stage status of the cluster agent.
// In case of an error the function will return the error.
func (cli *Client) UpdateClusterStatus(ctx context.Context, state string, nodeAuthToken string) (*proto.UpdateClusterStatusResponse, error) {
	updateClusterStatusRequest := proto.UpdateClusterStatusRequest{Code: convertStatus(state), NodeGuid: nodeAuthToken}

	logPrefix := fmt.Sprintf("Cluster Agent state update: state=%v guid=%v", state, nodeAuthToken)

	updateClusterStatusResponsePtr, err := cli.CoSouthboundClient.UpdateClusterStatus(ctx, &updateClusterStatusRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unknown:
			log.Errorf("%v Edge Cluster Manager error: %v", logPrefix, err)

		default:
			log.Errorf("%v error: %v", logPrefix, err)
		}

		return nil, err
	}

	log.Debugf("%v Edge Cluster Manager response: %s", logPrefix, updateClusterStatusResponsePtr.ActionRequest)

	return updateClusterStatusResponsePtr, nil
}

// registerToClusterOrch client method uses comms API's to register to the Cluster Orchestrator cluster
// The function will return the required data for executing the command from the registration response:
// Pointer to: Register Cluster Command.
// In case of error the function will print the error message, will sleep for some period time and will try again infnit.
func (cli *Client) registerToClusterOrch(ctx context.Context, guid string) (installCmd, uninstallCmd string) {

	var resp *proto.RegisterClusterResponse
	log.Infof("sending register cluster request for guid: %v", guid)

	// Add client information. This is used Southbound handler to bypass rbac for cluster-agent
	// NOTE: This is for testing only. Not to be done in production.
	niceMD := metautils.NiceMD{}
	niceMD.Add("client", "cluster-agent")
	newCtx := niceMD.ToOutgoing(ctx)

	op := func() error {
		res, err := cli.Register(newCtx, guid)
		resp = res
		if err == nil && res != nil {
			return nil
		}
		return err
	}

	err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), newCtx))
	if err != nil {
		log.Infoln("Registering to Edge Cluster Manager has been canceled")
		return
	}

	return resp.InstallCmd.GetCommand(), resp.UninstallCmd.GetCommand()
}

// ConnectToClusterOrch function uses comms API's and the Cluster Orchestrator cluster address to connect with GRPC the Cluster Orchestrator server.
// The function will return the Client struct
// In case of error the function will print the error message, will sleep for some period time and will try again infnit.
func ConnectToClusterOrch(serverAddr string, tlsConfig *tls.Config) (*Client, error) {
	clusterOrch := NewClient(serverAddr, tlsConfig)

	err := clusterOrch.Connect()
	if err != nil {
		return nil, err
	}

	return clusterOrch, nil
}

func convertStatus(s string) proto.UpdateClusterStatusRequest_Code {
	m := map[string]proto.UpdateClusterStatusRequest_Code{
		"INACTIVE":              proto.UpdateClusterStatusRequest_INACTIVE,
		"REGISTERING":           proto.UpdateClusterStatusRequest_REGISTERING,
		"INSTALL_IN_PROGRESS":   proto.UpdateClusterStatusRequest_INSTALL_IN_PROGRESS,
		"ACTIVE":                proto.UpdateClusterStatusRequest_ACTIVE,
		"DEREGISTERING":         proto.UpdateClusterStatusRequest_DEREGISTERING,
		"UNINSTALL_IN_PROGRESS": proto.UpdateClusterStatusRequest_UNINSTALL_IN_PROGRESS,
		"ERROR":                 proto.UpdateClusterStatusRequest_ERROR,
	}
	return m[s]
}
