package main

type NMOSNode struct {
	Version     string          `json:"version"`
	Hostname    string          `json:"hostname"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Tags        []string        `json:"tags"`
	Href        string          `json:"href"`
	API         NMOSAPI         `json:"api"`
	Services    []NMOSServices  `json:"services"`
	Caps        []string        `json:"caps"`
	Id          string          `json:"id"`
	Clocks      []NMOSClocks    `json:"clocks"`
	Interfaces  []NMOSInterface `json:"interfaces"`
}

type NMOSAPI struct {
	Versions  []string       `json:"version"`
	Endpoints []NMOSEndpoint `json:"endpoints"`
}

type NMOSEndpoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type NMOSServices struct {
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
