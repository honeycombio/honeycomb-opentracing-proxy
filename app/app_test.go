package app

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/honeycombio/zipkinproxy/types"
	"github.com/stretchr/testify/assert"
)

type MockSink struct {
	spans []*types.Span
}

func (ms *MockSink) Send(spans []*types.Span) error {
	ms.spans = append(ms.spans, spans...)
	return nil
}

func (ms *MockSink) Start() error { return nil }
func (ms *MockSink) Stop() error  { return nil }

// TestThriftDecoding takes a capture of a zipkin thrift payload, and ensures
// that it's decoded and forwarded correctly.
func TestThriftDecoding(t *testing.T) {
	assert := assert.New(t)
	ms := &MockSink{}

	a := &App{Sink: ms}

	thriftPayload, err := os.Open("testdata/payload_0.thrift")
	assert.NoError(err)
	r := httptest.NewRequest("POST", "/api/v1/spans", thriftPayload)
	r.Header.Add("Content-Type", "application/x-thrift")
	w := httptest.NewRecorder()
	a.handleSpans(w, r)
	assert.Equal(w.Code, http.StatusAccepted)
	expectedSpans := []*types.Span{
		&types.Span{
			TraceID:  "350565b6a90d4c8c",
			Name:     "/api.RetrieverService/Fetch",
			ID:       "3ba1d9a5451f81c4",
			ParentID: "350565b6a90d4c8c",
			BinaryAnnotations: map[string]interface{}{
				"component": "gRPC",
			},
			Timestamp:  time.Date(2017, 9, 28, 20, 15, 17, 286440000, time.UTC),
			DurationMs: 2.155,
			// TODO where's the endpoint data here?
		},
		&types.Span{
			TraceID:     "350565b6a90d4c8c",
			Name:        "persist",
			ID:          "34472e70cb669b31",
			ParentID:    "350565b6a90d4c8c",
			ServiceName: "poodle",
			HostIPv4:    "10.129.211.111",
			BinaryAnnotations: map[string]interface{}{
				"lc":             "poodle",
				"responseLength": "136", // TODO verify :/
			},
			Timestamp:  time.Date(2017, 9, 28, 20, 15, 17, 288651000, time.UTC),
			DurationMs: 0.192,
		},
		&types.Span{
			TraceID:     "350565b6a90d4c8c",
			Name:        "markAsDone",
			ID:          "2eb1b7009815c803",
			ParentID:    "350565b6a90d4c8c",
			ServiceName: "poodle",
			HostIPv4:    "10.129.211.111",
			BinaryAnnotations: map[string]interface{}{
				"lc": "poodle",
			},
			Timestamp:  time.Date(2017, 9, 28, 20, 15, 17, 288847000, time.UTC),
			DurationMs: 5.134,
		},
		&types.Span{
			TraceID:     "350565b6a90d4c8c",
			Name:        "executeQuery",
			ID:          "350565b6a90d4c8c",
			ParentID:    "",
			ServiceName: "poodle",
			HostIPv4:    "10.129.211.111",
			BinaryAnnotations: map[string]interface{}{
				"lc":             "poodle",
				"dataset_id":     "90",
				"hidden_reason":  "0",
				"hostname":       "sea-of-dreams",
				"jaeger.version": "Go-2.8.0",
				"query_hash":     "fca2835dced5d6fafb4eb9dd",
				"query_run_pk":   "7AREu8scycJ",
				"sampler.param":  true,
				"sampler.type":   "const",
				"team_id":        "12",
				"user_id":        "15",
			},
			Timestamp:  time.Date(2017, 9, 28, 20, 15, 17, 284010000, time.UTC),
			DurationMs: 9.98,
		},
	}
	assert.Equal(ms.spans[:4], expectedSpans)
	assert.Equal(w.Code, http.StatusAccepted)
}

// TestMirroring tests the mirroring of unmodified request data to a downstream
// service.
func TestMirroring(t *testing.T) {
	assert := assert.New(t)
	m := newMockDownstream()
	defer m.server.Close()
	ms := &MockSink{}

	url, err := url.Parse(m.server.URL)
	assert.NoError(err)

	mirror := &Mirror{
		DownstreamURL: url,
	}
	mirror.Start()

	a := &App{
		Sink:   ms,
		Mirror: mirror,
	}
	a.Start()
	defer a.Stop()

	testFile, err := os.Open("testdata/payload_0.thrift")
	assert.NoError(err)

	data, err := ioutil.ReadAll(testFile)
	assert.NoError(err)
	r := httptest.NewRequest("POST", "/api/v1/spans", bytes.NewReader(data))
	r.Header.Add("Content-Type", "application/x-thrift")
	w := httptest.NewRecorder()
	a.handleSpans(w, r)
	assert.Equal(w.Code, http.StatusAccepted)

	mirror.Stop()

	assert.Equal(len(m.payloads), 1)
	assert.Equal(m.payloads[0], data)
}

type mockDownstream struct {
	server   *httptest.Server
	payloads [][]byte
}

func newMockDownstream() *mockDownstream {
	var payloads [][]byte
	mu := &mockDownstream{
		payloads: payloads,
	}

	mu.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			data, _ := ioutil.ReadAll(r.Body)
			mu.payloads = append(mu.payloads, data)
			w.WriteHeader(http.StatusAccepted)
		}))
	return mu
}
