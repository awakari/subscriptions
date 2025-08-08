package queue

import (
	"context"
	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type clientPool struct {
	connPool *grpcpool.Pool
}

func NewClientPool(connPool *grpcpool.Pool) ServiceClient {
	return clientPool{
		connPool: connPool,
	}
}

func (cp clientPool) SetQueue(ctx context.Context, req *SetQueueRequest, opts ...grpc.CallOption) (resp *emptypb.Empty, err error) {
	var conn *grpcpool.ClientConn
	conn, err = cp.connPool.Get(ctx)
	if err == nil {
		defer conn.Close()
	}
	var client ServiceClient
	if err == nil {
		client = NewServiceClient(conn)
		resp, err = client.SetQueue(ctx, req, opts...)
	}
	return
}

func (cp clientPool) ReceiveMessages(ctx context.Context, opts ...grpc.CallOption) (stream Service_ReceiveMessagesClient, err error) {
	var conn *grpcpool.ClientConn
	conn, err = cp.connPool.Get(ctx)
	var c *grpc.ClientConn
	if err == nil {
		c = conn.ClientConn
		conn.Close()
	}
	var client ServiceClient
	if err == nil {
		client = NewServiceClient(c)
		stream, err = client.ReceiveMessages(ctx, opts...)
	}
	return
}

func (cp clientPool) GetLength(ctx context.Context, req *GetLengthRequest, opts ...grpc.CallOption) (resp *GetLengthResponse, err error) {
	var conn *grpcpool.ClientConn
	conn, err = cp.connPool.Get(ctx)
	if err == nil {
		defer conn.Close()
	}
	var client ServiceClient
	if err == nil {
		client = NewServiceClient(conn)
		resp, err = client.GetLength(ctx, req, opts...)
	}
	return
}
