package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/honeycombio/honeycomb-opentracing-proxy/sinks"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type App struct {
	Port   string
	server *http.Server
	Sink   sinks.Sink
	Mirror *Mirror
}

// handleSpans handles the /api/v1/spans POST endpoint. It decodes the request
// body and normalizes it to a slice of types.Span instances. The Sink
// handles that slice. The Mirror, if configured, takes the request body
// verbatim and sends it to another host.
func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Info("Error reading request body")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error reading request"))
	}

	contentType := r.Header.Get("Content-Type")

	if a.Mirror != nil {
		err := a.Mirror.Send(payload{ContentType: contentType, Body: data})
		if err != nil {
			logrus.WithError(err).Info("Error mirroring data")
		}
	}

	var spans []*types.Span
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

	if err := a.Sink.Send(spans); err != nil {
		logrus.WithError(err).Info("error forwarding spans")
	}
	w.WriteHeader(http.StatusAccepted)
}

// ungzipWrap wraps a handleFunc and transparently ungzips the body of the
// request if it is gzipped
func ungzipWrap(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var newBody io.ReadCloser
		isGzipped := r.Header.Get("Content-Encoding")
		if isGzipped == "gzip" {
			buf := bytes.Buffer{}
			if _, err := io.Copy(&buf, r.Body); err != nil {
				logrus.WithError(err).Info("error allocating buffer for ungzipping")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error allocating buffer for ungzipping"))
				return
			}
			var err error
			newBody, err = gzip.NewReader(&buf)
			if err != nil {
				logrus.WithError(err).Info("error ungzipping span data")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error ungzipping span data"))
				return
			}
			r.Body = newBody
		}
		hf(w, r)
	}
}

func (a *App) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", ungzipWrap(a.handleSpans))

	// Handlers enable CORS preflight check support
	// If ORIGINS_ALLOWED is unset, Access-Control-Allow-Origin header will never be sent,
	// triggering a CORS preflight check error in browser that issues the request
	originsOk := handlers.AllowedOrigins(strings.Split(os.Getenv("ORIGINS_ALLOWED"), ";"))
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	methodsOk := handlers.AllowedMethods([]string{"POST", "OPTIONS"})

	a.server = &http.Server{
		Addr:    a.Port,
		Handler: handlers.CORS(originsOk, headersOk, methodsOk)(mux),
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

type payload struct {
	ContentType string
	Body        []byte
}

type Mirror struct {
	DownstreamURL  *url.URL
	BufSize        int
	MaxConcurrency int

	payloads chan payload
	stopped  bool
	wg       sync.WaitGroup
}

func (m *Mirror) Start() error {
	if m.MaxConcurrency == 0 {
		m.MaxConcurrency = 100
	}
	if m.BufSize == 0 {
		m.BufSize = 4096
	}
	m.payloads = make(chan payload, m.BufSize)
	for i := 0; i < m.MaxConcurrency; i++ {
		m.wg.Add(1)
		go m.runWorker()
	}
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

func (m *Mirror) runWorker() {
	for p := range m.payloads {
		r, err := http.NewRequest("POST", m.DownstreamURL.String(), bytes.NewReader(p.Body))
		r.Header.Set("Content-Type", p.ContentType)
		if err != nil {
			logrus.WithError(err).Info("Error building downstream request")
			return
		}
		client := &http.Client{}
		resp, err := client.Do(r)
		if err != nil {
			logrus.WithError(err).Info("Error sending payload downstream")
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			responseBody, _ := ioutil.ReadAll(&io.LimitedReader{R: resp.Body, N: 1024})
			logrus.WithField("status", resp.Status).
				WithField("response", string(responseBody)).
				Info("Error response sending payload downstream")
		}
	}
	m.wg.Done()
}

func (m *Mirror) Send(p payload) error {
	if m.stopped {
		return errors.New("sink stopped")
	}
	select {
	case m.payloads <- p:
		return nil
	default:
		return errors.New("sink full")
	}
}
