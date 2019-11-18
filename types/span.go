package types

import (
	"strconv"
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
	Annotations       []*Annotation          `json:"annotations,omitempty"` // TODO lift annotation struct definition into this file
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

// convertTimestamp turns a Zipkin timestamp (a Unix timestamp in microseconds)
// into a time.Time value.
func convertTimestamp(tsMicros int64) time.Time {
	if tsMicros == 0 {
		return time.Now().UTC()
	}

	return time.Unix(tsMicros/1000000, (tsMicros%1000000)*1000).UTC()
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
	strVal, ok := v.(string)
	if !ok {
		return v
	} else if strVal == "false" {
		return false
	} else if strVal == "true" {
		return true
	} else if intVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
		return intVal
	} else if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
		return floatVal
	}
	return strVal
}
