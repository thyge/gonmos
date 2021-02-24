package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/thyge/gonmos/pkg/nmos"
)

func GetResource(uri string) {
	resp, err := http.Get(uri)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		nodes := []nmos.NMOSNodeData{}
		json.Unmarshal(body, &nodes)
		for _, n := range nodes {
			log.Println(n.Hostname, n.Href)
		}
	} else {
		log.Println("failed get on:", uri)
	}
}

func GetNodesFromReg(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		// fmt.Println("Found registry service:", entry.AddrIPv4, entry.Domain, entry.Port, entry.Text)
		apiVersion := "v1.3"
		regAddress := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
		queryuri := fmt.Sprintf("http://%s/x-nmos/query/%s/nodes", regAddress, apiVersion)

		GetResource(queryuri)
	}
}

func main() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}
	entries := make(chan *zeroconf.ServiceEntry)
	go GetNodesFromReg(entries)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = resolver.Browse(ctx, "_nmos-register._tcp", "local", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	<-ctx.Done()
	time.Sleep(1 * time.Second)
}
