package v2

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type ZipkinV2JSONSpan struct {
	TraceID        string                 `json:"traceId"`
	Name           string                 `json:"name"`
	ID             string                 `json:"id"`
	ParentID       string                 `json:"parentId,omitempty"`
	Kind           string                 `json:"kind,omitempty"`
	LocalEndpoint  LocalEndpoint          `json:"localEndpoint,omitempty"`
	RemoteEndpoint RemoteEndpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []*AnnotationV2        `json:"annotation,omitemptys"`
	Tags           map[string]interface{} `json:"tags,omitempty"`
	Debug          bool                   `json:"debug,omitempty"`
	Timestamp      int64                  `json:"timestamp,omitempty"`
	Duration       int64                  `json:"duration,omitempty"`
}

type AnnotationV2 struct {
	Timestamp int64           `json:"timestamp"`
	Value     string          `json:"value"`
	Host      *RemoteEndpoint `json:"endpoint,omitempty"`
}

type LocalEndpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}

type RemoteEndpoint struct {
	Ipv4 string `json:"ipv4"`
	Port int    `json:"port"`
}

// DecodeJSONV2 reads an array of JSON-encoded spans from an io.Reader, and
// converts that array to a slice of Spans.
func DecodeJSONV2(r io.Reader) ([]*types.Span, error) {
	var jsonSpans []ZipkinV2JSONSpan
	err := json.NewDecoder(r).Decode(&jsonSpans)
	if err != nil {
		return nil, err
	}
	spans := make([]*types.Span, len(jsonSpans))
	for i, s := range jsonSpans {
		spans[i] = convertV2JSONSpan(s)
	}

	return spans, nil
}

func convertV2JSONSpan(zs ZipkinV2JSONSpan) *types.Span {
	traceIDAsInt, _ := strconv.ParseInt(zs.TraceID, 16, 64)
	s := &types.Span{
		CoreSpanMetadata: types.CoreSpanMetadata{
			TraceID:      zs.TraceID,
			TraceIDAsInt: traceIDAsInt,
			Name:         zs.Name,
			ID:           zs.ID,
			ParentID:     zs.ParentID,
			Debug:        zs.Debug,
			DurationMs:   float64(zs.Duration) / 1000.,
		},
		Timestamp:         types.ConvertTimestamp(zs.Timestamp),
		BinaryAnnotations: zs.Tags,
	}

	if (zs.LocalEndpoint != LocalEndpoint{}) {
		s.HostIPv4 = zs.LocalEndpoint.Ipv4
		s.ServiceName = zs.LocalEndpoint.ServiceName
		s.Port = zs.LocalEndpoint.Port
	}

	s.BinaryAnnotations["kind"] = zs.Kind

	// TODO: do something with annotations
	return s
}
