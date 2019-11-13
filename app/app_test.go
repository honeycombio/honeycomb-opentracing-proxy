package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/honeycombio/honeycomb-opentracing-proxy/sinks"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	v1 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v1"
	v2 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v2"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger/thrift-gen/zipkincore"
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

func TestMissingJSONTimestampHandling_V1(t *testing.T) {
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
				"id":          "34472e70cb669b31"
			}]`

	now := time.Now()
	w := handleGzippedV1(a, []byte(jsonPayload), "application/json")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.WithinDuration(now, mockHoneycomb.Events()[0].Timestamp, 2*time.Second, "Missing timestamp should be populated")
}

func TestMissingJSONTimestampHandling_V2(t *testing.T) {
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
				"id":          "34472e70cb669b31"
			}]`

	now := time.Now()
	w := handleGzippedV2(a, []byte(jsonPayload), "application/json")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.WithinDuration(now, mockHoneycomb.Events()[0].Timestamp, 2*time.Second, "Missing timestamp should be populated")
}

func TestMissingThriftTimestampHandling(t *testing.T) {
	mockHoneycomb := &libhoney.MockOutput{}
	assert := assert.New(t)
	libhoney.Init(libhoney.Config{
		WriteKey: "test",
		Dataset:  "test",
		Output:   mockHoneycomb,
	})
	a := &App{Sink: &sinks.HoneycombSink{}}

	thriftPayload := serializeThriftSpans([]*zipkincore.Span{
		&zipkincore.Span{
			TraceID: 2222,
			ID:      2222,
			Name:    "mySpan",
		},
	})

	now := time.Now().UTC()
	w := handleGzippedV1(a, thriftPayload, "application/x-thrift")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.WithinDuration(now, mockHoneycomb.Events()[0].Timestamp, 10*time.Second, "Empty timestamp should be set to current")
}

// TestThriftDecoding takes a capture of a zipkin thrift payload, and ensures
// that it's decoded and forwarded correctly.
func TestThriftDecoding(t *testing.T) {
	assert := assert.New(t)
	testFile, err := os.Open("testdata/payload_0.thrift")
	assert.NoError(err)
	data, err := ioutil.ReadAll(testFile)
	assert.NoError(err)
	expectedSpans := []types.Span{
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:      "350565b6a90d4c8c",
				TraceIDAsInt: 3820571694088408204,
				Name:         "/api.RetrieverService/Fetch",
				ID:           "3ba1d9a5451f81c4",
				ParentID:     "350565b6a90d4c8c",
				DurationMs:   2.155,
				HostIPv4:     "10.129.211.111",
				ServiceName:  "poodle",
			},
			BinaryAnnotations: map[string]interface{}{
				"component": "gRPC",
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 286440000, time.UTC),
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:      "350565b6a90d4c8c",
				TraceIDAsInt: 3820571694088408204,
				Name:         "persist",
				ID:           "34472e70cb669b31",
				ParentID:     "350565b6a90d4c8c",
				ServiceName:  "poodle",
				HostIPv4:     "10.129.211.111",
				DurationMs:   0.192,
			},
			BinaryAnnotations: map[string]interface{}{
				"lc":             "poodle",
				"responseLength": int64(136),
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 288651000, time.UTC),
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:      "350565b6a90d4c8c",
				TraceIDAsInt: 3820571694088408204,
				Name:         "markAsDone",
				ID:           "2eb1b7009815c803",
				ParentID:     "350565b6a90d4c8c",
				ServiceName:  "poodle",
				HostIPv4:     "10.129.211.111",
				DurationMs:   5.134,
			},
			BinaryAnnotations: map[string]interface{}{
				"lc": "poodle",
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 288847000, time.UTC),
		},
		types.Span{
			CoreSpanMetadata: types.CoreSpanMetadata{
				TraceID:      "350565b6a90d4c8c",
				TraceIDAsInt: 3820571694088408204,
				Name:         "executeQuery",
				ID:           "350565b6a90d4c8c",
				ParentID:     "",
				ServiceName:  "poodle",
				HostIPv4:     "10.129.211.111",
				DurationMs:   9.98,
			},
			BinaryAnnotations: map[string]interface{}{
				"lc":             "poodle",
				"dataset_id":     int64(90),
				"hidden_reason":  int64(0),
				"hostname":       "sea-of-dreams",
				"jaeger.version": "Go-2.8.0",
				"query_hash":     "fca2835dced5d6fafb4eb9dd",
				"query_run_pk":   "7AREu8scycJ",
				"sampler.param":  true,
				"sampler.type":   "const",
				"team_id":        int64(12),
				"user_id":        int64(15),
			},
			Timestamp: time.Date(2017, 9, 28, 20, 15, 17, 284010000, time.UTC),
		},
	}
	// verify with both zipped and ungzipped data
	ms := &MockSink{}
	a := &App{Sink: ms}
	w := handleGzippedV1(a, data, "application/x-thrift")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.Equal(ms.spans[:4], expectedSpans)
	ms = &MockSink{}
	a = &App{Sink: ms}
	w = handleV1(a, data, "application/x-thrift")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.Equal(ms.spans[:4], expectedSpans)
}

