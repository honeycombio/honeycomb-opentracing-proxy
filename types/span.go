package types

import "time"

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

func convertTimestamp(tsMicros int64) time.Time {
	return time.Unix(tsMicros/1000000, (tsMicros%1000000)*1000).UTC()
}
