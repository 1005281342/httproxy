// Code generated by goctl. DO NOT EDIT!
// Source: hello.proto

package helloclient

import (
	"context"

	"github.com/1005281342/httproxy/grpchttp/example/hello/hello"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	SayReq = hello.SayReq
	SayRsp = hello.SayRsp

	Hello interface {
		Say(ctx context.Context, in *SayReq, opts ...grpc.CallOption) (*SayRsp, error)
	}

	defaultHello struct {
		cli zrpc.Client
	}
)

func NewHello(cli zrpc.Client) Hello {
	return &defaultHello{
		cli: cli,
	}
}

func (m *defaultHello) Say(ctx context.Context, in *SayReq, opts ...grpc.CallOption) (*SayRsp, error) {
	client := hello.NewHelloClient(m.cli.Conn())
	return client.Say(ctx, in, opts...)
}
