package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/thyge/gonmos/pkg/nmos"
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

	// example config
	d := &nmos.NMOSDevice{
		Id:          uuid.New(),
		Version:     fmt.Sprintf("%s:0", strconv.FormatInt(time.Now().Unix(), 10)),
		Description: "test",
		Label:       "Test",
		Type:        "urn:x-nmos:device:generic",
		Tags:        nmos.NMOSTags{},
	}
	d.Controls = append(d.Controls, nmos.NMOSControl{
		Type: "urn:x-nmos:control:sr-ctrl/v1.1",
		Href: "",
	})

	d.Senders = append(d.Senders, nmos.NMOSSender{
		Id:                 uuid.New(),
		Version:            fmt.Sprintf("%s:0", strconv.FormatInt(time.Now().Unix(), 10)),
		Description:        "Test Card",
		Label:              "Test Card",
		Tags:               nmos.NMOSTags{},
		Manifest_href:      "",
		Flow_id:            uuid.New(),
		Transport:          "urn:x-nmos:transport:rtp.mcast",
		Device_id:          d.Id,
		Interface_bindings: make([]string, 0),
		Subscription: nmos.NMOSSubscription{
			Receiver_id: uuid.New(),
		},
	})

	// Start node
	port := 8889
	app.Start(ctx, port, d)
}
