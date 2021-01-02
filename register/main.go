package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
)

const (
	SERVICE_PORT       int    = 8235
	DNS_SD_HTTP_PORT   int    = 80
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

func main() {
	hname, err := os.Hostname()
	DNS_SD_NAME := "registration_" + hname + "_https"
	AGGREGATOR_APIVERSIONS := []string{"v1.0", "v1.1", "v1.2", "v1.3"}
	flag.Parse()

	mdnstxt := mdnsText(priority, AGGREGATOR_APIVERSIONS, "http", oauth_mode)
	server, err := zeroconf.Register(DNS_SD_NAME, DNS_SD_TYPE, *domain, DNS_SD_HTTP_PORT, mdnstxt, nil)
	if err != nil {
		panic(err)
	}
	defer server.Shutdown()

	handleRequests()

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		fmt.Println("Exit by user")
	}

	log.Println("Shutting down.")
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
	// vars := mux.Vars(r)
	// v := vars["version"]

}

func handleRegHealth(w http.ResponseWriter, r *http.Request) {

}

func handleRegBase(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// v := vars["version"]
	json.NewEncoder(w).Encode([]string{"resource/", "health/"})
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/x-nmos/registration/{version}", handleRegBase)
	myRouter.HandleFunc("/x-nmos/registration/{version}/resource", handleRegResource)
	myRouter.HandleFunc("/x-nmos/registration/{version}/health", handleRegHealth)
	log.Fatal(http.ListenAndServe(":80", myRouter))
}
