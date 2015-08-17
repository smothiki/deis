package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/deis/deis/logger/syslogd"
)

var (
	logAddr         string
	logPort         int
	drainURI        string
	enablePublish   bool
	publishHost     string
	publishPath     string
	publishPort     string
	publishInterval int
	publishTTL      int
)

func init() {
	flag.StringVar(&logAddr, "log-addr", "0.0.0.0", "bind address for the logger")
	flag.IntVar(&logPort, "log-port", 514, "bind port for the logger")
	flag.StringVar(&drainURI, "drain-uri", "", "default drainURI, once set in etcd, this has no effect.")
	flag.StringVar(&syslogd.LogRoot, "log-root", "/data/logs", "log path to store logs")
	flag.BoolVar(&enablePublish, "enable-publish", false, "enable publishing to service discovery")
	flag.StringVar(&publishHost, "publish-host", getHostIP("127.0.0.1"), "service discovery hostname")
	flag.IntVar(&publishInterval, "publish-interval", 10, "publish interval in seconds")
	flag.StringVar(&publishPath, "publish-path", getopt("ETCD_PATH", "/deis/logs"), "path to publish host/port information")
	flag.StringVar(&publishPort, "publish-port", getopt("ETCD_PORT", "4001"), "service discovery port")
	flag.IntVar(&publishTTL, "publish-ttl", publishInterval*2, "publish TTL in seconds")
}

func main() {
	flag.Parse()

	client := etcd.NewClient([]string{"http://" + publishHost + ":" + publishPort})
	ticker := time.NewTicker(time.Duration(publishInterval) * time.Second)
	signalChan := make(chan os.Signal, 1)
	drainChan := make(chan string)
	exitChan := make(chan bool)
	cleanupChan := make(chan bool)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	// ensure the drain key exists in etcd.
	if _, err := client.Get(publishPath+"/drain", false, false); err != nil {
		setEtcd(client, publishPath+"/drain", drainURI, 0)
	}

	go syslogd.Listen(exitChan, cleanupChan, drainChan, fmt.Sprintf("%s:%d", logAddr, logPort))
	if enablePublish {
		publishKeys(client, publishHost, publishPath, strconv.Itoa(logPort), uint64(time.Duration(publishTTL)*time.Second))
	}

	for {
		select {
		case <-ticker.C:
			if enablePublish {
				publishKeys(client, publishHost, publishPath, strconv.Itoa(logPort), uint64(time.Duration(publishTTL)*time.Second))
			}
			// HACK (bacongobbler): poll etcd every publishInterval for changes in the log drain value.
			// etcd's .Watch() implementation is broken when you use TTLs
			//
			// https://github.com/coreos/etcd/issues/2679
			resp, err := client.Get(publishPath+"/drain", false, false)
			if err != nil {
				log.Printf("warning: could not retrieve drain URI from etcd: %v\n", err)
				continue
			}
			if resp != nil && resp.Node != nil {
				drainChan <- resp.Node.Value
			}
		case <-signalChan:
			close(exitChan)
		case <-cleanupChan:
			ticker.Stop()
			return
		}
	}
}

// publishKeys sets relevant etcd keys with a time-to-live.
func publishKeys(client *etcd.Client, host, etcdPath, port string, ttl uint64) {
	setEtcd(client, etcdPath+"/host", host, ttl)
	setEtcd(client, etcdPath+"/port", port, ttl)
}

func setEtcd(client *etcd.Client, key, value string, ttl uint64) {
	_, err := client.Set(key, value, ttl)
	if err != nil && !strings.Contains(err.Error(), "Key already exists") {
		log.Println(err)
	}
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func getHostIP(dfault string) string {
	f, err := os.Open("/etc/environment")
	if err != nil {
		log.Println(err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		s := strings.Split(line, "=")
		name, ip := s[0], s[1]
		if name == "COREOS_PRIVATE_IPV4" {
			return ip
		}
	}
	return dfault
}
