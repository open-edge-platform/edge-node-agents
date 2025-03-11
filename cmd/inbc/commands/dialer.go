/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"context"
	"fmt"
	"net"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial returns a new grpc client
 func Dial(ctx context.Context, addr string) (pb.InbServiceClient, *grpc.ClientConn,error) {

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		// cut off the unix:// part
		addr = addr[7:]
		return net.Dial("unix", addr)
	}
	
	conn, err := grpc.NewClient("unix:///tmp/inbd.sock", 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}

	return pb.NewInbServiceClient(conn), conn, nil
}