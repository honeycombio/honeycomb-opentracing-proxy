package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/honeycombio/zipkinproxy/forwarders"
	"github.com/honeycombio/zipkinproxy/types"
)

type App struct {
	Port      string
	server    *http.Server
	Forwarder forwarders.Forwarder
}

func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling spans!")

	var spans []*types.Span
	var err error

	defer r.Body.Close()
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		spans, err = types.DecodeJSON(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("error unmarshaling spans!", err)
			w.Write([]byte("error unmarshaling span JSON"))
			return
		}
	} else if contentType == "application/x-thrift" {
		spans, err = types.DecodeThrift(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("error unmarshaling spans!", err)
			w.Write([]byte("error unmarshaling span JSON"))
			return
		}
	} else {
		fmt.Println("unknown content type", contentType)
	}

	if err := a.Forwarder.Forward(spans); err != nil {
		// TODO log something
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
