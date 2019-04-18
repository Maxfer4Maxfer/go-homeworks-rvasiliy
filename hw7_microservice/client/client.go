package main

import (
	"context"
	"fmt"
	"log"

	"../protobuf"

	"google.golang.org/grpc"
)

func main() {

	// create a connection to a grpc server
	grpcConn, err := grpc.Dial(
		"127.0.0.1:8082",
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("cant connect to grpc")
	}
	defer grpcConn.Close()

	// connect to BizServer
	biz := protobuf.NewBizClient(grpcConn)

	ctx := context.Background()

	n, err := biz.Check(ctx, &protobuf.Nothing{})
	fmt.Println(n, err)

	// connect to AdminServer
	admin := protobuf.NewAdminClient(grpcConn)

	ctxAdmin := context.Background()
	client, err := admin.Logging(ctxAdmin, &protobuf.Nothing{})
	fmt.Println(client, err)

}
