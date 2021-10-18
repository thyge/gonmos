package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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
		Version:     "1441704616:890020555",
		Description: "test",
		Label:       "Test",
		Type:        "urn:x-nmos:device:pipeline",
		Tags:        nmos.NMOSTags{},
	}
	d.Controls = append(d.Controls, nmos.NMOSControl{
		Type: "urn:x-manufacturer:control:generic",
		Href: "wss://154.67.63.2:4535",
	})

	d.Senders = append(d.Senders, nmos.NMOSSender{
		Id:                 uuid.New(),
		Version:            "1:1",
		Description:        "Test Card",
		Label:              "Test Card",
		Tags:               nmos.NMOSTags{},
		Manifest_href:      "http://172.29.80.65/x-manufacturer/senders/d7aa5a30-681d-4e72-92fb-f0ba0f6f4c3e/stream.sdp",
		Flow_id:            uuid.New(),
		Transport:          "urn:x-nmos:transport:rtp.mcast",
		Device_id:          d.Id,
		Interface_bindings: make([]string, 0),
		Subscription: nmos.NMOSSubscription{
			Receiver_id: uuid.New(),
		},
	})
	d.Senders[0].Interface_bindings = append(d.Senders[0].Interface_bindings, "eth")

	// Start node
	port := 8889
	app.Start(ctx, port, d)
}
