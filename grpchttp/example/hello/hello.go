package main

import (
	"flag"
	"fmt"
	"github.com/1005281342/httproxy/grpchttp"
	"github.com/1005281342/httproxy/grpchttp/example/hello/hello"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/config"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/server"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/polaris"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/hello.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	srv := server.NewHelloServer(ctx)

	var sPort = grpchttp.RegisterAndStart(srv, &hello.ServiceDesc, 0)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		hello.RegisterHelloServer(grpcServer, srv)

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	const (
		namespaceZRPC = "default"
		namespaceHTTP = "default"
	)

	var err error
	// 注册zrpc服务
	if err = polaris.RegitserService(polaris.NewPolarisConfig(c.ListenOn,
		polaris.WithServiceName(c.Etcd.Key),
		polaris.WithNamespace(namespaceZRPC),
		polaris.WithHeartbeatInervalSec(5))); err != nil {
		logx.Errorf("注册zrpc到Polaris失败")
	}

	// 注册http服务
	var lo = "0.0.0.0:" + sPort
	if err = polaris.RegitserService(polaris.NewPolarisConfig(lo,
		polaris.WithServiceName(c.Etcd.Key+"-http"),
		polaris.WithNamespace(namespaceHTTP),
		polaris.WithHeartbeatInervalSec(5),
		polaris.WithProtocol("http"))); err != nil {
		panic(err)
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
