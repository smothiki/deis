package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/deis/deis/deisctl/backend"
	"github.com/deis/deis/deisctl/utils"
)

// InstallPlatform loads all components' definitions from local unit files.
// After InstallPlatform, all components will be available for StartPlatform.
func InstallPlatform(b backend.Backend) error {

	if err := checkRequiredKeys(); err != nil {
		return err
	}

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	outchan <- utils.DeisIfy("Installing Deis...")

	installDefaultServices(b, &wg, outchan, errchan)

	wg.Wait()
	close(outchan)

	fmt.Println("Done.")
	fmt.Println()
	fmt.Println("Please run `deisctl start platform` to boot up Deis.")
	return nil
}

func installDefaultServices(b backend.Backend, wg *sync.WaitGroup, outchan chan string, errchan chan error) {

	outchan <- fmt.Sprintf("Storage subsystem...")
	b.Create([]string{"store-daemon", "store-monitor", "store-metadata", "store-volume", "store-gateway"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Logging subsystem...")
	b.Create([]string{"logger", "logspout"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Control plane...")
	b.Create([]string{"cache", "database", "registry", "controller", "builder"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Data plane...")
	b.Create([]string{"publisher"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Routing mesh...")
	b.Create([]string{"router@1", "router@2", "router@3"}, wg, outchan, errchan)
	wg.Wait()
}

// UninstallPlatform unloads all components' definitions.
// After UninstallPlatform, all components will be unavailable.
func UninstallPlatform(b backend.Backend) error {

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	outchan <- utils.DeisIfy("Uninstalling Deis...")

	uninstallAllServices(b, &wg, outchan, errchan)

	wg.Wait()
	close(outchan)

	fmt.Println("Done.")
	return nil
}

func uninstallAllServices(b backend.Backend, wg *sync.WaitGroup, outchan chan string, errchan chan error) error {

	outchan <- fmt.Sprintf("Routing mesh...")
	b.Destroy([]string{"router@1", "router@2", "router@3"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Data plane...")
	b.Destroy([]string{"publisher"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Control plane...")
	b.Destroy([]string{"controller", "builder", "cache", "database", "registry"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Logging subsystem...")
	b.Destroy([]string{"logger", "logspout"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Storage subsystem...")
	b.Destroy([]string{"store-volume", "store-gateway"}, wg, outchan, errchan)
	wg.Wait()
	b.Destroy([]string{"store-metadata"}, wg, outchan, errchan)
	wg.Wait()
	b.Destroy([]string{"store-daemon"}, wg, outchan, errchan)
	wg.Wait()
	b.Destroy([]string{"store-monitor"}, wg, outchan, errchan)
	wg.Wait()

	return nil
}

// StartPlatform activates all components.
func StartPlatform(b backend.Backend) error {

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	outchan <- utils.DeisIfy("Starting Deis...")

	startDefaultServices(b, &wg, outchan, errchan)

	wg.Wait()
	close(outchan)

	fmt.Println("Done.")
	fmt.Println()
	fmt.Println("Please use `deis register` to setup an administrator account.")
	return nil
}

func startDefaultServices(b backend.Backend, wg *sync.WaitGroup, outchan chan string, errchan chan error) {

	// create separate channels for background tasks
	_outchan := make(chan string)
	_errchan := make(chan error)
	var _wg sync.WaitGroup

	// wait for groups to come up
	outchan <- fmt.Sprintf("Storage subsystem...")
	b.Start([]string{"store-monitor"}, wg, outchan, errchan)
	wg.Wait()
	b.Start([]string{"store-daemon"}, wg, outchan, errchan)
	wg.Wait()
	b.Start([]string{"store-metadata"}, wg, outchan, errchan)
	wg.Wait()

	// we start gateway first to give metadata time to come up for volume
	b.Start([]string{"store-gateway"}, wg, outchan, errchan)
	wg.Wait()
	b.Start([]string{"store-volume"}, wg, outchan, errchan)
	wg.Wait()

	// start logging subsystem first to collect logs from other components
	outchan <- fmt.Sprintf("Logging subsystem...")
	b.Start([]string{"logger"}, wg, outchan, errchan)
	wg.Wait()
	b.Start([]string{"logspout"}, wg, outchan, errchan)
	wg.Wait()

	// optimization: start all remaining services in the background
	b.Start([]string{
		"cache", "database", "registry", "controller", "builder",
		"publisher", "router@1", "router@2", "router@3"},
		&_wg, _outchan, _errchan)

	outchan <- fmt.Sprintf("Control plane...")
	b.Start([]string{"cache", "database", "registry", "controller"}, wg, outchan, errchan)
	wg.Wait()
	b.Start([]string{"builder"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Data plane...")
	b.Start([]string{"publisher"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Routing mesh...")
	b.Start([]string{"router@1", "router@2", "router@3"}, wg, outchan, errchan)
	wg.Wait()
}

// StopPlatform deactivates all components.
func StopPlatform(b backend.Backend) error {

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	outchan <- utils.DeisIfy("Stopping Deis...")

	stopDefaultServices(b, &wg, outchan, errchan)

	wg.Wait()
	close(outchan)

	fmt.Println("Done.")
	fmt.Println()
	fmt.Println("Please run `deisctl start platform` to restart Deis.")
	return nil
}

func stopDefaultServices(b backend.Backend, wg *sync.WaitGroup, outchan chan string, errchan chan error) {

	outchan <- fmt.Sprintf("Routing mesh...")
	b.Stop([]string{"router@1", "router@2", "router@3"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Data plane...")
	b.Stop([]string{"publisher"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Control plane...")
	b.Stop([]string{"controller", "builder", "cache", "database", "registry"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Logging subsystem...")
	b.Stop([]string{"logger", "logspout"}, wg, outchan, errchan)
	wg.Wait()

	outchan <- fmt.Sprintf("Storage subsystem...")
	b.Stop([]string{"store-volume", "store-gateway"}, wg, outchan, errchan)
	wg.Wait()
	b.Stop([]string{"store-metadata"}, wg, outchan, errchan)
	wg.Wait()
	b.Stop([]string{"store-daemon"}, wg, outchan, errchan)
	wg.Wait()
	b.Stop([]string{"store-monitor"}, wg, outchan, errchan)
	wg.Wait()
}
