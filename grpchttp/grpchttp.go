package grpchttp

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/fullstorydev/grpchan"
	"github.com/fullstorydev/grpchan/httpgrpc"
	"github.com/zeromicro/go-zero/rest"
	"google.golang.org/grpc"
)

// RegisterHandler 注册Handler
func RegisterHandler(reg grpc.ServiceRegistrar, srv interface{}, desc *grpc.ServiceDesc) {
	reg.RegisterService(desc, srv)
}

type mux map[string]http.HandlerFunc

func (m mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m[pattern] = handler
}

func RegisterWithGoZero(srv interface{}, desc *grpc.ServiceDesc, httpServer *rest.Server) {
	reg := grpchan.HandlerMap{}
	RegisterHandler(reg, srv, desc)

	var mux = mux{}
	httpgrpc.HandleServices(mux.HandleFunc, "/", reg, nil, nil)

	for path, handler := range mux {
		log.Printf("path: %s\n", path)
		httpServer.AddRoute(rest.Route{
			Method:  http.MethodPost,
			Path:    path,
			Handler: handler,
		})
	}
}

// RegisterAndStart 注册并启动
func RegisterAndStart(srv interface{}, desc *grpc.ServiceDesc, port int) string {
	reg := grpchan.HandlerMap{}
	RegisterHandler(reg, srv, desc)

	var mux http.ServeMux
	httpgrpc.HandleServices(mux.HandleFunc, "/", reg, nil, nil)
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		panic(err)
	}
	var a = strings.Split(lis.Addr().String(), ":")
	var sPort = a[len(a)-1]
	log.Printf("http port: %s\n", sPort)

	httpServer := http.Server{Handler: &mux}
	go httpServer.Serve(lis)
	return sPort
}
