package logic

import (
	"context"
	"errors"

	"github.com/1005281342/httproxy/grpchttp/example/hello/hello"
	"github.com/1005281342/httproxy/grpchttp/example/hello/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SayLogic {
	return &SayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SayLogic) Say(in *hello.SayReq) (*hello.SayRsp, error) {

	if in.Name == "sb" {
		return nil, errors.New("name illegal")
	}

	return &hello.SayRsp{Reply: "hello, " + in.Name}, nil
}
