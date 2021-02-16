package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
)

var node NMOSNode

const (
	SERVICE_PORT       int    = 8235
	DNS_SD_HTTP_PORT   int    = 3232
	DNS_SD_HTTPS_PORT  int    = 443
	DNS_SD_TYPE        string = "_nmos-register._tcp"
	DNS_SD_LEGACY_TYPE string = "_nmos-registration._tcp"
	REGISTRY_PORT      int    = 2379

	priority    int64  = 99
	https_mode  string = "disabled"
	enable_mdns bool   = true
	oauth_mode  bool   = false
)

var (
	name     = flag.String("name", "GoZeroconfGo", "The name for the service.")
	service  = flag.String("service", "_nmos-register._tcp", "Set the service type of the new service.")
	domain   = flag.String("domain", "local.", "Set the network domain. Default should be fine.")
	port     = flag.Int("port", 42424, "Set the port the service is listening to.")
	waitTime = flag.Int("wait", 0, "Duration in [s] to publish service for.")
)

func GetMainInterface(s string) []net.Interface {
	var selectedIF []net.Interface
	ifaces, _ := net.Interfaces()
	for _, ifi := range ifaces {
		if ifi.Name == s {
			addrs, _ := ifi.Addrs()
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				fmt.Println(ip)
			}
			selectedIF = append(selectedIF, ifi)
		}
	}
	return selectedIF
}

type APPContainer struct {
	Node   NMOSNode
	Router *mux.Router
	srv    *http.Server
}

var a APPContainer

func main() {
	muuid, _ := uuid.NewRandom()
	a.Node.Id = muuid.String()
	a.Node.Hostname, _ = os.Hostname()
	a.Node.Description = "gonmos example"
	a.Node.Href = "http://127.0.0.1:3232"

	hname, err := os.Hostname()
	DNS_SD_NAME := "registration_" + hname + "_https"
	AGGREGATOR_APIVERSIONS := []string{"v1.0", "v1.1", "v1.2", "v1.3"}
	flag.Parse()

	// Select 1 rather than all IFs by default
	selectedIF := GetMainInterface("Wi-Fi")

	mdnstxt := mdnsText(priority, AGGREGATOR_APIVERSIONS, "http", oauth_mode)
	mdnsRegistry, err := zeroconf.Register(DNS_SD_NAME, DNS_SD_TYPE, *domain, DNS_SD_HTTP_PORT, mdnstxt, selectedIF)
	// queryApi, err := zeroconf.Register(DNS_SD_NAME, "_nmos-query._tcp", *domain, DNS_SD_HTTP_PORT, mdnstxt, nil)
	mdnsNode, err := zeroconf.Register(DNS_SD_NAME, "_nmos-node._tcp", *domain, DNS_SD_HTTP_PORT, mdnstxt, selectedIF)
	if err != nil {
		panic(err)
	}
	defer mdnsRegistry.Shutdown()
	defer mdnsNode.Shutdown()
	// defer queryApi.Shutdown()

	handleRequests()
	AddNodeToRegistry()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	a.srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}

func AddNodeToRegistry() {
	// Check for registry here:
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Println(entry)
		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(*waitTime))
	defer cancel()
	resolver.Browse(ctx, DNS_SD_TYPE, "local", entries)

	apiVersion := "v1.3"
	uri := fmt.Sprintf("localhost:3232/x-nmos/registration/%s/resource", apiVersion)
	// POST NODE to registry
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(a.Node)
	http.Post(uri, "application/json; charset=utf-8", b)
	// io.Copy(os.Stdout, res.Body)
}

func mdnsText(priority int64, versions []string, protocol string, oauth_mode bool) []string {
	var returnString []string
	returnString = append(returnString, "pri="+strconv.FormatInt(priority, 10))
	returnString = append(returnString, "api_ver="+strings.Join(versions, ","))
	returnString = append(returnString, "api_proto="+protocol)
	returnString = append(returnString, "api_auth="+strconv.FormatBool(oauth_mode))
	return returnString
}

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

func handleRegHealth(w http.ResponseWriter, r *http.Request) {

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

func handleNodeAPI(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(node)
	}
	fmt.Println(version, rpath)
}

func catchAllHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
}

func handleRequests() {
	a.Router = mux.NewRouter()
	// r.PathPrefix("/").HandlerFunc(catchAllHandler)

	// NODE API
	a.Router.HandleFunc("/x-nmos", handleNMOSBase)
	nodeSubRouter := a.Router.PathPrefix("/x-nmos/node").Subrouter()
	nodeSubRouter.HandleFunc("", handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}", handleNodeAPI)
	nodeSubRouter.HandleFunc("/{version}/{resourcePath}", handleNodeAPI).Methods("POST")

	// QUERY API
	querySubRouter := a.Router.PathPrefix("/x-nmos/query").Subrouter()
	querySubRouter.HandleFunc("", handleQueryAPI).Methods("POST")
	querySubRouter.HandleFunc("/{version}/", handleQueryAPI).Methods("POST")
	querySubRouter.HandleFunc("/{version}/{resourcePath}", handleQueryAPI).Methods("POST")

	// REGISTRATION API
	a.Router.HandleFunc("/x-nmos/registration/{version}", handleRegBase)
	a.Router.HandleFunc("/x-nmos/registration/{version}/resource", handleRegResource)
	a.Router.HandleFunc("/x-nmos/registration/{version}/{resourceType}/{resourceId}", handleGetResource)
	a.Router.HandleFunc("/x-nmos/registration/{version}/health/nodes/{nodeId}", handleRegHealth)

	a.srv = &http.Server{
		Addr: "0.0.0.0:3232",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      a.Router, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := a.srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
}
