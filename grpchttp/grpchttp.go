package grpchttp

import (
	"google.golang.org/grpc"

	"github.com/fullstorydev/grpchan"
)

// RegisterHandler 注册Handler
func RegisterHandler(reg grpchan.ServiceRegistry, srv interface{}, desc *grpc.ServiceDesc) {
	reg.RegisterService(desc, srv)
}
