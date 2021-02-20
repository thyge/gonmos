package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/thyge/gonmos/pkg/nmos"
)

type NMOSNode struct {
	Registers               []zeroconf.ServiceEntry
	Node                    nmos.NMOSNodeData
	RegisterHBURI           string
	CancelHeartBeat         context.CancelFunc
	CancelRegistryDiscovery context.CancelFunc
	Ctx                     context.Context
	WSApi                   nmos.NMOSWebServer
}

func (a *NMOSNode) ProcessEntries(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		a.Registers = append(a.Registers, *entry)
		fmt.Println("Found registry service:", entry.AddrIPv4, entry.Domain, entry.Port, entry.Text)
		a.AddNodeToReg(*entry)
	}
}

func (a *NMOSNode) StartRegistryDiscovery() {
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

func (a *NMOSNode) AddNodeToReg(reg zeroconf.ServiceEntry) {
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
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, body, "", "\t")
		if error != nil {
			log.Fatal("JSON parse error: ", error)
		}
		log.Fatal(resp, string(prettyJSON.Bytes()))
	}

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

func (a *NMOSNode) Start(ctx context.Context, port int) {
	a.Ctx, a.CancelHeartBeat = context.WithCancel(ctx)
	a.Node.Init(port)
	// start api and mdns
	a.WSApi.Start(port)
	a.WSApi.InitNode()
	// brows for registry
	a.StartRegistryDiscovery()
	// await external cancel, then cleanup
	<-ctx.Done()
	a.WSApi.Stop()
	log.Println("Stopping heartbeat")
	a.CancelHeartBeat()
	log.Println("stopping registry discovery")
	a.CancelRegistryDiscovery()
}
