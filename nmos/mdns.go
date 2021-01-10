package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/grandcat/zeroconf"
)

var (
	service  = flag.String("service", "_nmos-register._tcp", "Set the service category to look for devices.")
	domain   = flag.String("domain", "local", "Set the search domain. For local networks, default is fine.")
	waitTime = flag.Int("wait", 10, "Duration in [s] to run discovery.")
)

type NMOSMdnsService struct {
	Resolver        *zeroconf.Resolver
	RegistryEntries []zeroconf.ServiceEntry
}

type MDNSEntry struct {
}

func (nms NMOSMdnsService) MDNSPrintCallback(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		nms.RegistryEntries = append(nms.RegistryEntries, *entry)
	}
}

func CreateMDNService() NMOSMdnsService {
	mdns := new(NMOSMdnsService)
	service := "_nmos-register._tcp"
	domain := "local"
	waitTime := 1

	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}
	mdns.Resolver = resolver

	entries := make(chan *zeroconf.ServiceEntry)
	go mdns.MDNSPrintCallback(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(waitTime))
	defer cancel()
	err = mdns.Resolver.Browse(ctx, service, domain, entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
	// Wait some additional time to see debug messages on go routine shutdown.
	time.Sleep(1 * time.Second)

	for entry := range mdns.RegistryEntries {
		print(entry)
	}

	return *mdns
}
