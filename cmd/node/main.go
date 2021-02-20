package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/thyge/gonmos/pkg/node"
)

func main() {

	app := new(node.NMOSNode)
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	port := 8889
	app.Start(ctx, port)
}
