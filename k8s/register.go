package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/fleet/client"
)

var (
	apiEndpoint   string
	fleetEndpoint string
	metadata      string
	syncInterval  int
)

func init() {
	log.SetFlags(0)
	flag.StringVar(&apiEndpoint, "api-endpoint", "", "kubernetes API endpoint")
	flag.IntVar(&syncInterval, "sync-interval", 30, "sync interval")
}

type Minion struct {
	Kind       string `json:"kind,omitempty"`
	ID         string `json:"id,omitempty"`
	HostIP     string `json:"hostIP,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

func register(endpoint, addr string) error {
	m := &Minion{
		Kind:       "Minion",
		APIVersion: "v1beta1",
		ID:         addr,
		HostIP:     addr,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1beta1/minions", endpoint)
	res, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 202 || res.StatusCode == 200 {
		log.Printf("registered machine: %s\n", addr)
		return nil
	}
	data, err = ioutil.ReadAll(res.Body)
	log.Printf("Response: %#v", res)
	log.Printf("Response Body:\n%s", string(data))
	return errors.New("error registering: " + addr)
}

func getMachines(endpoint string) ([]string, error) {
	dialFunc := net.Dial
	machineList := make([]string, 0)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "unix" {
		endpoint = "http://domain-sock/"
		dialFunc = func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", u.Path)
		}
	}
	c := &http.Client{
		Transport: &http.Transport{
			Dial:              dialFunc,
			DisableKeepAlives: true,
		},
	}
	fleetClient, err := client.NewHTTPClient(c, *u)
	if err != nil {
		return nil, err
	}
	machines, err := fleetClient.Machines()
	if err != nil {
		return nil, err
	}
	for _, m := range machines {
		if isHealthy(m.PublicIP) {
			machineList = append(machineList, m.PublicIP)
		}
	}
	return machineList, nil
}

func isHealthy(addr string) bool {
	url := fmt.Sprintf("http://%s:%d/healthz", addr, 10250)
	res, err := http.Get(url)
	if err != nil {
		log.Printf("error health checking %s: %s", addr, err)
		return false
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		return true
	}
	log.Printf("unhealthy machine: %s will not be registered", addr)
	return false
}

func main() {
	flag.Parse()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	fleetEndpoint = "unix:///var/run/fleet.sock"
	for {
		machines, err := getMachines(fleetEndpoint)
		if err != nil {
			log.Println(err)
		}
		for _, machine := range machines {
			if err := register(apiEndpoint, machine); err != nil {
				log.Println(err)
			}
		}
		select {
		case c := <-signalChan:
			log.Println(fmt.Sprintf("captured %v exiting...", c))
			os.Exit(0)
		case <-time.After(time.Duration(syncInterval) * time.Second):
			// Continue syncing machines.
		}
	}
}
