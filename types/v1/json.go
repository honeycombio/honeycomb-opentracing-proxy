package v1

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type ZipkinJSONSpan struct {
	TraceID           string              `json:"traceId"`
	Name              string              `json:"name"`
	ID                string              `json:"id"`
	ParentID          string              `json:"parentId,omitempty"`
	Annotations       []*AnnotationV1     `json:"annotations"`
	BinaryAnnotations []*binaryAnnotation `json:"binaryAnnotations"`
	Debug             bool                `json:"debug,omitempty"`
	Timestamp         int64               `json:"timestamp,omitempty"`
	Duration          int64               `json:"duration,omitempty"`
}

type AnnotationV1 struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Host      *Endpoint `json:"endpoint,omitempty"`
}

type binaryAnnotation struct {
	Key      string      `json:"key"`
	Value    interface{} `json:"value"`
	Endpoint *Endpoint   `json:"endpoint,omitempty"`
}

type Endpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
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
			DurationMs:   float64(zs.Duration) / 1000.,
		},
		Timestamp:         types.ConvertTimestamp(zs.Timestamp),
		BinaryAnnotations: make(map[string]interface{}, len(zs.BinaryAnnotations)),
	}

	var endpoint *Endpoint
	for _, ba := range zs.BinaryAnnotations {
		if ba == nil {
			continue
		}
		if ba.Key == "ca" || ba.Key == "sa" {
			// BinaryAnnotations with key "ca" (client addr) or "sa" (server addr)
			// are special: the endpoint value for those is the address of the
			// *remote* source or destination of an RPC, rather than the local
			// hostname. See
			// https://github.com/openzipkin/zipkin/blob/c7b341b9b421e7a57c/zipkin/src/main/java/zipkin/Endpoint.java#L35
			// So for those, we don't want to lift the endpoint into the span's
			// own hostIPv4/ServiceName/etc. fields. Simply skip those for now.
			continue
		}
		if ba.Endpoint != nil {
			endpoint = ba.Endpoint
		}
		s.BinaryAnnotations[ba.Key] = guessAnnotationType(ba.Value)
	}
	for _, a := range zs.Annotations {
		// TODO: do more with annotations (i.e., point-in-time logs within a span)
		// besides extracting host info.
		if a.Host != nil {
			endpoint = a.Host
		}
	}
	if endpoint != nil {
		s.HostIPv4 = endpoint.Ipv4
		s.ServiceName = endpoint.ServiceName
		s.Port = endpoint.Port
	}
	// TODO: do something with annotations
	return s
}

// guessAnnotationType takes a value and, if it is a string, turns it into a bool,
// int64 or float64 value when possible. This is a workaround for the fact that
// Zipkin v1 BinaryAnnotation values are always transmitted as strings.
// (See e.g. the Zipkin API spec here:
// https://github.com/openzipkin/zipkin-api/blob/72280f3/zipkin-api.yaml#L235-L245)
//
// However it considers the possibility that the value is not a string in case the
// BinaryAnnotation does not implement the Zipkin API v1 spec. In this case it
// will just return the same value, without modifying it. See this issue
// for such an example:
// https://github.com/honeycombio/honeycomb-opentracing-proxy/issues/37
func guessAnnotationType(v interface{}) interface{} {
	switch v.(type) {
	default:
		return v
	case string:
		if v.(string) == "false" {
			return false
		} else if v.(string) == "true" {
			return true
		} else if intVal, err := strconv.ParseInt(v.(string), 10, 64); err == nil {
			return intVal
		} else if floatVal, err := strconv.ParseFloat(v.(string), 64); err == nil {
			return floatVal
		}
	}

	return v
}
