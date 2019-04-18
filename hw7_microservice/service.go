package main

import (
	"context"

	"./server"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	return server.StartMyMicroservice(ctx, listenAddr, ACLData)
}
