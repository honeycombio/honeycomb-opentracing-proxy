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

	Debug    bool   `long:"debug" description:"Also print spans to stdout"`
	Upstream string `long:"upstream" description:"Upstream host to forward span data along to (e.g., https://zipkin.example.com:9411)."`
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

	forwarder := &forwarders.CompositeForwarder{}
	forwarder.Add(
		&forwarders.HoneycombForwarder{
			Writekey: options.Writekey,
			Dataset:  options.Dataset,
		},
	)
	if options.Debug {
		forwarder.Add(&forwarders.StdoutForwarder{})
	}

	forwarder.Start()
	defer forwarder.Stop()

	a := &app.App{
		Port:      options.Port,
		Forwarder: forwarder,
		Upstream:  options.Upstream,
	}
	err = a.Start()
	if err != nil {
		fmt.Printf("Error starting app: %v\n", err)
		os.Exit(1)
	}
	defer a.Stop()
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
