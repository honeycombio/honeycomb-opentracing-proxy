package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/app"
	"github.com/honeycombio/honeycomb-opentracing-proxy/sinks"
	flag "github.com/jessevdk/go-flags"
)

type Options struct {
	Writekey   string   `long:"writekey" short:"k" description:"Team write key"`
	Dataset    string   `long:"dataset" short:"d" description:"Name of the dataset to send events to"`
	Port       string   `long:"port" short:"p" description:"Port to listen on" default:":9411"`
	APIHost    string   `long:"api_host" description:"Hostname for the Honeycomb API server" default:"https://api.honeycomb.io/"`
	Debug      bool     `long:"debug" description:"Also print spans to stdout"`
	Downstream string   `long:"downstream" description:"A host to forward span data along to (e.g., https://zipkin.example.com:9411). Use this to send data to Honeycomb and another Zipkin-compatible backend."`
	DropFields []string `long:"drop_field" description:"Drop any span tags with this name instead of sending them to Honeycomb. You can specify this multiple times."`
	SampleRate uint     `long:"samplerate" description:"Only forward a sampled subset of traces to Honeycomb. Passing --samplerate=10 will forward 1 out of 10 traces."`
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

	sink := &sinks.CompositeSink{}
	sink.Add(
		&sinks.HoneycombSink{
			Writekey:   options.Writekey,
			Dataset:    options.Dataset,
			APIHost:    options.APIHost,
			DropFields: options.DropFields,
			SampleRate: options.SampleRate,
		},
	)
	if options.Debug {
		sink.Add(&sinks.StdoutSink{})
		logrus.SetLevel(logrus.DebugLevel)
	}

	sink.Start()
	defer sink.Stop()

	var mirror *app.Mirror
	if options.Downstream != "" {
		downstreamURL, err := url.Parse(options.Downstream)
		if err != nil {
			fmt.Printf("Invalid downstream url %s\n", options.Downstream)
			os.Exit(1)
		}

		scheme := downstreamURL.Scheme
		isHTTP := scheme == "http" || scheme == "https"
		if !isHTTP {
			fmt.Printf("Invalid downstream url %s. Must be prefixed with http:// or https://\n", options.Downstream)
			os.Exit(1)
		}

		mirror = &app.Mirror{
			DownstreamURL: downstreamURL,
		}
		mirror.Start()
		defer mirror.Stop()
	}

	a := &app.App{
		Port:   options.Port,
		Sink:   sink,
		Mirror: mirror,
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
