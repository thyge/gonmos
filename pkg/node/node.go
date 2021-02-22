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
	Receivers               []nmos.NMOSReceivers
	RegistryURI             string
	RegisterHBURI           string
	DeleteURI               string
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
	a.RegistryURI = fmt.Sprintf("http://%s/x-nmos/registration/%s/resource", regAddress, apiVersion)
	a.RegisterHBURI = fmt.Sprintf("http://%s/x-nmos/registration/%s/health/nodes/%s", regAddress, apiVersion, a.Node.Id)
	a.DeleteURI = fmt.Sprintf("%s/nodes/%s", a.RegistryURI, a.Node.Id)

	wrapped := nmos.MakeTransmission(a.Node, "node")
	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(wrapped)

	resp, _ := http.Post(a.RegistryURI, "application/json", payloadBuf)
	if resp.StatusCode == 201 {
		fmt.Println("Registring with registry: ", a.RegistryURI)
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

	a.SendSenders()
}

func (a *NMOSNode) RemoveFromRegistry() {
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", a.DeleteURI, nil)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
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
		log.Fatal(resp, string(prettyJSON.Bytes()))
	}
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

func (a *NMOSNode) InitTestSendersAndRecievers() {

}

func (a *NMOSNode) SendDevice() {
	a.Device = nmos.NMOSDevice{
		Id:          uuid.New(),
		Version:     "1441704616:890020555",
		Description: "test",
		Label:       "Test",
		Type:        "urn:x-nmos:device:pipeline",
		Tags:        nmos.NMOSTags{},
	}
	a.Device.Senders = make([]nmos.NMOSSender, 0)
	a.Device.Receivers = make([]nmos.NMOSReceivers, 0)
	a.Device.Controls = append(a.Device.Controls, nmos.NMOSControl{
		Type: "urn:x-manufacturer:control:generic",
		Href: "wss://154.67.63.2:4535",
	})
	a.Device.Node_id = a.Node.Id
	a.Device.Senders = append(a.Senders, nmos.NMOSSender{
		Id:            uuid.New(),
		Version:       "1441704616:890020555",
		Description:   "Test Card",
		Label:         "Test Card",
		Tags:          nmos.NMOSTags{},
		Manifest_href: "http://172.29.80.65/x-manufacturer/senders/d7aa5a30-681d-4e72-92fb-f0ba0f6f4c3e/stream.sdp",
		Flow_id:       uuid.New(),
		Transport:     "urn:x-nmos:transport:rtp.mcast",
		Device_id:     a.Device.Id,
	})

	wrapped := nmos.MakeTransmission(a.Device, "device")
	payloadBuf := new(bytes.Buffer)
	enc := json.NewEncoder(payloadBuf)
	enc.SetIndent("", "    ")
	enc.Encode(wrapped)

	fmt.Println(payloadBuf)
	if a.RegistryURI == "" {
		time.Sleep(time.Second * 2)
	}
	resp, err := http.Post(a.RegistryURI, "application/json", payloadBuf)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode == 201 {
		log.Println("Sending Sender to:", a.RegistryURI)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, body, "", "\t")
		if error != nil {
			log.Fatal("JSON parse error: ", error)
		}
		log.Fatal(resp, string(prettyJSON.Bytes()))
	}
}

func (a *NMOSNode) SendSenders() {
	for _, sender := range a.Senders {
		wrapped := nmos.MakeTransmission(sender, "sender")
		payloadBuf := new(bytes.Buffer)
		enc := json.NewEncoder(payloadBuf)
		enc.SetIndent("", "    ")
		enc.Encode(wrapped)

		fmt.Println(payloadBuf)
		resp, _ := http.Post(a.RegistryURI, "application/json", payloadBuf)
		if resp.StatusCode == 201 {
			log.Println("Sending Sender to:", a.RegistryURI)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			var prettyJSON bytes.Buffer
			error := json.Indent(&prettyJSON, body, "", "\t")
			if error != nil {
				log.Fatal("JSON parse error: ", error)
			}
			log.Fatal(resp, string(prettyJSON.Bytes()))
		}
	}
}

func (a *NMOSNode) Start(ctx context.Context, port int) {
	a.Ctx, a.CancelHeartBeat = context.WithCancel(ctx)
	a.Node.Init(port)
	// add senders

	// a.InitTestSendersAndRecievers()
	// start api and mdns
	a.WSApi.Start(port)
	a.WSApi.InitNode(&a.Node)
	// brows for registry
	a.StartRegistryDiscovery()
	// await external cancel, then cleanup
	a.SendDevice()
	a.InitTestSendersAndRecievers()
	<-ctx.Done()
	// cleanup
	a.RemoveFromRegistry()
	a.WSApi.Stop()
	log.Println("Stopping heartbeat")
	a.CancelHeartBeat()
	log.Println("stopping registry discovery")
	a.CancelRegistryDiscovery()
}
