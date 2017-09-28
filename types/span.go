package types

import "time"

type Span struct {
	TraceID           string                 `json:"traceId"`
	Name              string                 `json:"name"`
	ID                string                 `json:"id"`
	ParentID          string                 `json:"parentId,omitempty"`
	Annotations       []*Annotation          `json:"annotations"`
	BinaryAnnotations map[string]interface{} `json:"binaryAnnotations"`
	Debug             bool                   `json:"debug,omitempty"`
	Timestamp         time.Time              `json:"timestamp,omitempty"`
	DurationMs        float64                `json:"duration,omitempty"`
}
