package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deis/deis/deisctl/backend"
	"github.com/deis/deis/deisctl/config"

	docopt "github.com/docopt/docopt-go"
)

const (
	// PlatformCommand is shorthand for "all the Deis components."
	PlatformCommand string = "platform"

	// MesosCommand is shorthand for "all Mesos components."
	MesosCommand string = "mesos"
)

// ListUnits prints a list of installed units.
func ListUnits(argv []string, b backend.Backend) error {
	usage := `Prints a list of installed units.

Usage:
  deisctl list [options]
`
	// parse command-line arguments
	if _, err := docopt.Parse(usage, argv, true, "", false); err != nil {
		return err
	}
	return b.ListUnits()
}

// ListUnitFiles prints the contents of all defined unit files.
func ListUnitFiles(argv []string, b backend.Backend) error {
	err := b.ListUnitFiles()
	return err
}

// Scale grows or shrinks the number of running components.
// Currently "router" is the only type that can be scaled.
func Scale(argv []string, b backend.Backend) error {
	usage := `Grows or shrinks the number of running components.

Currently "router" is the only type that can be scaled.

Usage:
  deisctl scale [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}
	targets := args["<target>"].([]string)

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	for _, target := range targets {
		component, num, err := splitScaleTarget(target)
		if err != nil {
			return err
		}
		// the router is the only component that can scale at the moment
		if !strings.Contains(component, "router") {
			return fmt.Errorf("cannot scale %s components", component)
		}
		b.Scale(component, num, &wg, outchan, errchan)
		wg.Wait()
	}
	close(outchan)
	return nil
}

// Start activates the specified components.
func Start(argv []string, b backend.Backend) error {
	usage := `Activates the specified components.

Usage:
  deisctl start [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	// check for special targets
	targets := args["<target>"].([]string)
	if len(targets) == 1 {
		target := targets[0]
		if target == PlatformCommand {
			return StartPlatform(b)
		}
		if target == MesosCommand {
			return StartMesos(b)
		}
	}

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	b.Start(targets, &wg, outchan, errchan)
	wg.Wait()
	close(outchan)

	return nil
}

// checkRequiredKeys exist in etcd
func checkRequiredKeys() error {
	if err := config.CheckConfig("/deis/platform/", "domain"); err != nil {
		return fmt.Errorf(`Missing platform domain, use:
deisctl config platform set domain=<your-domain>`)
	}

	if err := config.CheckConfig("/deis/platform/", "sshPrivateKey"); err != nil {
		fmt.Printf(`Warning: Missing sshPrivateKey, "deis run" will be unavailable. Use:
deisctl config platform set sshPrivateKey=<path-to-key>
`)
	}
	return nil
}

// Stop deactivates the specified components.
func Stop(argv []string, b backend.Backend) error {
	usage := `Deactivates the specified components.

Usage:
  deisctl stop [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	// check for special targets
	targets := args["<target>"].([]string)
	if len(targets) == 1 {
		target := targets[0]
		if target == PlatformCommand {
			return StopPlatform(b)
		}
		if target == MesosCommand {
			return StopMesos(b)
		}
	}

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	b.Stop(targets, &wg, outchan, errchan)
	wg.Wait()
	close(outchan)

	return nil
}

// Restart stops and then starts the specified components.
func Restart(argv []string, b backend.Backend) error {
	usage := `Stops and then starts the specified components.

Usage:
  deisctl restart [<target>...] [options]
`
	// parse command-line arguments
	if _, err := docopt.Parse(usage, argv, true, "", false); err != nil {
		return err
	}

	// act as if the user called "stop" and then "start"
	argv[0] = "stop"
	if err := Stop(argv, b); err != nil {
		return err
	}
	argv[0] = "start"
	return Start(argv, b)
}

// Status prints the current status of components.
func Status(argv []string, b backend.Backend) error {
	usage := `Prints the current status of components.

Usage:
  deisctl status [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	targets := args["<target>"].([]string)
	for _, target := range targets {
		if err := b.Status(target); err != nil {
			return err
		}
	}
	return nil
}

// Journal prints log output for the specified components.
func Journal(argv []string, b backend.Backend) error {
	usage := `Prints log output for the specified components.

Usage:
  deisctl journal [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	targets := args["<target>"].([]string)
	for _, target := range targets {
		if err := b.Journal(target); err != nil {
			return err
		}
	}
	return nil
}

// Install loads the definitions of components from local unit files.
// After Install, the components will be available to Start.
func Install(argv []string, b backend.Backend) error {
	usage := `Loads the definitions of components from local unit files.

After install, the components will be available to start.

"deisctl install" looks for unit files in these directories, in this order:
- the $DEISCTL_UNITS environment variable, if set
- $HOME/.deis/units
- /var/lib/deis/units

Usage:
  deisctl install [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	// check for special targets
	targets := args["<target>"].([]string)
	if len(targets) == 1 {
		target := targets[0]
		if target == PlatformCommand {
			return InstallPlatform(b)
		}
		if target == MesosCommand {
			return InstallMesos(b)
		}
	}

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	// otherwise create the specific targets
	b.Create(targets, &wg, outchan, errchan)
	wg.Wait()

	close(outchan)
	return nil
}

// Uninstall unloads the definitions of the specified components.
// After Uninstall, the components will be unavailable until Install is called.
func Uninstall(argv []string, b backend.Backend) error {
	usage := `Unloads the definitions of the specified components.

After uninstall, the components will be unavailable until install is called.

Usage:
  deisctl uninstall [<target>...] [options]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}

	// check for special targets
	targets := args["<target>"].([]string)
	if len(targets) == 1 {
		target := targets[0]
		if target == PlatformCommand {
			return UninstallPlatform(b)
		}
		if target == MesosCommand {
			return UninstallMesos(b)
		}
	}

	outchan := make(chan string)
	errchan := make(chan error)
	var wg sync.WaitGroup

	go printState(outchan, errchan, 500*time.Millisecond)

	// uninstall the specific target
	b.Destroy(targets, &wg, outchan, errchan)
	wg.Wait()
	close(outchan)

	return nil
}

func printState(outchan chan string, errchan chan error, interval time.Duration) error {
	for {
		select {
		case out := <-outchan:
			// done on closed channel
			if out == "" {
				return nil
			}
			fmt.Println(out)
		case err := <-errchan:
			if err != nil {
				fmt.Println(err.Error())
				return err
			}
		}
		time.Sleep(interval)
	}
}

func splitScaleTarget(target string) (c string, num int, err error) {
	r := regexp.MustCompile(`([a-z-]+)=([\d]+)`)
	match := r.FindStringSubmatch(target)
	if len(match) == 0 {
		err = fmt.Errorf("Could not parse: %v", target)
		return
	}
	c = match[1]
	num, err = strconv.Atoi(match[2])
	if err != nil {
		return
	}
	return
}

// Config gets or sets a configuration value from the cluster.
//
// A configuration value is stored and retrieved from a key/value store (in this case, etcd)
// at /deis/<component>/<config>. Configuration values are typically used for component-level
// configuration, such as enabling TLS for the routers.
func Config(argv []string) error {
	usage := `Gets or sets a configuration value from the cluster.

A configuration value is stored and retrieved from a key/value store
(in this case, etcd) at /deis/<component>/<config>. Configuration
values are typically used for component-level configuration, such as
enabling TLS for the routers.

Usage:
  deisctl config <target> get [<key>...] [options]
  deisctl config <target> set <key=val>... [options]

Options:
  --verbose		print out the request bodies [default: false]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		return err
	}
	if err := config.Config(args); err != nil {
		return err
	}
	return nil
}

// RefreshUnits overwrites local unit files with those requested.
// Downloading from the Deis project GitHub URL by tag or SHA is the only mechanism
// currently supported.
func RefreshUnits(argv []string) error {
	usage := `Overwrites local unit files with those requested.

Downloading from the Deis project GitHub URL by tag or SHA is the only mechanism
currently supported.

"deisctl install" looks for unit files in these directories, in this order:
- the $DEISCTL_UNITS environment variable, if set
- $HOME/.deis/units
- /var/lib/deis/units

Usage:
  deisctl refresh-units [-p <target>] [-t <tag>]

Options:
  -p --path=<target>   where to save unit files [default: $HOME/.deis/units]
  -t --tag=<tag>       git tag, branch, or SHA to use when downloading unit files
                       [default: master]
`
	// parse command-line arguments
	args, err := docopt.Parse(usage, argv, true, "", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(2)
	}
	dir := args["--path"].(string)
	if dir == "$HOME/.deis/units" || dir == "~/.deis/units" {
		dir = path.Join(os.Getenv("HOME"), ".deis", "units")
	}
	// create the target dir if necessary
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// download and save the unit files to the specified path
	rootURL := "https://raw.githubusercontent.com/deis/deis/"
	tag := args["--tag"].(string)
	units := []string{
		"deis-builder.service",
		"deis-cache.service",
		"deis-controller.service",
		"deis-database.service",
		"deis-logger.service",
		"deis-logspout.service",
		"deis-publisher.service",
		"deis-registry.service",
		"deis-router.service",
		"deis-store-daemon.service",
		"deis-store-gateway.service",
		"deis-store-metadata.service",
		"deis-store-monitor.service",
		"deis-store-volume.service",
	}
	for _, unit := range units {
		src := rootURL + tag + "/deisctl/units/" + unit
		dest := filepath.Join(dir, unit)
		res, err := http.Get(src)
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			return errors.New(res.Status)
		}
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		fmt.Printf("Refreshed %s from %s\n", unit, tag)
	}
	return nil
}
