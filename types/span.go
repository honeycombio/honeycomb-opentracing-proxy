package types

import "time"

// Span represents a Zipkin span in a format more useful for consumption in
// Honeycomb.
// - BinaryAnnotations are turned into a key: value map.
// - Endpoint values in BinaryAnnotations are lifted into top-level
//   HostIPv4/Port/ServiceName values on the span.
// - Timestamp and Duration values are turned into time.Time and millisecond
//   values, respectively.
type Span struct {
	TraceID           string                 `json:"traceId"`
	Name              string                 `json:"name"`
	ID                string                 `json:"id"`
	ParentID          string                 `json:"parentId,omitempty"`
	ServiceName       string                 `json:"serviceName,omitempty"`
	HostIPv4          string                 `json:"hostIPv4,omitempty"`
	Port              int                    `json:"port,omitempty"`
	Annotations       []*Annotation          `json:"annotations,omitempty"` // TODO lift annotation struct definition into this file
	BinaryAnnotations map[string]interface{} `json:"binaryAnnotations,omitempty"`
	Debug             bool                   `json:"debug,omitempty"`
	Timestamp         time.Time              `json:"timestamp,omitempty"`
	DurationMs        float64                `json:"duration,omitempty"`
}

// convertTimestamp turns a Zipkin timestamp (a Unix timestamp in microseconds)
// into a time.Time value.
func convertTimestamp(tsMicros int64) time.Time {
	return time.Unix(tsMicros/1000000, (tsMicros%1000000)*1000).UTC()
}
