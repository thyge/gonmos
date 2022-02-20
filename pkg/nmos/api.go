package nmos

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
)

func handleRegResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	fmt.Println("post to register", version)
}

func handleGetResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	resourceType := vars["resourceType"]
	resourceId := vars["resourceId"]
	fmt.Println("post to register", version, resourceType, resourceId)
}

func handleRegBase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	fmt.Println(version)
	json.NewEncoder(w).Encode([]string{"resource/", "health/"})
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Host, r.URL.Path)
}

func handleNMOSBase(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"node/", "query/", "registration/"})
}

func handleQueryAPI(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method, r.URL)
	vars := mux.Vars(r)
	version := vars["version"]
	if version == "" {
		json.NewEncoder(w).Encode([]string{"v1.0/", "v1.1/", "v1.2/", "v1.3/"})
		return
	}
	json.NewEncoder(w).Encode([]string{"devices/", "flows/", "nodes/", "receivers/", "senders/", "sources/", "subscriptions/"})
	fmt.Println(r)
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Printf("%s", body)
}

func (n *NMOSWebServer) handleSendersAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	// version := vars["version"]
	mode := vars["mode"]
	resource := vars["resource"]
	id := vars["id"]
	fmt.Println(vars)
	if (mode == "bulk") || (mode == "single") {
		if resource == "" {
			json.NewEncoder(w).Encode([]string{"senders/", "recievers/"})
		} else {
			if id == "" {
				// Get all uuids
				var sender_uuids []uuid.UUID
				for i := 0; i < len(n.Device.Senders); i++ {
					sender_uuids = append(sender_uuids, n.Device.Senders[0].Device_id)
				}
				// Return all senders UUID
				enc := json.NewEncoder(w)
				enc.SetIndent("", "\t")
				enc.Encode(sender_uuids)
			} else {
				// Match UUID with uuid in array
				queryuuid, _ := uuid.Parse(id)
				for i := 0; i < len(n.Device.Senders); i++ {
					if queryuuid == n.Device.Senders[i].Device_id {
						enc := json.NewEncoder(w)
						enc.SetIndent("", "\t")
						enc.Encode(n.Device.Senders[i])
					}
				}
			}
		}
	} else {
		json.NewEncoder(w).Encode([]string{"bulk/", "single/"})
	}
}

func (n *NMOSWebServer) handleNodeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	version := vars["version"]
	rpath := vars["resourcePath"]

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")

	if version == "" {
		enc.Encode([]string{"v1.0/", "v1.1/", "v1.2/", "v1.3/"})
		return
	}
	switch rpath {
	case "self":
		enc.Encode(n.Node)
	case "devices":
		enc.Encode(n.Device)
	case "senders":
		enc.Encode(n.Device.Senders)
	default:
		enc.Encode([]string{"devices/", "flows/", "receivers/", "self/", "senders/", "sources/"})
	}
}

func handleRegHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling health")
}

type NMOSWebServer struct {
	Router           *mux.Router
	Port             int
	Node             *NMOSNodeData
	Device           *NMOSDevice
	srv              *http.Server
	MDNSNode         *zeroconf.Server
	MDNSQuery        *zeroconf.Server
	MDNSRegister     *zeroconf.Server
	MDNSRegistration *zeroconf.Server
}

func (n *NMOSWebServer) Start(port int) {
	n.Port = port
	n.Router = mux.NewRouter()
	n.srv = &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      n.Router, // Pass our instance of gorilla/mux in.
	}
	go func() {
		fmt.Println("Starting webserver:", n.srv.Addr)
		if err := n.srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
}

