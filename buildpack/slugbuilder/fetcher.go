package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/coreos/go-etcd/etcd"
)

const (
	downloadDir = "/app/"
	cmdstring   = "/tmp/builder/build.sh"
)

func main() {
	host := getBuilderHost()
	// _ = downloadFromURL("http://"+host+":"+"3000/git/home/"+os.Getenv("APP")+"/tar", os.Getenv("APP"))
	_ = downloadFromURL("http://"+host+":"+"3000/git/home/"+os.Getenv("APP")+"/tar", os.Getenv("APP"))
	os.Chdir(downloadDir)
	cmd := exec.Command("sh", "-c", "tar -xzf "+os.Getenv("APP"))
	if out, err := cmd.Output(); err != nil {
		fmt.Printf("%v\nOutput:\n%v", err, string(out))
	} else {
		fmt.Println("ok")
	}
	fmt.Println(os.Getwd())
	os.Remove(os.Getenv("APP"))
	cmd = exec.Command("sh", "-c", cmdstring)
	if out, err := cmd.Output(); err != nil {
		fmt.Printf("%v\nOutput:\n%v", err, string(out))
	} else {
		fmt.Println("ok")
	}
	uploadTarfile(host)
}

func getBuilderHost() string {
	host := os.Getenv("HOST")
	if host == "" {
		fmt.Println("fetch from etcd")
		machines := []string{"http://" + os.Getenv("HOST") + ":" + "4001"}
		result, _ := etcd.NewClient(machines).Get("/deis/builder/host", true, true)
		host = result.Node.Value
	}
	return host
}
func downloadFromURL(url, fileName string) (err error) {
	fmt.Printf("Downloading %s to %s", url, fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(downloadDir + fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
	return
}
func uploadTarfile(host string) {
	client := &http.Client{}
	buf := bytes.NewBuffer(nil)
	f, _ := os.Open("/tmp/slug.tgz") // Error handling elided for brevity.
	io.Copy(buf, f)                  // Error handling elided for brevity.
	f.Close()
	req, _ := http.NewRequest("POST", "http://"+host+":"+"3000/git/home/"+os.Getenv("APP")+"/push", buf)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}
