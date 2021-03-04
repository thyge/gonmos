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

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
	"github.com/thyge/gonmos/pkg/nmos"
)

type NMOSNode struct {
	Registers               []zeroconf.ServiceEntry
	Node                    nmos.NMOSNodeData
	Device                  nmos.NMOSDevice
	Senders                 []nmos.NMOSSender
	Receivers               []nmos.NMOSReceiver
	RegistryURI             string
	RegisterHBURI           string
	DeleteURI               string
	CancelHeartBeat         context.CancelFunc
	CancelRegistryDiscovery context.CancelFunc
	Ctx                     context.Context
	WSApi                   nmos.NMOSWebServer
}

func (a *NMOSNode) ProcessEntries(results <-chan *zeroconf.ServiceEntry, regFoundChan chan string) {
	for entry := range results {
		a.Registers = append(a.Registers, *entry)
		fmt.Println("Found registry service:", entry.AddrIPv4, entry.Domain, entry.Port, entry.Text)
		a.AddNodeToReg(*entry)
		regFoundChan <- "found reg"
	}
}

func (a *NMOSNode) StartRegistryDiscovery(regFoundChan chan string) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}
	entries := make(chan *zeroconf.ServiceEntry)
	go a.ProcessEntries(entries, regFoundChan)

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
	a.RegistryURI = fmt.Sprintf("http://%s/x-nmos/registration/%s/resource", regAddress, apiVersion)
	a.RegisterHBURI = fmt.Sprintf("http://%s/x-nmos/registration/%s/health/nodes/%s", regAddress, apiVersion, a.Node.Id)
	a.DeleteURI = fmt.Sprintf("%s/nodes/%s", a.RegistryURI, a.Node.Id)

	a.SendResource(a.Node, "node")
	go RegisterHeartBeat(a.Ctx, a.RegisterHBURI)
}

func (a *NMOSNode) RemoveFromRegistry() {
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, a.DeleteURI, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.StatusCode == 204 {
		log.Println("Deleted resource from registry")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, body, "", "\t")
		if error != nil {
			log.Fatal("JSON parse error: ", error)
		}
		log.Println(resp, string(prettyJSON.Bytes()))
	}
}

func RegisterHeartBeat(ctx context.Context, uri string) {
	hbclient := &http.Client{}
	var connectioCounter int
	for {
		select {
		case <-ctx.Done(): // if cancel() execute
			log.Println("Stopping heartbeat")
			return
		default:
			req, reqerr := http.NewRequest(http.MethodPost, uri, nil)
			if reqerr != nil {
				panic(reqerr)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Close = true
			resp, err := hbclient.Do(req)
			if err != nil {
				log.Println("error", err)
				return
			}
			connectioCounter++
			if resp.StatusCode != 200 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}
				var prettyJSON bytes.Buffer
				error := json.Indent(&prettyJSON, body, "", "\t")
				if error != nil {
					log.Fatal("JSON parse error: ", error)
				}
				log.Fatal(resp, string(prettyJSON.Bytes()))
			}
			// defer resp.Body.Close() was not happening
			// force close instead to prevent memory leak
			resp.Body.Close()
			time.Sleep(5 * time.Second)
		}
	}
}

func (a *NMOSNode) SendResource(i interface{}, name string) {
	wrapped := nmos.MakeTransmission(i, name)
	payloadBuf := new(bytes.Buffer)
	enc := json.NewEncoder(payloadBuf)
	enc.SetIndent("", "\t")
	enc.Encode(wrapped)

	resp, err := http.Post(a.RegistryURI, "application/json", payloadBuf)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 201 {
		log.Println("Sent:", name)
	} else {
		enc.Encode(wrapped)
		log.Println(payloadBuf)
		body, _ := ioutil.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, body, "", "\t")
		if error != nil {
			log.Fatal("JSON parse error: ", error)
		}
		log.Fatal(resp, string(prettyJSON.Bytes()))
	}
}

func (a *NMOSNode) InitTestSendersAndRecievers() {
	a.Device = nmos.NMOSDevice{
		Id:          uuid.New(),
		Version:     "1441704616:890020555",
		Description: "test",
		Label:       "Test",
		Type:        "urn:x-nmos:device:pipeline",
		Tags:        nmos.NMOSTags{},
	}
	a.Device.Controls = append(a.Device.Controls, nmos.NMOSControl{
		Type: "urn:x-manufacturer:control:generic",
		Href: "wss://154.67.63.2:4535",
	})
	a.Device.Node_id = a.Node.Id
	a.Device.Senders = append(a.Senders, nmos.NMOSSender{
		Id:                 uuid.New(),
		Version:            "1441704616:890020555",
		Description:        "Test Card",
		Label:              "Test Card",
		Tags:               nmos.NMOSTags{},
		Manifest_href:      "http://172.29.80.65/x-manufacturer/senders/d7aa5a30-681d-4e72-92fb-f0ba0f6f4c3e/stream.sdp",
		Flow_id:            uuid.New(),
		Transport:          "urn:x-nmos:transport:rtp.mcast",
		Device_id:          a.Device.Id,
		Interface_bindings: make([]string, 0),
		Subscription: nmos.NMOSSubscription{
			Receiver_id: uuid.New(),
		},
	})
	a.Device.Senders[0].Interface_bindings = append(a.Device.Senders[0].Interface_bindings, "eth")
	a.SendResource(a.Device, "device")
	for _, sender := range a.Device.Senders {
		a.SendResource(sender, "sender")
	}
	for _, receiver := range a.Device.Receivers {
		a.SendResource(receiver, "receiver")
	}
}

func (a *NMOSNode) Start(ctx context.Context, port int) {
	a.Ctx, a.CancelHeartBeat = context.WithCancel(ctx)
	a.Node.Init(port)
	// start api and mdns
	a.WSApi.Start(port)
	a.WSApi.InitNode(&a.Node)
	// brows for registry
	regFoundChan := make(chan string)
	a.StartRegistryDiscovery(regFoundChan)
	// await registry to be discovered
	<-regFoundChan
	// a.InitTestSendersAndRecievers()
	// await external cancel, then cleanup
	<-ctx.Done()
	// cleanup
	a.RemoveFromRegistry()
	a.WSApi.Stop()
	log.Println("Stopping heartbeat")
	a.CancelHeartBeat()
	log.Println("stopping registry discovery")
	a.CancelRegistryDiscovery()
}
