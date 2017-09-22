package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
)

type App struct {
	Port      string
	server    *http.Server
	Forwarder Forwarder
}

func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling spans!")
	var spans []Span

	defer r.Body.Close()
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {

		if err := json.NewDecoder(r.Body).Decode(&spans); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("error unmarshaling spans!", err)
			w.Write([]byte("error unmarshaling span JSON"))
			return
		}
	} else if contentType == "application/x-thrift" {
		// TODO: gah! thrift!
	} else {
		fmt.Println("unknown content type", contentType)
	}

	if err := a.Forwarder.Forward(spans); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error unmarshaling span JSON"))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *App) Start() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", a.handleSpans)

	a.server = &http.Server{
		Addr:    a.Port,
		Handler: mux,
	}
	go a.server.ListenAndServe()
	fmt.Println("Listening!")
	return nil
}

func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

type Forwarder interface {
	Forward(spans []Span) error
}

type HoneycombForwarder struct {
	Writekey string
	Dataset  string
}

func (hf *HoneycombForwarder) Start() {
	libhoney.Init(libhoney.Config{
		WriteKey: hf.Writekey,
		Dataset:  hf.Dataset,
	})

	go func() {
		for resp := range libhoney.Responses() {
			fmt.Println("got libhoney response!", resp)
		}
	}()

}

func (hf *HoneycombForwarder) Stop() {
	libhoney.Close()
}

func (hf *HoneycombForwarder) Forward(spans []Span) error {
	for _, span := range spans {
		ev := convertSpan(span)
		// Error sending here probably means no events will succeed, so no
		// point returning error to client for client to retry.
		// Should do something better.
		ev.Send()
	}
	return nil
}

func main() {
	var port string
	var writekey string
	var dataset string
	flag.StringVar(&port, "p", ":9411", "port to listen on")
	flag.StringVar(&writekey, "k", "", "honeycomb write key")
	flag.StringVar(&dataset, "d", "", "honeycomb dataset name")
	flag.Parse()
	if writekey == "" {
		fmt.Println("No writekey provided")
		os.Exit(1)
	}
	if dataset == "" {
		fmt.Println("No dataset provided")
		os.Exit(1)
	}

	forwarder := &HoneycombForwarder{
		Writekey: writekey,
		Dataset:  dataset,
	}
	forwarder.Start()
	defer forwarder.Stop()

	a := &App{
		Port:      port,
		Forwarder: forwarder,
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

func convertSpan(span Span) *libhoney.Event {
	ev := libhoney.NewEvent()
	binaryAnnotations := make(map[string]string)
	for _, ba := range span.BinaryAnnotations {
		ev.AddField(ba.Key, ba.Value)
		binaryAnnotations[ba.Key] = string(ba.Value)
		// TODO: do something with endpoint value
	}

	// TODO: do something with annotations as well

	ev.AddField("traceID", span.TraceID)
	ev.AddField("name", span.Name)
	ev.AddField("durationMs", float64(span.Duration)/1e3)
	ev.Timestamp = time.Unix(span.Timestamp/1000000, (span.Timestamp%1000000)*1000)

	return ev
}

// Types below are from
// github.com/uber/jaeger-client-go/thrift-gen/zipkincore.Span
// except that the zipkin JSON API uses camelCase for field keys
// and IDs have string type
type Span struct {
	TraceID string `thrift:"trace_id,1" json:"traceId"`
	// unused field # 2
	Name        string        `thrift:"name,3" json:"name"`
	ID          string        `thrift:"id,4" json:"id"`
	ParentID    string        `thrift:"parent_id,5" json:"parentId,omitempty"`
	Annotations []*Annotation `thrift:"annotations,6" json:"annotations"`
	// unused field # 7
	BinaryAnnotations []*BinaryAnnotation `thrift:"binary_annotations,8" json:"binaryAnnotations"`
	Debug             bool                `thrift:"debug,9" json:"debug,omitempty"`
	Timestamp         int64               `thrift:"timestamp,10" json:"timestamp,omitempty"`
	Duration          int64               `thrift:"duration,11" json:"duration,omitempty"`
}

// An annotation is similar to a log statement. It includes a host field which
// allows these events to be attributed properly, and also aggregatable.
//
// Attributes:
//  - Timestamp: Microseconds from epoch.
//
// This value should use the most precise value possible. For example,
// gettimeofday or syncing nanoTime against a tick of currentTimeMillis.
//  - Value
//  - Host: Always the host that recorded the event. By specifying the host you allow
// rollup of all events (such as client requests to a service) by IP address.
type Annotation struct {
	Timestamp int64     `thrift:"timestamp,1" json:"timestamp"`
	Value     string    `thrift:"value,2" json:"value"`
	Host      *Endpoint `thrift:"host,3" json:"host,omitempty"`
}

type BinaryAnnotation struct {
	Key            string         `thrift:"key,1" json:"key"`
	Value          string         `thrift:"value,2" json:"value"`
	AnnotationType AnnotationType `thrift:"annotation_type,3" json:"annotationType"`
	Host           *Endpoint      `thrift:"host,4" json:"host,omitempty"`
}

type Endpoint struct {
	Ipv4        int32  `thrift:"ipv4,1" json:"ipv4"`
	Port        int16  `thrift:"port,2" json:"port"`
	ServiceName string `thrift:"service_name,3" json:"serviceName"`
}

type AnnotationType int64

const (
	AnnotationType_BOOL   AnnotationType = 0
	AnnotationType_BYTES  AnnotationType = 1
	AnnotationType_I16    AnnotationType = 2
	AnnotationType_I32    AnnotationType = 3
	AnnotationType_I64    AnnotationType = 4
	AnnotationType_DOUBLE AnnotationType = 5
	AnnotationType_STRING AnnotationType = 6
)
