package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/honeycombio/zipkinproxy/app"
	"github.com/honeycombio/zipkinproxy/forwarders"
	flag "github.com/jessevdk/go-flags"
)

type Options struct {
	Writekey string `long:"writekey" short:"k" description:"Team write key"`
	Dataset  string `long:"dataset" short:"d" description:"Name of the dataset to send events to"`
	Port     string `long:"port" short:"p" description:"Port to listen on" default:":9411"`
	APIHost  string `long:"api_host" description:"Hostname for the Honeycomb API server" default:"https://api.honeycomb.io/"`

	Debug bool `long:"debug" description:"Also print spans to stdout"`
}

func main() {
	options, err := parseFlags()
	if err != nil {
		fmt.Println("Error parsing options:", err)
		os.Exit(1)
	}
	if !options.Debug && options.Writekey == "" {
		fmt.Println("No writekey provided")
		os.Exit(1)
	}
	if !options.Debug && options.Dataset == "" {
		fmt.Println("No dataset provided")
		os.Exit(1)
	}

	var fw []forwarders.Forwarder

	honeycombForwarder := &forwarders.HoneycombForwarder{
		Writekey: options.Writekey,
		Dataset:  options.Dataset,
	}
	honeycombForwarder.Start()
	defer honeycombForwarder.Stop()
	fw = append(fw, honeycombForwarder)

	if options.Debug {
		stdoutForwarder := &forwarders.StdoutForwarder{}
		stdoutForwarder.Start()
		defer stdoutForwarder.Stop()
		fw = append(fw, stdoutForwarder)
	}

	a := &app.App{
		Port:       options.Port,
		Forwarders: fw,
	}
	defer a.Stop()
	a.Start()
	waitForSignal()
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	defer close(ch)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)
	<-ch
}

func parseFlags() (*Options, error) {
	var options Options
	flagParser := flag.NewParser(&options, flag.Default)
	extraArgs, err := flagParser.Parse()

	if err != nil {
		if flagErr, ok := err.(*flag.Error); ok && flagErr.Type == flag.ErrHelp {
			os.Exit(0)
		} else {
			return nil, err
		}
	} else if len(extraArgs) != 0 {
		fmt.Printf("Unexpected extra arguments: %s\n", strings.Join(extraArgs, " "))
		return nil, errors.New("")
	}
	return &options, nil
}
