package nmos

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/google/uuid"
)

type NMOSNodeData struct {
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

func GetPreferredNetworkAdapters() []net.Interface {
	ifaces, _ := net.Interfaces()
	var retFaces []net.Interface
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			// process IP address
			// up|broadcast|multicast
			if i.Flags != net.FlagBroadcast|net.FlagUp|net.FlagMulticast {
				continue
			}
			ipv4Addr := addr.(*net.IPNet).IP.To4()
			if ipv4Addr == nil {
				continue
			}
			retFaces = append(retFaces, i)
			// fmt.Println(i, addr)
		}
	}
	return retFaces
}

func (n *NMOSNodeData) Init(port int) {

	myIPAddresses := GetPreferredNetworkAdapters()
	hostName, _ := os.Hostname()
	splitHostName := strings.Split(hostName, ".")
	n.Description = fmt.Sprintf("%s-node", splitHostName[0])
	n.Version = "1441973902:879053935"
	n.Hostname = hostName
	n.Label = splitHostName[0]
	n.Id = uuid.New()

	for _, intf := range myIPAddresses {
		addr, _ := intf.Addrs()
		for _, add := range addr {
			if add.(*net.IPNet).IP.To4() != nil {
				if n.Href == "" {
					n.Href = fmt.Sprintf("http://%s:%d", add.(*net.IPNet).IP.To4().String(), port)
				}
				n.API.Endpoints = append(n.API.Endpoints, NMOSEndpoint{
					Host:     add.(*net.IPNet).IP.To4().String(),
					Port:     port,
					Protocol: "http",
				})
			}
		}
		localMac := strings.Replace(intf.HardwareAddr.String(), ":", "-", -1)
		n.Interfaces = append(n.Interfaces, NMOSInterface{
			Name:      intf.Name,
			ChassisID: localMac,
			PortID:    localMac,
		})
	}
	n.API.Versions = append(n.API.Versions, "v1.0")
	n.API.Versions = append(n.API.Versions, "v1.1")
	n.API.Versions = append(n.API.Versions, "v1.2")
	n.API.Versions = append(n.API.Versions, "v1.3")
	n.Services = make([]NMOSService, 0)
	n.Clocks = make([]NMOSClocks, 0)
}

type NMOSTypeHolder struct {
	Type string       `json:"type"`
	Data NMOSNodeData `json:"data"`
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
	ChassisID string `json:"chassis_id"`
	// MAC ADDRESS
	PortID string `json:"port_id"`
	// Private for now
	attNetDevice NMOSAttachedNetworkDevice `json:"attached_network_device"`
}

type NMOSAttachedNetworkDevice struct {
	// This is LLDP information
	ChassisId string `json:"chassis_id"`
	PortId    string `json:"port_id"`
}

func MakeTransmission(d interface{}, name string) interface{} {
	return struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: name,
		Data: d,
	}
}

type NMOSSubscription struct {
	Receiver_id uuid.UUID `json:"receiver_id"`
	Active      bool      `json:"active"`
}

type NMOSReceiver struct {
	description        string
	label              string
	version            string
	manifest_href      string
	flow_id            uuid.UUID
	Id                 uuid.UUID
	transport          string
	device_id          uuid.UUID
	interface_bindings []string
	tags               NMOSTags
}

type NMOSSender struct {
	Id                 uuid.UUID        `json:"id"`
	Version            string           `json:"version"`
	Description        string           `json:"description"`
	Label              string           `json:"label"`
	Tags               NMOSTags         `json:"tags"`
	Manifest_href      string           `json:"manifest_href"`
	Flow_id            uuid.UUID        `json:"flow_id"`
	Transport          string           `json:"transport"`
	Device_id          uuid.UUID        `json:"device_id"`
	caps               NMOSCapabilities `json:"caps"`
	Interface_bindings []string         `json:"interface_bindings"`
	Subscription       NMOSSubscription `json:"subscription"`
}

func (ns *NMOSSender) InitHREF(ip string) {
	ns.Manifest_href = fmt.Sprintf("http://%s/x-manufacturer/senders/%s/stream.sdp", ip, ns.Id)
}

type NMOSControl struct {
	Type string `json:"type"`
	Href string `json:"href"`
}

type NMOSDevice struct {
	Id          uuid.UUID      `json:"id"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Label       string         `json:"label"`
	Tags        NMOSTags       `json:"tags"`
	Type        string         `json:"type"`
	Node_id     uuid.UUID      `json:"node_id"`
	Senders     []NMOSSender   `json:"senders"`
	Receivers   []NMOSReceiver `json:"receivers"`
	Controls    []NMOSControl  `json:"controls"`
}

func (d NMOSDevice) MarshalJSON() ([]byte, error) {
	// stdMarshal, err :=
	nd := new(pNMOSDevice)
	nd.Id = d.Id
	nd.Version = d.Version
	nd.Description = d.Description
	nd.Label = d.Label
	nd.Tags = d.Tags
	nd.Type = d.Type
	nd.Node_id = d.Node_id
	// Replace sender with uuid array
	nd.Senders = make([]uuid.UUID, 0)
	for _, s := range d.Senders {
		nd.Senders = append(nd.Senders, s.Id)
	}
	nd.Receivers = make([]uuid.UUID, 0)
	for _, r := range d.Receivers {
		nd.Receivers = append(nd.Receivers, r.Id)
	}
	nd.Controls = d.Controls
	return json.Marshal(nd)
}

type pNMOSDevice struct {
	Id          uuid.UUID     `json:"id"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Label       string        `json:"label"`
	Tags        NMOSTags      `json:"tags"`
	Type        string        `json:"type"`
	Node_id     uuid.UUID     `json:"node_id"`
	Senders     []uuid.UUID   `json:"senders"`
	Receivers   []uuid.UUID   `json:"receivers"`
	Controls    []NMOSControl `json:"controls"`
}
