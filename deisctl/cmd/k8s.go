package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/deis/deis/deisctl/backend"
	"github.com/deis/deis/deisctl/config"
	"github.com/deis/deis/deisctl/utils"
)

//InstallK8s Installs K8s
func InstallK8s(b backend.Backend) error {
	outchan := make(chan string)
	errchan := make(chan error)
	defer close(outchan)
	defer close(errchan)
	var wg sync.WaitGroup
	go printState(outchan, errchan, 500*time.Millisecond)
	outchan <- utils.DeisIfy("Installing K8s...")
	outchan <- fmt.Sprintf("K8s API Server ...")
	b.Create([]string{"kube-apiserver"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s controller and scheduler ...")
	b.Create([]string{"kube-controller-manager", "kube-scheduler"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s proxy and kubelet ...")
	b.Create([]string{"kube-proxy", "kube-kubelet"}, &wg, outchan, errchan)
	wg.Wait()
	fmt.Println("Done.")
	fmt.Println()
	fmt.Println("Please run `deisctl start k8s` to start K8s.")
	return nil
}

func startDNS(outchan chan string, errchan chan error) {
	client, err := config.EtcdClient()
	if err != nil {
		errchan <- err
	}
	val, err := client.Get("/deis/scheduler/k8s/master")
	if err != nil {
		errchan <- err
	}

	var jsonStr = []byte(`{
  "kind": "ReplicationController",
  "spec": {
    "replicas": 1,
    "template": {
      "spec": {
        "dnsPolicy": "Default",
        "containers": [
          {
            "image": "gcr.io/google_containers/etcd:2.0.9",
            "command": [
              "/usr/local/bin/etcd",
              "-listen-client-urls",
              "http://127.0.0.1:2379,http://127.0.0.1:4001",
              "-advertise-client-urls",
              "http://127.0.0.1:2379,http://127.0.0.1:4001",
              "-initial-cluster-token",
              "skydns-etcd"
            ],
            "name": "etcd"
          },
          {
            "image": "gcr.io/google_containers/kube2sky:1.7",
            "args": [
              "-kube_master_url=http://` + val + `:8080",
              "-domain=cluster.local"
            ],
            "name": "kube2sky"
          },
          {
            "image": "gcr.io/google_containers/skydns:2015-03-11-001",
            "args": [
              "-machines=http://localhost:2379",
              "-addr=0.0.0.0:53",
              "-domain=cluster.local",
              "-nameservers=8.8.8.8:53,8.8.4.4:53"
            ],
            "name": "skydns",
            "livenessProbe": {
              "initialDelaySeconds": 30,
              "timeoutSeconds": 5,
              "exec": {
                "command": [
                  "/bin/sh",
                  "-c",
                  "nslookup kubernetes.default.cluster.local localhost >/dev/null"
                ]
              }
            },
            "ports": [
              {
                "protocol": "UDP",
                "containerPort": 53,
                "name": "dns"
              },
              {
                "protocol": "TCP",
                "containerPort": 53,
                "name": "dns-tcp"
              }
            ]
          }
        ]
      },
      "metadata": {
        "labels": {
          "k8s-app": "kube-dns",
          "kubernetes.io/cluster-service": "true"
        }
      }
    },
    "selector": {
      "k8s-app": "kube-dns"
    }
  },
  "apiVersion": "v1beta3",
  "metadata": {
    "labels": {
      "k8s-app": "kube-dns",
      "kubernetes.io/cluster-service": "true"
    },
    "namespace": "default",
    "name": "kube-dns"
  }
}`)
	url := "http://" + val + ":8080/api/v1beta3/namespaces/default/replicationcontrollers/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	repClient := &http.Client{}
	resp, err := repClient.Do(req)
	if err != nil {
		errchan <- err
	}
	defer resp.Body.Close()

	jsonStr = []byte(`{
  "kind": "Service",
  "spec": {
    "clusterIP": "10.100.0.10",
    "ports": [
      {
        "protocol": "UDP",
        "name": "dns",
        "port": 53
      },
      {
        "protocol": "TCP",
        "name": "dns-tcp",
        "port": 53
      }
    ],
    "selector": {
      "k8s-app": "kube-dns"
    }
  },
  "apiVersion": "v1",
  "metadata": {
    "labels": {
      "k8s-app": "kube-dns",
      "name": "kube-dns",
      "kubernetes.io/cluster-service": "true"
    },
    "namespace": "default",
    "name": "kube-dns"
  }
}`)
	servURL := "http://" + val + ":8080/api/v1beta3/namespaces/default/services/"
	servReq, err := http.NewRequest("POST", servURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		errchan <- err
	}
	servReq.Header.Set("Content-Type", "application/json")
	servClient := &http.Client{}
	servResp, err := servClient.Do(servReq)
	if err != nil {
		errchan <- err
	}
	defer servResp.Body.Close()

}

//StartK8s starts K8s Schduler
func StartK8s(b backend.Backend) error {
	outchan := make(chan string)
	errchan := make(chan error)
	defer close(outchan)
	defer close(errchan)
	var wg sync.WaitGroup
	go printState(outchan, errchan, 500*time.Millisecond)
	outchan <- utils.DeisIfy("Starting K8s...")
	outchan <- fmt.Sprintf("K8s API Server ...")
	b.Start([]string{"kube-apiserver"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s controller and scheduler ...")
	b.Start([]string{"kube-controller-manager", "kube-scheduler"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s proxy and kubelet ...")
	b.Start([]string{"kube-proxy", "kube-kubelet"}, &wg, outchan, errchan)
	wg.Wait()
	startDNS(outchan, errchan)
	fmt.Println("Done.")
	fmt.Println("Please run `deisctl config controller set schedulerModule=k8s` to use the K8s scheduler.")
	return nil
}

//StopK8s stops K8s
func StopK8s(b backend.Backend) error {

	outchan := make(chan string)
	errchan := make(chan error)
	defer close(outchan)
	defer close(errchan)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	outchan <- utils.DeisIfy("Stopping K8s...")
	outchan <- fmt.Sprintf("K8s proxy and kubelet ...")
	b.Stop([]string{"kube-proxy", "kube-kubelet"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s controller and scheduler ...")
	b.Stop([]string{"kube-controller-manager", "kube-scheduler"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s API Server ...")
	b.Stop([]string{"kube-apiserver"}, &wg, outchan, errchan)
	wg.Wait()
	fmt.Println("Done.")
	fmt.Println()
	return nil
}

func stopDNS(outchan chan string, errchan chan error) {
	client, err := config.EtcdClient()
	if err != nil {
		errchan <- err
	}
	val, err := client.Get("/deis/scheduler/k8s/master")
	if err != nil {
		errchan <- err
	}
	var jsonStr = []byte(`{}`)
	url := "http://" + val + ":8080/api/v1beta3/namespaces/default/replicationcontrollers/kube-dns"
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	k8Client := &http.Client{}
	resp, err := k8Client.Do(req)
	if err != nil {
		errchan <- err
	}
	defer resp.Body.Close()

	servURL := "http://" + val + ":8080/api/v1beta3/namespaces/default/services/kube-dns"
	servReq, err := http.NewRequest("DELETE", servURL, nil)
	if err != nil {
		errchan <- err
	}
	servClient := &http.Client{}
	servResp, err := servClient.Do(servReq)
	if err != nil {
		errchan <- err
	}
	defer servResp.Body.Close()

	podURL := "http://" + val + ":8080/api/v1beta3/namespaces/default/pods"
	podReq, err := http.NewRequest("GET", podURL, nil)
	podReq.Header.Set("Content-Type", "application/json")
	podClient := &http.Client{}
	podResp, err := podClient.Do(podReq)
	if err != nil {
		errchan <- err
	}
	defer podResp.Body.Close()

	body, _ := ioutil.ReadAll(podResp.Body)
	var pods map[string]interface{}
	err = json.Unmarshal(body, &pods)
	items := pods["items"].([]interface{})
	for i := range items {
		pod := items[i].(map[string]interface{})
		metadata := pod["metadata"].(map[string]interface{})
		if metadata["generateName"] == "kube-dns-" {
			url = "http://" + val + ":8080/api/v1beta3/namespaces/default/pods/" + metadata["name"].(string)
			delReq, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
			delReq.Header.Set("Content-Type", "application/json")
			delClient := &http.Client{}
			delResp, err := delClient.Do(delReq)
			if err != nil {
				errchan <- err
			}
			defer delResp.Body.Close()
		}
	}
}

//UnInstallK8s uninstall K8s
func UnInstallK8s(b backend.Backend) error {
	outchan := make(chan string)
	errchan := make(chan error)
	defer close(outchan)
	defer close(errchan)
	var wg sync.WaitGroup
	go printState(outchan, errchan, 500*time.Millisecond)
	outchan <- utils.DeisIfy("Destroying K8s...")
	stopDNS(outchan, errchan)
	outchan <- fmt.Sprintf("K8s proxy and kubelet ...")
	b.Destroy([]string{"kube-proxy", "kube-kubelet"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s controller and scheduler ...")
	b.Destroy([]string{"kube-controller-manager", "kube-scheduler"}, &wg, outchan, errchan)
	wg.Wait()
	outchan <- fmt.Sprintf("K8s API Server ...")
	b.Destroy([]string{"kube-apiserver"}, &wg, outchan, errchan)
	wg.Wait()
	fmt.Println("Done.")
	fmt.Println()
	return nil
}
