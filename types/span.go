package types

import (
	"time"
)

// Span represents a Zipkin span in a format more useful for consumption in
// Honeycomb.
// - BinaryAnnotations are turned into a key: value map.
// - Endpoint values in BinaryAnnotations are lifted into top-level
//   HostIPv4/Port/ServiceName values on the span.
// - Timestamp and Duration values are turned into time.Time and millisecond
//   values, respectively.
type Span struct {
	CoreSpanMetadata
	Annotations       []*SpanAnnotation      `json:"annotations,omitempty"` // TODO lift annotation struct definition into this file
	BinaryAnnotations map[string]interface{} `json:"binaryAnnotations,omitempty"`
	Timestamp         time.Time              `json:"timestamp,omitempty"`
}

// CoreSpanMetadata is the subset of span data that can be added directly into
// a libhoney event. Annotations, BinaryAnnotations and Timestamp need special
// handling.
type CoreSpanMetadata struct {
	TraceID      string  `json:"traceId"`
	TraceIDAsInt int64   `json:"-"` // Zipkin trace ID as integer; not added to events, but used for sampling decisions
	Name         string  `json:"name"`
	ID           string  `json:"id"`
	ParentID     string  `json:"parentId,omitempty"`
	ServiceName  string  `json:"serviceName,omitempty"`
	HostIPv4     string  `json:"hostIPv4,omitempty"`
	Port         int     `json:"port,omitempty"`
	Debug        bool    `json:"debug,omitempty"`
	DurationMs   float64 `json:"durationMs,omitempty"`
}

type SpanAnnotation struct {
	Timestamp int64         `json:"timestamp"`
	Value     string        `json:"value"`
	Host      *SpanEndpoint `json:"endpoint,omitempty"`
}

type SpanEndpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
}

// ConvertTimestamp turns a Zipkin timestamp (a Unix timestamp in microseconds)
// into a time.Time value.
func ConvertTimestamp(tsMicros int64) time.Time {
	if tsMicros == 0 {
		return time.Now().UTC()
	}

	return time.Unix(tsMicros/1000000, (tsMicros%1000000)*1000).UTC()
}
