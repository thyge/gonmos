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

func (n *NMOSWebServer) handleNodeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	version := vars["version"]
	rpath := vars["resourcePath"]
	if version == "" {
		json.NewEncoder(w).Encode([]string{"v1.0/", "v1.1/", "v1.2/", "v1.3/"})
		return
	}
	if rpath == "" {
		json.NewEncoder(w).Encode([]string{"devices/", "flows/", "receivers/", "self/", "senders/", "sources/"})
	}
	if rpath == "self" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(n.Node)
		// fmt.Println(n.Node)
	}
	fmt.Println(version, rpath)
}

func handleRegHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling health")
}

type NMOSWebServer struct {
	Router           *mux.Router
	Port             int
	Node             *NMOSNodeData
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

func (n *NMOSWebServer) InitNode(nodeptr *NMOSNodeData) {
	n.Node = nodeptr
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
	n.Router.HandleFunc("/x-nmos", handleNMOSBase)
	nodeSubRouter := n.Router.PathPrefix("/x-nmos/node").Subrouter()
	nodeSubRouter.HandleFunc("", n.handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}", n.handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}/{resourcePath}", n.handleNodeAPI)
}

func (n *NMOSWebServer) InitQuery() {
	// MDNS
	hostName, _ := os.Hostname()
	hostName = strings.Replace(hostName, ".local", "", -1)
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
	hostName, _ := os.Hostname()
	hostName = strings.Replace(hostName, ".local", "", -1)
	txt := MdnsText(99, []string{"v1.0", "v1.1", "v1.2", "v1.3"}, "http", false)
	var err error
	n.MDNSRegistration, err = zeroconf.Register(hostName, "_nmos-registration._tcp", "local", n.Port, txt, nil)
	if err != nil {
		panic(err)
	}
	n.MDNSRegister, err = zeroconf.Register(hostName, "_nmos-registration._tcp", "local", n.Port, txt, nil)
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
