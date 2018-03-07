package types

import (
	"encoding/json"
	"io"
)

// DecodeJSON reads an array of JSON-encoded spans from an io.Reader, and
// converts that array to a slice of Spans.
func DecodeJSON(r io.Reader) ([]*Span, error) {
	var jsonSpans []ZipkinJSONSpan
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

type ZipkinJSONSpan struct {
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

func convertJSONSpan(zs ZipkinJSONSpan) *Span {
	s := &Span{
		CoreSpanMetadata: CoreSpanMetadata{
			TraceID:    zs.TraceID,
			Name:       zs.Name,
			ID:         zs.ID,
			ParentID:   zs.ParentID,
			Debug:      zs.Debug,
			DurationMs: float64(zs.Duration) / 1000.,
		},
		Timestamp:         convertTimestamp(zs.Timestamp),
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

type Annotation struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Host      *Endpoint `json:"host,omitempty"`
}

type binaryAnnotation struct {
	Key      string    `json:"key"`
	Value    string    `json:"value"`
	Endpoint *Endpoint `json:"host,omitempty"`
}

type Endpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}
