package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/thyge/gonmos/pkg/nmos"
)

type App struct {
	Registers               []zeroconf.ServiceEntry
	Node                    nmos.NMOSNode
	RegisterHBURI           string
	CancelHeartBeat         context.CancelFunc
	CancelRegistryDiscovery context.CancelFunc
	Ctx                     context.Context
}

func (a *App) ProcessEntries(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		a.Registers = append(a.Registers, *entry)
		fmt.Println("Found registry service:", entry.AddrIPv4, entry.Domain, entry.Port, entry.Text)
		a.AddNodeToReg(*entry)
	}
}

func (a *App) StartRegistryDiscovery() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}
	entries := make(chan *zeroconf.ServiceEntry)
	go a.ProcessEntries(entries)

	var ctx context.Context
	ctx, a.CancelRegistryDiscovery = context.WithCancel(a.Ctx)
	err = resolver.Browse(ctx, "_nmos-register._tcp", "local", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
}

func (a *App) AddNodeToReg(reg zeroconf.ServiceEntry) {
	apiVersion := "v1.3"
	regAddress := fmt.Sprintf("%s:%d", reg.AddrIPv4[0], reg.Port)
	regUri := fmt.Sprintf("http://%s/x-nmos/registration/%s/resource", regAddress, apiVersion)
	a.RegisterHBURI = fmt.Sprintf("http://%s/x-nmos/registration/%s/health/nodes/%s", regAddress, apiVersion, a.Node.Id)

	wrapped := nmos.MakeTransmission(a.Node)
	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(wrapped)

	resp, _ := http.Post(regUri, "application/json", payloadBuf)
	if resp.StatusCode == 201 {
		fmt.Println("Registring with registry: ", regUri)
	}
	// body, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(body))
	go RegisterHeartBeat(a.Ctx, a.RegisterHBURI)
}

func RegisterHeartBeat(ctx context.Context, uri string) {
	for {
		select {
		case <-ctx.Done(): // if cancel() execute
			fmt.Println("Stopping heartbeat")
			return
		default:
			fmt.Println("Heartbeat running")
			http.Post(uri, "application/json", nil)
			time.Sleep(5 * time.Second)
		}
	}
}

func main() {

	flag.Parse()

	app := new(App)
	app.Ctx, app.CancelHeartBeat = context.WithCancel(context.Background())

	app.Node.Init()
	app.StartRegistryDiscovery()

	// Wait some additional time to see debug messages on go routine shutdown.
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		app.CancelHeartBeat()
		app.CancelRegistryDiscovery()
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
