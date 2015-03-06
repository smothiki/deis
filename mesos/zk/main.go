package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
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
	etcdPath := getopt("ETCD_PATH", "/deis/mesos")

	client := etcd.NewClient([]string{"http://" + host + ":" + etcdPort})

	exitChan := make(chan os.Signal, 2)
	cleanupChan := make(chan bool)
	signal.Notify(exitChan, syscall.SIGTERM, syscall.SIGINT)

	go runService(cleanupChan)
	go publishService(client, host, etcdPath, uint64(ttl.Seconds()))

	<-cleanupChan
}

func runService(cleanupChan chan bool) error {
	cmd := exec.Command("/opt/zookeeper/bin/zkServer.sh", "start-foreground")

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

func publishService(client *etcd.Client, host string, etcdPath string, ttl uint64) {
	for {
		setEtcd(client, etcdPath+"/zk/hosts/"+host, host, ttl)
		time.Sleep(timeout)
	}
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
