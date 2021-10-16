package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/thyge/gonmos/pkg/nmos"
)

func main() {
	nmosws := new(nmos.NMOSWebServer)
	nmosws.Start(8888)
	nmosws.InitRegister()
	defer nmosws.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		nmosws.Stop()
	}()

	forever := make(chan int)
	<-forever
}
