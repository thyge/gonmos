package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Node API Path Information
var NODE_APINAMESPACE string = "x-nmos"
var NODE_APINAME string = "node"
var NODE_APIROOT string = "/" + NODE_APINAMESPACE + "/" + NODE_APINAME + "/"
var NODE_APIVERSIONS = []string{"v1.0", "v1.1", "v1.2", "v1.3"}

// if PROTOCOL == "https":
//     NODE_APIVERSIONS.remove("v1.0")
var RESOURCE_TYPES = []string{"sources", "flows", "devices", "senders", "receivers"}

func RegisterWithRegistry() {
	// The Node registers itself with the Registration API by taking the object
	// it holds under the Node API’s /self resource and POSTing this to the
	// Registration API.

}

type NMOSNodeService struct {
	Node      NMOSNode
	APIRouter *mux.Router
	WebServer *http.Server
	MDNS      NMOSMdnsService
}

func (nns NMOSNodeService) BaseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("base handler hit")
	json.NewEncoder(w).Encode(NODE_APIVERSIONS)
}

func (nns NMOSNodeService) APIRootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("APIRoot handler hit")
	vars := mux.Vars(r)
	// Validate version here
	fmt.Println("NMOS API VERSION: ", vars["version"])
	json.NewEncoder(w).Encode(RESOURCE_TYPES)
}

func (nns NMOSNodeService) ResourceRootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Resource handler hit")
	vars := mux.Vars(r)
	// Validate version here
	fmt.Println("NMOS API VERSION: ", vars["version"])
	// Validate resource here
	// Check for self in resource
	fmt.Println("NMOS API VERSION: ", vars["resource_type"])

	switch vars["resource_type"] {
	case "self":
		json.NewEncoder(w).Encode(nns.Node)
	case "sources":
		json.NewEncoder(w).Encode(nns.Node)
	}
}

func CreateSampleNode() NMOSNode {
	node := new(NMOSNode)
	muuid, _ := uuid.NewRandom()
	node.Id = muuid.String()
	node.Hostname, _ = os.Hostname()
	node.Description = "gonmos example"
	node.Href = "http://127.0.0.1:80"
	return *node
}

func (nns NMOSNodeService) StartWebServer() {
	nns.Node = CreateSampleNode()
	nns.APIRouter = mux.NewRouter().StrictSlash(true)
	fmt.Println("Setting up routes")

	// NMOS routes
	nns.APIRouter.HandleFunc("/", nns.BaseHandler)
	nns.APIRouter.HandleFunc("/x-nmos", nns.BaseHandler)
	nns.APIRouter.HandleFunc("/x-nmos/node", nns.BaseHandler)
	nns.APIRouter.HandleFunc("/x-nmos/node/{version}", nns.APIRootHandler)
	nns.APIRouter.HandleFunc("/x-nmos/node/{version}/{resource_type}", nns.ResourceRootHandler)
	nns.APIRouter.HandleFunc("/x-nmos/node/{version}/{resource_type}/{id}", nns.ResourceRootHandler)

	srv := &http.Server{
		Handler: nns.APIRouter,
		Addr:    ":80",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	nns.WebServer = srv

	// Run our server in a goroutine so that it doesn't block.
	// go func() {
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Println(err)
	// 	}
	// }()
	log.Println(nns.WebServer.ListenAndServe())

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

	// Create a deadline to wait for.
	wait := time.Duration(0)
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	nns.WebServer.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
}

func (nns NMOSNodeService) StartNMDS() {
	nns.MDNS = CreateMDNService()
}

func main() {
	nns := new(NMOSNodeService)
	nns.StartNMDS()
	// nns.StartWebServer()
	// Find registry
	// Post SELF_NODE to registry

	os.Exit(0)
}