func (n *NMOSWebServer) Stop() {
	log.Println("shutting down mdns")
	if n.MDNSNode != nil {
		n.MDNSNode.Shutdown()
	}
	if n.MDNSQuery != nil {
		n.MDNSQuery.Shutdown()
	}
	if n.MDNSRegister != nil {
		n.MDNSRegister.Shutdown()
	}
	if n.MDNSRegistration != nil {
		n.MDNSRegistration.Shutdown()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	n.srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.

	n.srv.Close()
}

func (n *NMOSWebServer) InitNode(nodeptr *NMOSNodeData, deviceptr *NMOSDevice) {
	n.Node = nodeptr
	n.Device = deviceptr
	// MDNS
	hostName, _ := os.Hostname()
	hostName = strings.Replace(hostName, ".local", "", -1)
	txt := MdnsText(99, []string{"v1.0", "v1.1", "v1.2", "v1.3"}, "http", false)
	var err error
	n.MDNSNode, err = zeroconf.Register(hostName, "_nmos-node._tcp", "local.", n.Port, txt, nil)
	if err != nil {
		panic(err)
	}
	// NODE API
	n.Router.HandleFunc("/", homeHandler)
	n.Router.HandleFunc("/x-nmos", handleNMOSBase)
	// IS-04
	nodeSubRouter := n.Router.PathPrefix("/x-nmos/node").Subrouter()
	nodeSubRouter.HandleFunc("", n.handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}", n.handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}/{resourcePath}", n.handleNodeAPI)
	// IS-05
	conSubRouter := n.Router.PathPrefix("/x-nmos/connection").Subrouter()
	conSubRouter.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"v1.0/", "v1.1/", "v1.2/", "v1.3/"})
	})
	conSubRouter.HandleFunc("/{version}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"senders/", "recievers/", "sinks/"})
	})

	conSubRouter.HandleFunc("/{version}/single", n.handleSendersAPI)
	conSubRouter.HandleFunc("/{version}/single/senders", n.handleSendersAPI)
	conSubRouter.HandleFunc("/{version}/single/senders/{id}", n.handleSendersAPI)

	conSubRouter.HandleFunc("/{version}/single/recievers", n.handleSendersAPI)
	conSubRouter.HandleFunc("/{version}/single/recievers/{id}", n.handleSendersAPI)
	// IS-11
	conSubRouter.HandleFunc("/{version}/single/sinks/{sinkId}", n.handleSendersAPI)
	conSubRouter.HandleFunc("/{version}/single/sinks/{sinkId}/properties", n.handleSendersAPI)
	conSubRouter.HandleFunc("/{version}/single/sinks/{sinkId}/edid", n.handleSendersAPI)
}

func (n *NMOSWebServer) InitQuery() {
	// MDNS
	hostNameDomain, _ := os.Hostname()
	splitHostName := strings.Split(hostNameDomain, ".")
	hostName := splitHostName[0]
	txt := MdnsText(99, []string{"v1.0", "v1.1", "v1.2", "v1.3"}, "http", false)
	var err error
	n.MDNSQuery, err = zeroconf.Register(hostName, "_nmos-query._tcp", "local", n.Port, txt, nil)
	if err != nil {
		panic(err)
	}
	// QUERY API
	querySubRouter := n.Router.PathPrefix("/x-nmos/query").Subrouter()
	querySubRouter.HandleFunc("", handleQueryAPI)
	querySubRouter.HandleFunc("/{version}/", handleQueryAPI)
	querySubRouter.HandleFunc("/{version}/{resourcePath}", handleQueryAPI)
}

func (n *NMOSWebServer) InitRegister() {
	// MDNS
	hostNameDomain, _ := os.Hostname()
	splitHostName := strings.Split(hostNameDomain, ".")
	hostName := splitHostName[0]
	txt := MdnsText(99, []string{"v1.0", "v1.1", "v1.2", "v1.3"}, "http", false)
	var err error
	n.MDNSRegistration, err = zeroconf.Register(hostName, "_nmos-registration._tcp", "local", n.Port, txt, nil)
	if err != nil {
		panic(err)
	}
	n.MDNSRegister, err = zeroconf.Register(hostName, "_nmos-register._tcp", "local", n.Port, txt, nil)
	if err != nil {
		panic(err)
	}
	// REGISTRATION API
	regSubRouter := n.Router.PathPrefix("/x-nmos/registration").Subrouter()
	regSubRouter.HandleFunc("/{version}", handleRegBase)
	regSubRouter.HandleFunc("/{version}/resource", handleRegResource)
	regSubRouter.HandleFunc("/{resourceType}/{resourceId}", handleGetResource)
	regSubRouter.HandleFunc("/{version}/health/nodes/{nodeId}", handleRegHealth)
}

func MdnsText(priority int64, versions []string, protocol string, oauth_mode bool) []string {
	var returnString []string
	returnString = append(returnString, "pri="+strconv.FormatInt(priority, 10))
	returnString = append(returnString, "api_ver="+strings.Join(versions, ","))
	returnString = append(returnString, "api_proto="+protocol)
	returnString = append(returnString, "api_auth="+strconv.FormatBool(oauth_mode))
	return returnString
}
