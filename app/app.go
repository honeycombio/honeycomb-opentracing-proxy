package app

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/zipkinproxy/forwarders"
	"github.com/honeycombio/zipkinproxy/types"
)

type App struct {
	Port        string
	server      *http.Server
	Forwarder   forwarders.Forwarder
	Upstream    string
	upstreamUrl *url.URL

	testWaitGroup *sync.WaitGroup
}

func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Info("Error reading request body")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error reading request"))
	}

	if a.upstreamUrl != nil {
		if a.testWaitGroup != nil {
			a.testWaitGroup.Add(1)
		}
		go sendUpstream(a.upstreamUrl.String(), bytes.NewReader(data), a.testWaitGroup)
	}

	var spans []*types.Span
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "application/json":
		spans, err = types.DecodeJSON(bytes.NewReader(data))
	case "application/x-thrift":
		spans, err = types.DecodeThrift(bytes.NewReader(data))
	default:
		logrus.WithField("contentType", contentType).Info("unknown content type")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unknown content type"))
		return
	}
	if err != nil {
		logrus.WithError(err).WithField("type", contentType).Info("error unmarshaling spans")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error unmarshaling span data"))
		return
	}

	if err := a.Forwarder.Forward(spans); err != nil {
		logrus.WithError(err).Info("error forwarding spans")
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *App) Start() error {

	if a.Upstream != "" {
		var err error
		a.upstreamUrl, err = url.Parse(a.Upstream)
		if err != nil {
			return err
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", a.handleSpans)

	a.server = &http.Server{
		Addr:    a.Port,
		Handler: mux,
	}
	go a.server.ListenAndServe()
	logrus.WithField("port", a.Port).Info("Listening")
	return nil
}

func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

func sendUpstream(upstream string, body io.Reader, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	r, err := http.NewRequest("POST", upstream, body)
	if err != nil {
		logrus.WithError(err).Info("Error building upstream request")
		return
	}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logrus.WithError(err).Info("Error sending payload upstream")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		logrus.WithField("status", resp.Status).Info("Error sending payload upstream")
	}
}
