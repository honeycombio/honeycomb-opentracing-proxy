package app

import (
	"bytes"
	"context"
	"errors"
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
	Port      string
	server    *http.Server
	Forwarder forwarders.Forwarder
	Mirror    *Mirror
}

func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Info("Error reading request body")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error reading request"))
	}

	if a.Mirror != nil {
		err := a.Mirror.Send(data)
		if err != nil {
			logrus.WithError(err).Info("Error mirroring data")
		}
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

type Mirror struct {
	UpstreamURL *url.URL
	BufSize     int

	payloads chan []byte
	stopped  bool
	wg       sync.WaitGroup
}

func (m *Mirror) Start() error {
	if m.BufSize == 0 {
		m.BufSize = 4096
	}
	m.payloads = make(chan []byte, m.BufSize)
	m.wg.Add(1)
	go m.run()
	return nil
}

func (m *Mirror) Stop() error {
	m.stopped = true
	if m.payloads == nil {
		return nil
	}
	close(m.payloads)
	m.wg.Wait()
	return nil
}

func (m *Mirror) run() {
	for p := range m.payloads {
		r, err := http.NewRequest("POST", m.UpstreamURL.String(), bytes.NewReader(p))
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
			responseBody, _ := ioutil.ReadAll(&io.LimitedReader{resp.Body, 1024})
			logrus.WithField("status", resp.Status).
				WithField("response", string(responseBody)).
				Info("Error response sending payload upstream")
		}
	}
	m.wg.Done()
}

func (m *Mirror) Send(data []byte) error {
	if m.stopped {
		return errors.New("sink stopped")
	}
	select {
	case m.payloads <- data:
		return nil
	default:
		return errors.New("sink full")
	}
}
