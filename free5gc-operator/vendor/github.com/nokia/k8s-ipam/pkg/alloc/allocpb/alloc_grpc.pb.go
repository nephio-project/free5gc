// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: pkg/alloc/allocpb/alloc.proto

package allocpb

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

// AllocationClient is the client API for Allocation service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AllocationClient interface {
	Get(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	Allocate(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	DeAllocate(ctx context.Context, in *Request, opts ...grpc.CallOption) (*EmptyResponse, error)
	WatchAlloc(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Allocation_WatchAllocClient, error)
}

type allocationClient struct {
	cc grpc.ClientConnInterface
}

func NewAllocationClient(cc grpc.ClientConnInterface) AllocationClient {
	return &allocationClient{cc}
}

func (c *allocationClient) Get(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/alloc.Allocation/Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *allocationClient) Allocate(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/alloc.Allocation/Allocate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *allocationClient) DeAllocate(ctx context.Context, in *Request, opts ...grpc.CallOption) (*EmptyResponse, error) {
	out := new(EmptyResponse)
	err := c.cc.Invoke(ctx, "/alloc.Allocation/DeAllocate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *allocationClient) WatchAlloc(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Allocation_WatchAllocClient, error) {
	stream, err := c.cc.NewStream(ctx, &Allocation_ServiceDesc.Streams[0], "/alloc.Allocation/WatchAlloc", opts...)
	if err != nil {
		return nil, err
	}
	x := &allocationWatchAllocClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Allocation_WatchAllocClient interface {
	Recv() (*WatchResponse, error)
	grpc.ClientStream
}

type allocationWatchAllocClient struct {
	grpc.ClientStream
}

func (x *allocationWatchAllocClient) Recv() (*WatchResponse, error) {
	m := new(WatchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AllocationServer is the server API for Allocation service.
// All implementations must embed UnimplementedAllocationServer
// for forward compatibility
type AllocationServer interface {
	Get(context.Context, *Request) (*Response, error)
	Allocate(context.Context, *Request) (*Response, error)
	DeAllocate(context.Context, *Request) (*EmptyResponse, error)
	WatchAlloc(*WatchRequest, Allocation_WatchAllocServer) error
	mustEmbedUnimplementedAllocationServer()
}

// UnimplementedAllocationServer must be embedded to have forward compatible implementations.
type UnimplementedAllocationServer struct {
}

func (UnimplementedAllocationServer) Get(context.Context, *Request) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedAllocationServer) Allocate(context.Context, *Request) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Allocate not implemented")
}
func (UnimplementedAllocationServer) DeAllocate(context.Context, *Request) (*EmptyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeAllocate not implemented")
}
func (UnimplementedAllocationServer) WatchAlloc(*WatchRequest, Allocation_WatchAllocServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchAlloc not implemented")
}
func (UnimplementedAllocationServer) mustEmbedUnimplementedAllocationServer() {}

// UnsafeAllocationServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AllocationServer will
// result in compilation errors.
type UnsafeAllocationServer interface {
	mustEmbedUnimplementedAllocationServer()
}

func RegisterAllocationServer(s grpc.ServiceRegistrar, srv AllocationServer) {
	s.RegisterService(&Allocation_ServiceDesc, srv)
}

func _Allocation_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AllocationServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/alloc.Allocation/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AllocationServer).Get(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Allocation_Allocate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AllocationServer).Allocate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/alloc.Allocation/Allocate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AllocationServer).Allocate(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Allocation_DeAllocate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AllocationServer).DeAllocate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/alloc.Allocation/DeAllocate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AllocationServer).DeAllocate(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Allocation_WatchAlloc_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AllocationServer).WatchAlloc(m, &allocationWatchAllocServer{stream})
}

type Allocation_WatchAllocServer interface {
	Send(*WatchResponse) error
	grpc.ServerStream
}

type allocationWatchAllocServer struct {
	grpc.ServerStream
}

func (x *allocationWatchAllocServer) Send(m *WatchResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Allocation_ServiceDesc is the grpc.ServiceDesc for Allocation service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Allocation_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "alloc.Allocation",
	HandlerType: (*AllocationServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Get",
			Handler:    _Allocation_Get_Handler,
		},
		{
			MethodName: "Allocate",
			Handler:    _Allocation_Allocate_Handler,
		},
		{
			MethodName: "DeAllocate",
			Handler:    _Allocation_DeAllocate_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WatchAlloc",
			Handler:       _Allocation_WatchAlloc_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pkg/alloc/allocpb/alloc.proto",
}