func TestThriftRootSpans(t *testing.T) {
	// Test that spans with a *zero* parentID get converted to spans with a nil
	// parentID.
	assert := assert.New(t)
	now := time.Now().UTC().Round(time.Microsecond)
	n := now.UnixNano() / int64(time.Microsecond)
	var zero int64
	body := serializeThriftSpans([]*zipkincore.Span{
		&zipkincore.Span{
			TraceID:   2222,
			ID:        2222,
			ParentID:  &zero,
			Name:      "mySpan",
			Timestamp: &n,
		},
	})
	ms := &MockSink{}
	a := &App{Sink: ms}
	w := handleV1(a, body, "application/x-thrift")
	assert.Equal(w.Code, http.StatusAccepted)
	assert.Equal(types.Span{
		CoreSpanMetadata: types.CoreSpanMetadata{
			TraceID:      "8ae",
			TraceIDAsInt: 2222,
			ID:           "8ae",
			ParentID:     "",
			Name:         "mySpan",
		},
		Timestamp:         now,
		BinaryAnnotations: map[string]interface{}{},
	}, ms.spans[0])
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
	w := handleGzippedV1(a, data, "application/x-thrift")
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
	w := handleGzippedV1(a, data, "application/x-thrift")
	assert.Equal(w.Code, http.StatusAccepted)

}

func TestHoneycombOutput_V1(t *testing.T) {
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
						"endpoint": {
							"ipv4": "10.129.211.111",
							"serviceName": "poodle"
						}
					},
					{
						"key": "responseLength",
						"value": "136",
						"endpoint": {
							"ipv4": "10.129.211.111",
							"serviceName": "poodle"
						}
					}
				],
				"timestamp":  1506629747288651,
				"duration": 192
			}]`

	w := handleGzippedV1(a, []byte(jsonPayload), "application/json")
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
			"responseLength": int64(136),
			"durationMs":     0.192,
		})
	assert.Equal(mockHoneycomb.Events()[0].Dataset, "test")
}

func TestHoneycombOutput_V2(t *testing.T) {
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
				"kind":		   "SERVER",
				"localEndpoint": {
					"serviceName": "poodle"
					"ipv4": "10.129.211.111"
				}
				"tags": {
					"lc": "poodle",
					"responseLength": 136
				}
				"timestamp":  1506629747288651,
				"duration": 192
			}]`

	w := handleGzippedV2(a, []byte(jsonPayload), "application/json")
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
			"responseLength": int64(136),
			"durationMs":     0.192,
			"kind":           "SERVER",
		})
	assert.Equal(mockHoneycomb.Events()[0].Dataset, "test")
}

