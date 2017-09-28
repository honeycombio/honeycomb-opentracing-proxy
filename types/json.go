package types

import (
	"encoding/json"
	"io"
	"time"
)

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
	BinaryAnnotations []*BinaryAnnotation `json:"binaryAnnotations"`
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

		// TODO: do something with endpoint value
	}
	// TODO: do something with annotations
	return s
}

func convertTimestamp(tsMicros int64) time.Time {
	return time.Unix(tsMicros/1000000, (tsMicros%1000000)*1000).UTC()
}

//

type Annotation struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Host      *Endpoint `json:"host,omitempty"`
}

type BinaryAnnotation struct {
	Key            string         `json:"key"`
	Value          string         `json:"value"`
	AnnotationType AnnotationType `json:"annotationType"`
	Endpoint       *Endpoint      `json:"host,omitempty"`
}

type Endpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}

type AnnotationType int64

const (
	AnnotationType_BOOL   AnnotationType = 0
	AnnotationType_BYTES  AnnotationType = 1
	AnnotationType_I16    AnnotationType = 2
	AnnotationType_I32    AnnotationType = 3
	AnnotationType_I64    AnnotationType = 4
	AnnotationType_DOUBLE AnnotationType = 5
	AnnotationType_STRING AnnotationType = 6
)
