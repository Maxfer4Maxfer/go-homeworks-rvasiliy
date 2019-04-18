package server

import (
	"context"
	"fmt"

	"../protobuf"

	"github.com/maxfer4maxfer/goDebuger"
)

type BizServer struct {
}

func NewBizServer() *BizServer {
	return &BizServer{}
}

func (bs *BizServer) Check(ctx context.Context, n *protobuf.Nothing) (*protobuf.Nothing, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	return &protobuf.Nothing{Dummy: true}, nil
}

func (bs *BizServer) Add(ctx context.Context, n *protobuf.Nothing) (*protobuf.Nothing, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}
	return &protobuf.Nothing{Dummy: true}, nil
}

func (bs *BizServer) Test(ctx context.Context, n *protobuf.Nothing) (*protobuf.Nothing, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}
	// md, _ := metadata.FromIncomingContext(ctx)
	// fmt.Println("md = ", md)

	return &protobuf.Nothing{Dummy: true}, nil
}
