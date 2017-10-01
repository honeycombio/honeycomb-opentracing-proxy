package types

import (
	"encoding/json"
	"io"
)

// DecodeJSON reads an array of JSON-encoded spans from an io.Reader, and
// converts that array to a slice of Spans.
func DecodeJSON(r io.Reader) ([]*Span, error) {
	var jsonSpans []zipkinJSONSpan
	err := json.NewDecoder(r).Decode(&jsonSpans)
	if err != nil {
		return nil, err
	}
	spans := make([]*Span, len(jsonSpans))
	for i, s := range jsonSpans {
		spans[i] = convertJSONSpan(s)
	}

	return spans, nil
}

type zipkinJSONSpan struct {
	TraceID           string              `json:"traceId"`
	Name              string              `json:"name"`
	ID                string              `json:"id"`
	ParentID          string              `json:"parentId,omitempty"`
	Annotations       []*Annotation       `json:"annotations"`
	BinaryAnnotations []*binaryAnnotation `json:"binaryAnnotations"`
	Debug             bool                `json:"debug,omitempty"`
	Timestamp         int64               `json:"timestamp,omitempty"`
	Duration          int64               `json:"duration,omitempty"`
}

func convertJSONSpan(zs zipkinJSONSpan) *Span {
	s := &Span{
		TraceID:           zs.TraceID,
		Name:              zs.Name,
		ID:                zs.ID,
		ParentID:          zs.ParentID,
		Debug:             zs.Debug,
		Timestamp:         convertTimestamp(zs.Timestamp),
		DurationMs:        float64(zs.Duration) / 1000.,
		BinaryAnnotations: make(map[string]interface{}, len(zs.BinaryAnnotations)),
	}

	for _, ba := range zs.BinaryAnnotations {
		if ba == nil {
			continue
		}
		if ba.Key == "cs" || ba.Key == "sr" {
			// Special case, skip this for now
			// https://github.com/openzipkin/zipkin/blob/master/zipkin/src/main/java/zipkin/Endpoint.java#L35
			continue
		}
		s.BinaryAnnotations[ba.Key] = string(ba.Value)
		if endpoint := ba.Endpoint; endpoint != nil {
			s.HostIPv4 = endpoint.Ipv4
			s.ServiceName = endpoint.ServiceName
			s.Port = endpoint.Port
		}
	}
	// TODO: do something with annotations
	return s
}

type Annotation struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Host      *Endpoint `json:"host,omitempty"`
}

type binaryAnnotation struct {
	Key      string    `json:"key"`
	Value    string    `json:"value"` // TODO: are BinaryAnnotations really always strings in the Zipkin JSON API?
	Endpoint *Endpoint `json:"host,omitempty"`
}

type Endpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}
