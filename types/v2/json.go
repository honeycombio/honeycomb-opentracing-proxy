package v2

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type ZipkinJSONSpan struct {
	TraceID        string                 `json:"traceId"`
	Name           string                 `json:"name"`
	ID             string                 `json:"id"`
	ParentID       string                 `json:"parentId,omitempty"`
	Kind           string                 `json:"kind,omitempty"`
	LocalEndpoint  localEndpoint          `json:"localEndpoint,omitempty"`
	RemoteEndpoint remoteEndpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []*annotation          `json:"annotation,omitempty"`
	Tags           map[string]interface{} `json:"tags,omitempty"`
	Debug          bool                   `json:"debug,omitempty"`
	Timestamp      int64                  `json:"timestamp,omitempty"`
	Duration       int64                  `json:"duration,omitempty"`
}

type annotation struct {
	Timestamp int64           `json:"timestamp"`
	Value     string          `json:"value"`
	Host      *remoteEndpoint `json:"endpoint,omitempty"`
}

type localEndpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}

type remoteEndpoint struct {
	Ipv4 string `json:"ipv4"`
	Port int    `json:"port"`
}

// DecodeJSON reads an array of JSON-encoded spans from an io.Reader, and
// converts that array to a slice of Spans.
func DecodeJSON(r io.Reader) ([]*types.Span, error) {
	var jsonSpans []ZipkinJSONSpan
	err := json.NewDecoder(r).Decode(&jsonSpans)
	if err != nil {
		return nil, err
	}
	spans := make([]*types.Span, len(jsonSpans))
	for i, s := range jsonSpans {
		spans[i] = convertJSONSpan(s)
	}

	return spans, nil
}

func convertJSONSpan(zs ZipkinJSONSpan) *types.Span {
	traceIDAsInt, _ := strconv.ParseInt(zs.TraceID, 16, 64)
	s := &types.Span{
		CoreSpanMetadata: types.CoreSpanMetadata{
			TraceID:      zs.TraceID,
			TraceIDAsInt: traceIDAsInt,
			Name:         zs.Name,
			ID:           zs.ID,
			ParentID:     zs.ParentID,
			Debug:        zs.Debug,
			DurationMs:   float64(zs.Duration) / 1000.0,
		},
		Timestamp: types.ConvertTimestamp(zs.Timestamp),

		// this is needed to allocate the memory for BinaryAnnotations
		// simply doing `BinaryAnnotations: zs.Tags` might cause a null pointer
		// in case zs.Tags is null
		BinaryAnnotations: make(map[string]interface{}, len(zs.Tags)),
	}

	for k, v := range zs.Tags {
		s.BinaryAnnotations[k] = v
	}

	s.BinaryAnnotations["kind"] = zs.Kind

	if (zs.LocalEndpoint != localEndpoint{}) {
		s.HostIPv4 = zs.LocalEndpoint.Ipv4
		s.ServiceName = zs.LocalEndpoint.ServiceName
		s.Port = zs.LocalEndpoint.Port
	}

	// TODO: do something with annotations
	return s
}
