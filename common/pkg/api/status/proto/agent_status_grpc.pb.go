// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: status/proto/agent_status.proto

package agent_status

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// StatusServiceClient is the client API for StatusService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StatusServiceClient interface {
	ReportStatus(ctx context.Context, in *ReportStatusRequest, opts ...grpc.CallOption) (*ReportStatusResponse, error)
	GetStatusInterval(ctx context.Context, in *GetStatusIntervalRequest, opts ...grpc.CallOption) (*GetStatusIntervalResponse, error)
}

type statusServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStatusServiceClient(cc grpc.ClientConnInterface) StatusServiceClient {
	return &statusServiceClient{cc}
}

func (c *statusServiceClient) ReportStatus(ctx context.Context, in *ReportStatusRequest, opts ...grpc.CallOption) (*ReportStatusResponse, error) {
	out := new(ReportStatusResponse)
	err := c.cc.Invoke(ctx, "/agent_status_proto.v1.StatusService/ReportStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statusServiceClient) GetStatusInterval(ctx context.Context, in *GetStatusIntervalRequest, opts ...grpc.CallOption) (*GetStatusIntervalResponse, error) {
	out := new(GetStatusIntervalResponse)
	err := c.cc.Invoke(ctx, "/agent_status_proto.v1.StatusService/GetStatusInterval", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StatusServiceServer is the server API for StatusService service.
// All implementations must embed UnimplementedStatusServiceServer
// for forward compatibility
type StatusServiceServer interface {
	ReportStatus(context.Context, *ReportStatusRequest) (*ReportStatusResponse, error)
	GetStatusInterval(context.Context, *GetStatusIntervalRequest) (*GetStatusIntervalResponse, error)
	mustEmbedUnimplementedStatusServiceServer()
}

// UnimplementedStatusServiceServer must be embedded to have forward compatible implementations.
type UnimplementedStatusServiceServer struct {
}

func (UnimplementedStatusServiceServer) ReportStatus(context.Context, *ReportStatusRequest) (*ReportStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportStatus not implemented")
}
func (UnimplementedStatusServiceServer) GetStatusInterval(context.Context, *GetStatusIntervalRequest) (*GetStatusIntervalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatusInterval not implemented")
}
func (UnimplementedStatusServiceServer) mustEmbedUnimplementedStatusServiceServer() {}

// UnsafeStatusServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StatusServiceServer will
// result in compilation errors.
type UnsafeStatusServiceServer interface {
	mustEmbedUnimplementedStatusServiceServer()
}

func RegisterStatusServiceServer(s grpc.ServiceRegistrar, srv StatusServiceServer) {
	s.RegisterService(&StatusService_ServiceDesc, srv)
}

func _StatusService_ReportStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusServiceServer).ReportStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agent_status_proto.v1.StatusService/ReportStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusServiceServer).ReportStatus(ctx, req.(*ReportStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StatusService_GetStatusInterval_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStatusIntervalRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusServiceServer).GetStatusInterval(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agent_status_proto.v1.StatusService/GetStatusInterval",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusServiceServer).GetStatusInterval(ctx, req.(*GetStatusIntervalRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// StatusService_ServiceDesc is the grpc.ServiceDesc for StatusService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StatusService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "agent_status_proto.v1.StatusService",
	HandlerType: (*StatusServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ReportStatus",
			Handler:    _StatusService_ReportStatus_Handler,
		},
		{
			MethodName: "GetStatusInterval",
			Handler:    _StatusService_GetStatusInterval_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "status/proto/agent_status.proto",
}
