package nmos

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

type NMOSNode struct {
	Version     string           `json:"version"`
	Hostname    string           `json:"hostname"`
	Label       string           `json:"label"`
	Description string           `json:"description"`
	Tags        NMOSTags         `json:"tags"`
	Href        string           `json:"href"`
	API         NMOSAPI          `json:"api"`
	Services    []NMOSService    `json:"services"`
	Caps        NMOSCapabilities `json:"caps"`
	Id          uuid.UUID        `json:"id"`
	Clocks      []NMOSClocks     `json:"clocks"`
	Interfaces  []NMOSInterface  `json:"interfaces"`
}

func (n *NMOSNode) Init() {
	hosnam, _ := os.Hostname()
	n.Description = fmt.Sprintf("%s-node", hosnam)
	n.Version = "1441973902:879053935"
	n.Hostname = hosnam
	n.Label = hosnam
	n.Id = uuid.New()
	n.Href = "http://192.168.170.247:8080/"
	n.API.Endpoints = append(n.API.Endpoints, NMOSEndpoint{
		Host:     "192.168.170.247",
		Port:     8080,
		Protocol: "http",
	})
	n.API.Versions = append(n.API.Versions, "v1.3")
	n.Services = make([]NMOSService, 0)
	n.Clocks = make([]NMOSClocks, 0)
	n.Interfaces = make([]NMOSInterface, 0)
}

type NMOSTypeHolder struct {
	Type string   `json:"type"`
	Data NMOSNode `json:"data"`
}

type NMOSTags struct {
}

type NMOSCapabilities struct {
}

type NMOSAPI struct {
	Versions  []string       `json:"versions"`
	Endpoints []NMOSEndpoint `json:"endpoints"`
}

type NMOSEndpoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type NMOSService struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

type NMOSClocks struct {
	Name      string `json:"name"`
	Ref_type  string `json:"ref_type"`
	Traceable bool   `json:"traceable"`
	Version   string `json:"version"`
	Gmid      string `json:"gmid"`
	Locked    bool   `json:"locked"`
}

type NMOSInterface struct {
	// NIC NAME
	Name      string `json:"name"`
	ChassisId string `json:"chassis_id"`
	// MAC ADDRESS
	PortId       string                    `json:"port_id"`
	AttNetDevice NMOSAttachedNetworkDevice `json:"attached_network_device"`
}

type NMOSAttachedNetworkDevice struct {
	// This is LLDP information
	ChassisId string `json:"chassis_id"`
	PortId    string `json:"port_id"`
}

func MakeTransmission(n NMOSNode) interface{} {
	return struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: "node",
		Data: n,
	}
}

type NMOSSubscription struct {
	receiver_id uuid.UUID
	active      bool
}

type NMOSSender struct {
	description        string
	label              string
	version            string
	manifest_href      string
	flow_id            uuid.UUID
	id                 uuid.UUID
	transport          string
	device_id          uuid.UUID
	interface_bindings []string
	caps               NMOSCapabilities
	tags               NMOSTags
	subscription       NMOSSubscription
}

func (n *NMOSSender) Init() {
	n.description = "Test Card"
	n.label = "Test Card"
	n.version = "1441704616:890020555"
	n.manifest_href = "http://172.29.80.65/x-manufacturer/senders/d7aa5a30-681d-4e72-92fb-f0ba0f6f4c3e/stream.sdp"
	n.flow_id, _ = uuid.Parse("5fbec3b1-1b0f-417d-9059-8b94a47197ed")
	n.id, _ = uuid.Parse("d7aa5a30-681d-4e72-92fb-f0ba0f6f4c3e")
	n.transport = "urn:x-nmos:transport:rtp.mcast"
	n.device_id, _ = uuid.Parse("9126cc2f-4c26-4c9b-a6cd-93c4381c9be5")
	n.interface_bindings = append(n.interface_bindings, "eth0")
}
