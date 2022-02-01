module github.com/1005281342/httproxy/grpchttp/example

go 1.16

replace (
	github.com/1005281342/httproxy v0.0.0 =>  ../../../httproxy
)

require (
    github.com/1005281342/httproxy v0.0.0
	github.com/fullstorydev/grpchan v1.1.0
	github.com/zeromicro/go-zero v1.3.0
	github.com/zeromicro/zero-contrib/zrpc/registry/polaris v0.0.0-20220119015825-25bad15c389d
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)