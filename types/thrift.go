package types

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/jaeger/thrift-gen/zipkincore"
)

func convertThriftSpan(ts *zipkincore.Span) *Span {
	s := &Span{
		TraceID:           convertID(ts.TraceID),
		Name:              ts.Name,
		ID:                convertID(ts.ID),
		Debug:             ts.Debug,
		BinaryAnnotations: make(map[string]interface{}, len(ts.BinaryAnnotations)),
	}
	if ts.ParentID != nil {
		s.ParentID = convertID(*ts.ParentID)
	}

	if ts.Duration != nil {
		s.DurationMs = float64(*ts.Duration) / 1000
	}

	if ts.Timestamp != nil {
		s.Timestamp = convertTimestamp(*ts.Timestamp)
	}

	for _, ba := range ts.BinaryAnnotations {
		s.BinaryAnnotations[ba.Key] = string(ba.Value)
		// TODO: do something with endpoint value
	}
	// TODO: do something with annotations
	return s
}

func convertID(id int64) string {
	return fmt.Sprintf("%x", id) // TODO is this right?
}

// from jaeger internals but not exported there
func DecodeThrift(r io.Reader) ([]*Span, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	buffer := thrift.NewTMemoryBuffer()
	buffer.Write(body)

	transport := thrift.NewTBinaryProtocolTransport(buffer)
	_, size, err := transport.ReadListBegin() // Ignore the returned element type
	if err != nil {
		return nil, err
	}

	// We don't depend on the size returned by ReadListBegin to preallocate the array because it
	// sometimes returns a nil error on bad input and provides an unreasonably large int for size
	var spans []*Span
	for i := 0; i < size; i++ {
		zs := &zipkincore.Span{}
		if err = zs.Read(transport); err != nil {
			return nil, err
		}
		spans = append(spans, convertThriftSpan(zs))
	}

	return spans, nil
}
