package svc

import "github.com/1005281342/httproxy/grpchttp/example/hello-rk-boot/internal/config"

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
