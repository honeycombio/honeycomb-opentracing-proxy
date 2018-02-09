package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/jaeger/thrift-gen/zipkincore"
)

func convertThriftSpan(ts *zipkincore.Span) *Span {
	s := &Span{
		CoreSpanMetadata: CoreSpanMetadata{
			TraceID: convertID(ts.TraceID),
			Name:    ts.Name,
			ID:      convertID(ts.ID),
			Debug:   ts.Debug,
		},
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
		if ba.Key == "cs" || ba.Key == "sr" {
			// Special case, skip this for now
			// https://github.com/openzipkin/zipkin/blob/master/zipkin/src/main/java/zipkin/Endpoint.java#L35
			continue
		}
		s.BinaryAnnotations[ba.Key] = convertBinaryAnnotationValue(ba)
		if endpoint := ba.Host; endpoint != nil {
			s.HostIPv4 = convertIPv4(endpoint.Ipv4)
			s.ServiceName = endpoint.ServiceName
			s.Port = int(endpoint.Port)
		}
	}
	// TODO: do something with annotations
	return s
}

func convertID(id int64) string {
	return fmt.Sprintf("%x", id) // TODO is this right?
}

func convertIPv4(ip int32) string {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)).String()
}

func convertBinaryAnnotationValue(ba *zipkincore.BinaryAnnotation) interface{} {
	switch ba.AnnotationType {
	case zipkincore.AnnotationType_BOOL:
		return bytes.Compare(ba.Value, []byte{0}) == 1
	case zipkincore.AnnotationType_BYTES:
		return ba.Value
	case zipkincore.AnnotationType_DOUBLE, zipkincore.AnnotationType_I16, zipkincore.AnnotationType_I32, zipkincore.AnnotationType_I64:
		var number interface{}
		binary.Read(bytes.NewReader(ba.Value), binary.BigEndian, number)
		return number
	case zipkincore.AnnotationType_STRING:
		return guessAnnotationType(string(ba.Value))
	}

	return ba.Value
}

// DecodeThrift reads a list of encoded thrift spans from an io.Reader, and
// converts that list to a slice of Spans.
// The implementation is based on jaeger internals, but not exported there.
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