func TestHoneycombSinkTagHandling_V1(t *testing.T) {
	assert := assert.New(t)
	sampleSpanJSON := `{
		"traceId":     "8fe5ac327a4a4a88",
		"name":        "persist",
		"id":          "bb433fd338b2cecb",
		"parentId":    "",
		"binaryAnnotations": [
			{
				"key": "lc",
				"value": "shepherd",
				"endpoint": {
					"ipv4": "10.129.211.121",
					"serviceName": "shepherd"
				}
			},
			{
				"key": "keyToDrop",
				"value": "secret",
				"endpoint": {
					"ipv4": "10.129.211.121",
					"serviceName": "shepherd"
				}
			},
			{
				"key": "honeycomb.dataset",
				"value": "write-traces",
				"endpoint": {
					"ipv4": "10.129.211.121",
					"serviceName": "shepherd"
				}
			},
			{
				"key": "honeycomb.samplerate",
				"value": "22",
				"endpoint": {
					"ipv4": "10.129.211.121",
					"serviceName": "shepherd"
				}
			}
		],
		"timestamp":  1506629747288651,
		"duration": 222
	}`

	var sampleSpan v1.ZipkinJSONSpan
	err := json.Unmarshal([]byte(sampleSpanJSON), &sampleSpan)
	assert.NoError(err)

	sink := &sinks.HoneycombSink{DropFields: []string{"keyToDrop"}}
	sink.Start()

	mockHoneycomb := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "test",
		Dataset:  "test",
		Output:   mockHoneycomb,
	})

	a := &App{Sink: sink}

	payload, err := json.Marshal([]v1.ZipkinJSONSpan{sampleSpan})
	assert.NoError(err)
	w := handleGzippedV1(a, payload, "application/json")
	assert.Equal(w.Code, http.StatusAccepted)

	assert.Equal(mockHoneycomb.Events()[0].Dataset, "write-traces")
	assert.Equal(mockHoneycomb.Events()[0].SampleRate, uint(22))
	assert.Equal(mockHoneycomb.Events()[0].Fields(),
		map[string]interface{}{
			"id":          "bb433fd338b2cecb",
			"traceId":     "8fe5ac327a4a4a88",
			"name":        "persist",
			"hostIPv4":    "10.129.211.121",
			"serviceName": "shepherd",
			"durationMs":  0.222,
			"lc":          "shepherd",
		})

	sampleSpan.BinaryAnnotations[3].Value = "-22"
	payload, err = json.Marshal([]v1.ZipkinJSONSpan{sampleSpan})
	assert.NoError(err)
	w = handleGzippedV1(a, payload, "application/json")
	assert.Equal(w.Code, http.StatusAccepted)
	libhoney.Close()
	assert.Equal(mockHoneycomb.Events()[1].Dataset, "write-traces")
	assert.Equal(mockHoneycomb.Events()[1].SampleRate, uint(1))
}

func TestHoneycombSinkTagHandling_V2(t *testing.T) {
	assert := assert.New(t)
	sampleSpanJSON := `{
		"traceId":     "8fe5ac327a4a4a88",
		"name":        "persist",
		"id":          "bb433fd338b2cecb",
		"parentId":    "",
		"localEndpoint": {
			"serviceName": "shepherd"
			"ipv4": "10.129.211.121"
		}
		"tags": {
			"lc": "shepherd",
			"keyToDrop": "secret",
			"honeycomb.dataset": "write-traces",
			"honeycomb.samplerate": 22
		}
		"timestamp":  1506629747288651,
		"duration": 222
	}`
	var sampleSpan v2.ZipkinJSONSpan
	err := json.Unmarshal([]byte(sampleSpanJSON), &sampleSpan)
	assert.NoError(err)

	sink := &sinks.HoneycombSink{DropFields: []string{"keyToDrop"}}
	sink.Start()

	mockHoneycomb := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "test",
		Dataset:  "test",
		Output:   mockHoneycomb,
	})

	a := &App{Sink: sink}

	payload, err := json.Marshal([]v2.ZipkinJSONSpan{sampleSpan})
	assert.NoError(err)
	w := handleGzippedV2(a, payload, "application/json")
	assert.Equal(w.Code, http.StatusAccepted)

	assert.Equal(mockHoneycomb.Events()[0].Dataset, "write-traces")
	assert.Equal(mockHoneycomb.Events()[0].SampleRate, uint(22))
	assert.Equal(mockHoneycomb.Events()[0].Fields(),
		map[string]interface{}{
			"id":          "bb433fd338b2cecb",
			"traceId":     "8fe5ac327a4a4a88",
			"name":        "persist",
			"hostIPv4":    "10.129.211.121",
			"serviceName": "shepherd",
			"durationMs":  0.222,
			"lc":          "shepherd",
		})

	//sampleSpan.Tags = "-22" I think I don't need this
	payload, err = json.Marshal([]v2.ZipkinJSONSpan{sampleSpan})
	assert.NoError(err)
	w = handleGzippedV2(a, payload, "application/json")
	assert.Equal(w.Code, http.StatusAccepted)
	libhoney.Close()
	assert.Equal(mockHoneycomb.Events()[1].Dataset, "write-traces")
	assert.Equal(mockHoneycomb.Events()[1].SampleRate, uint(1))
}

