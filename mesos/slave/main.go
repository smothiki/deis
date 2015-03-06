package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

const (
	timeout time.Duration = 10 * time.Second
	ttl     time.Duration = timeout * 2
)

func main() {
	host := getopt("HOST", "127.0.0.1")

	etcdPort := getopt("ETCD_PORT", "4001")

	client := etcd.NewClient([]string{"http://" + host + ":" + etcdPort})

	exitChan := make(chan os.Signal, 2)
	cleanupChan := make(chan bool)
	signal.Notify(exitChan, syscall.SIGTERM, syscall.SIGINT)

	go runService(cleanupChan, client)

	<-cleanupChan
}

func runService(cleanupChan chan bool, c *etcd.Client) error {

	args, err := gatherArgs(c)
	if err != nil {
		return err
	}

	cmd := exec.Command("mesos-slave", args...)

	stderrPipe, err := cmd.StderrPipe()
	stdoutPipe, err := cmd.StdoutPipe()

	if err != nil {
		return fmt.Errorf("Error: could not create pipes")
	}

	go streamLineOutput(stdoutPipe, os.Stdout)
	go streamLineOutput(stderrPipe, os.Stderr)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err.Error())
		cleanupChan <- true
	}

	return nil
}

func gatherArgs(c *etcd.Client) ([]string, error) {
	var args []string

	// discover zk hosts from etcd
	resp, err := getEtcd(c, "/deis/mesos/zk/hosts", true)
	if err != nil {
		return []string{}, err
	}

	// append zk hosts
	var hosts []string
	for _, node := range resp.Node.Nodes {
		hosts = append(hosts, node.Value+":2181")
	}
	zkHosts := strings.Join(hosts, ",")
	args = append(args, "--master=zk://"+zkHosts+"/mesos")

	// only use docker containerizer
	args = append(args, "--containerizers=docker,mesos")

	fmt.Printf("%v\n", args)

	return args, nil
}

// streamOutput from a source reader to destination writer
func streamLineOutput(src io.Reader, out io.Writer) error {

	s := bufio.NewReader(src)

	for {
		var line []byte
		line, err := s.ReadSlice('\n')
		if err == io.EOF && len(line) == 0 {
			break // done
		}
		if err == io.EOF {
			return fmt.Errorf("Improper termination: %v", line)
		}
		if err != nil {
			return err
		}

		out.Write(line)
	}

	return nil
}

func getEtcd(client *etcd.Client, key string, recursive bool) (*etcd.Response, error) {
	resp, err := client.Get(key, recursive, false)
	if err != nil {
		log.Println(err)
	}
	return resp, nil
}

func setEtcd(client *etcd.Client, key, value string, ttl uint64) {
	_, err := client.Set(key, value, ttl)
	if err != nil {
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
