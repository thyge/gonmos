package main

import (
	"github.com/thyge/gonmos/pkg/nmos"
	"github.com/thyge/gonmos/pkg/node"
)

func main() {
	nmosws := new(nmos.NMOSWebServer)
	nmosws.Start(8888)
	nmosws.InitRegister()
	defer nmosws.Stop()

	go node.GracefulShutdown()
	forever := make(chan int)
	<-forever
}