// Test that spans are sampled on a per-trace basis
func TestSampling(t *testing.T) {
	assert := assert.New(t)

	mockHoneycomb := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "test",
		Dataset:  "test",
		Output:   mockHoneycomb,
	})

	downstream := newMockDownstream()
	defer downstream.server.Close()
	url, err := url.Parse(downstream.server.URL)
	assert.NoError(err)
	mirror := &Mirror{
		DownstreamURL: url,
	}
	mirror.Start()

	a := &App{
		Sink:   &sinks.HoneycombSink{SampleRate: 10},
		Mirror: mirror,
	}

	// Construct 30 traces of 10 spans each.
	for spanID := int64(0); spanID < 10; spanID++ {
		for traceID := int64(0); traceID < 30; traceID++ {
			span := &zipkincore.Span{
				TraceID: traceID,
				ID:      spanID,
				Name:    "someSpan",
			}
			body := serializeThriftSpans([]*zipkincore.Span{span})
			w := handleGzippedV1(a, body, "application/x-thrift")
			assert.Equal(w.Code, http.StatusAccepted)
		}
	}

	mirror.Stop()

	// Check that we sent 30 out of 300 spans to Honeycomb, and all 300 out of
	// 300 spans to the Zipkin mirror.
	assert.Equal(len(mockHoneycomb.Events()), 30)
	assert.Equal(len(downstream.payloads), 300)

	sampledSpanCounts := make(map[string]int)
	for _, ev := range mockHoneycomb.Events() {
		sampledSpanCounts[ev.Fields()["traceId"].(string)]++
	}

	// Check that we sent 3 out of 30 traces, and that each trace has a
	// complete set of 10 spans.
	assert.Equal(len(sampledSpanCounts), 3)
	for _, v := range sampledSpanCounts {
		assert.Equal(v, 10)
	}
}

type mockDownstream struct {
	server   *httptest.Server
	payloads []payload

	sync.Mutex
}

func newMockDownstream() *mockDownstream {
	var payloads []payload
	m := &mockDownstream{
		payloads: payloads,
	}

	m.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			data, _ := ioutil.ReadAll(r.Body)
			m.Lock()
			m.payloads = append(m.payloads, payload{ContentType: r.Header.Get("Content-Type"), Body: data})
			m.Unlock()
			w.WriteHeader(http.StatusAccepted)
		}))
	return m
}

func handleV1(a *App, payload []byte, contentType string) *httptest.ResponseRecorder {
	return handle(a, payload, "/api/v1/spans", contentType)
}

func handleV2(a *App, payload []byte, contentType string) *httptest.ResponseRecorder {
	return handle(a, payload, "/api/v2/spans", contentType)
}

func handleGzippedV1(a *App, payload []byte, contentType string) *httptest.ResponseRecorder {
	return handleGzipped(a, payload, "/api/v1/spans", contentType)
}

func handleGzippedV2(a *App, payload []byte, contentType string) *httptest.ResponseRecorder {
	return handleGzipped(a, payload, "/api/v2/spans", contentType)
}

func handle(a *App, payload []byte, path, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, bytes.NewReader(payload))
	r.Header.Add("Content-Type", contentType)
	w := httptest.NewRecorder()
	a.handleSpans(w, r)
	return w
}

func handleGzipped(a *App, payload []byte, path, contentType string) *httptest.ResponseRecorder {
	var compressedPayload bytes.Buffer
	zw := gzip.NewWriter(&compressedPayload)
	zw.Write(payload)
	zw.Close()

	r := httptest.NewRequest("POST", path, &compressedPayload)
	r.Header.Add("Content-Encoding", "gzip")
	r.Header.Add("Content-Type", contentType)
	w := httptest.NewRecorder()
	ungzipWrap(a.handleSpans)(w, r)
	return w
}

func serializeThriftSpans(spans []*zipkincore.Span) []byte {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	p.WriteListBegin(thrift.STRUCT, len(spans))
	for _, s := range spans {
		s.Write(p)
	}
	p.WriteListEnd()
	return t.Buffer.Bytes()
}
