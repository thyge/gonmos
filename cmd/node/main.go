package main

import (
	"context"

	"github.com/thyge/gonmos/pkg/node"
)

func main() {
	app := new(node.NMOSNode)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.Start(ctx)

	go node.GracefulShutdown()
	forever := make(chan int)
	<-forever
}
