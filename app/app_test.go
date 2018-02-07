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

	"github.com/honeycombio/honeycomb-opentracing-proxy/sinks"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

type MockSink struct {
	spans []types.Span
}

func (ms *MockSink) Send(spans []*types.Span) error {
	for _, span := range spans {
		ms.spans = append(ms.spans, *span)
	}
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
	expectedSpans := []types.Span{
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:    "350565b6a90d4c8c",
				Name:       "/api.RetrieverService/Fetch",
				ID:         "3ba1d9a5451f81c4",
				ParentID:   "350565b6a90d4c8c",
				DurationMs: 2.155,
			},
			BinaryAnnotations: map[string]interface{}{
				"component": "gRPC",
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 286440000, time.UTC),
			// TODO where's the endpoint data here?
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:     "350565b6a90d4c8c",
				Name:        "persist",
				ID:          "34472e70cb669b31",
				ParentID:    "350565b6a90d4c8c",
				ServiceName: "poodle",
				HostIPv4:    "10.129.211.111",
				DurationMs:  0.192,
			},
			BinaryAnnotations: map[string]interface{}{
				"lc":             "poodle",
				"responseLength": "136", // TODO verify :/
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 288651000, time.UTC),
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:     "350565b6a90d4c8c",
				Name:        "markAsDone",
				ID:          "2eb1b7009815c803",
				ParentID:    "350565b6a90d4c8c",
				ServiceName: "poodle",
				HostIPv4:    "10.129.211.111",
				DurationMs:  5.134,
			},
			BinaryAnnotations: map[string]interface{}{
				"lc": "poodle",
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 288847000, time.UTC),
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:     "350565b6a90d4c8c",
				Name:        "executeQuery",
				ID:          "350565b6a90d4c8c",
				ParentID:    "",
				ServiceName: "poodle",
				HostIPv4:    "10.129.211.111",
				DurationMs:  9.98,
			},
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
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 284010000, time.UTC),
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
	assert.Equal(m.payloads[0].Body, data)
	assert.Equal(m.payloads[0].ContentType, "application/x-thrift")
}

// Test that we still forward span data even when the "mirror" (e.g., a real
// Zipkin installation that should also receive the Zipkin data) is
// unavailable.
func TestMirroringWhenDestinationUnavailable(t *testing.T) {
	assert := assert.New(t)
	url, _ := url.Parse("http://localhost:9")
	mirror := &Mirror{DownstreamURL: url}
	mirror.Start()
	defer mirror.Stop()
	a := &App{
		Sink:   &MockSink{},
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

}

func TestHoneycombOutput(t *testing.T) {
	mockHoneycomb := &libhoney.MockOutput{}
	assert := assert.New(t)
	libhoney.Init(libhoney.Config{
		WriteKey: "test",
		Dataset:  "test",
		Output:   mockHoneycomb,
	})
	a := &App{Sink: &sinks.HoneycombSink{}}

	jsonPayload := `[{
			"traceId":     "350565b6a90d4c8c",
			"name":        "persist",
			"id":          "34472e70cb669b31",
			"parentId":    "",
			"binaryAnnotations": [
				{
					"key": "lc",
					"value": "poodle",
					"host": {
						"ipv4": "10.129.211.111",
						"serviceName": "poodle"
					}
				},
				{
					"key": "responseLength",
					"value": "136",
					"host": {
						"ipv4": "10.129.211.111",
						"serviceName": "poodle"
					}
				}
			],
			"timestamp":  1506629747288651,
			"duration": 192
		}]`

	r := httptest.NewRequest("POST", "/api/v1/spans",
		bytes.NewReader([]byte(jsonPayload)))
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.handleSpans(w, r)
	assert.Equal(w.Code, http.StatusAccepted)
	assert.Equal(len(mockHoneycomb.Events()), 1)
	assert.Equal(mockHoneycomb.Events()[0].Fields(),
		map[string]interface{}{
			"traceId":        "350565b6a90d4c8c",
			"name":           "persist",
			"id":             "34472e70cb669b31",
			"serviceName":    "poodle",
			"hostIPv4":       "10.129.211.111",
			"lc":             "poodle",
			"responseLength": "136",
			"durationMs":     0.192,
		})
}

type mockDownstream struct {
	server   *httptest.Server
	payloads []payload
}

func newMockDownstream() *mockDownstream {
	var payloads []payload
	mu := &mockDownstream{
		payloads: payloads,
	}

	mu.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			data, _ := ioutil.ReadAll(r.Body)
			mu.payloads = append(mu.payloads, payload{ContentType: r.Header.Get("Content-Type"), Body: data})
			w.WriteHeader(http.StatusAccepted)
		}))
	return mu
}
