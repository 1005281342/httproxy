package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/1005281342/httproxy/grpchttp"
	"github.com/1005281342/httproxy/grpchttp/example/hello/hello"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/config"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/server"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/svc"
	"github.com/fullstorydev/grpchan"
	"github.com/fullstorydev/grpchan/httpgrpc"
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

	reg := grpchan.HandlerMap{}
	grpchttp.RegisterHandler(reg, srv, &hello.ServiceDesc)

	var mux http.ServeMux
	httpgrpc.HandleServices(mux.HandleFunc, "/", reg, nil, nil)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.HttpPort))
	if err != nil {
		panic(err)
	}
	logx.Infof("http port: %s", lis.Addr().String())

	httpServer := http.Server{Handler: &mux}
	go httpServer.Serve(lis)
	defer httpServer.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		hello.RegisterHelloServer(grpcServer, srv)

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	var lo = fmt.Sprintf("0.0.0.0:%d", c.HttpPort)
	// 注册服务
	err = polaris.RegitserService(polaris.NewPolarisConfig(lo,
		polaris.WithServiceName(c.Etcd.Key),
		polaris.WithNamespace("default"),
		polaris.WithHeartbeatInervalSec(5)))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
