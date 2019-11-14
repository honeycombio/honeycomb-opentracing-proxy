package v1

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/jaeger/thrift-gen/zipkincore"
)

func convertThriftSpan(ts *zipkincore.Span) *types.Span {
	s := &types.Span{
		CoreSpanMetadata: types.CoreSpanMetadata{
			TraceID:      convertID(ts.TraceID),
			TraceIDAsInt: ts.TraceID,
			Name:         ts.Name,
			ID:           convertID(ts.ID),
			Debug:        ts.Debug,
		},
		BinaryAnnotations: make(map[string]interface{}, len(ts.BinaryAnnotations)),
	}
	if ts.ParentID != nil && *ts.ParentID != 0 {
		s.ParentID = convertID(*ts.ParentID)
	}

	if ts.Duration != nil {
		s.DurationMs = float64(*ts.Duration) / 1000
	}

	if ts.Timestamp != nil {
		s.Timestamp = types.ConvertTimestamp(*ts.Timestamp)
	} else {
		s.Timestamp = time.Now().UTC()
	}

	var endpoint *zipkincore.Endpoint
	for _, ba := range ts.BinaryAnnotations {
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
		s.BinaryAnnotations[ba.Key] = convertBinaryAnnotationValue(ba)
		if ba.Host != nil {
			endpoint = ba.Host
		}
	}

	for _, a := range ts.Annotations {
		// TODO: do more with annotations (i.e., point-in-time logs within a span)
		// besides extracting host info.
		if a.Host != nil {
			endpoint = a.Host
		}
	}
	if endpoint != nil {
		s.HostIPv4 = convertIPv4(endpoint.Ipv4)
		s.ServiceName = endpoint.ServiceName
		s.Port = int(endpoint.Port)
	}
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
		return types.GuessAnnotationType(string(ba.Value))
	}

	return ba.Value
}

// DecodeThrift reads a list of encoded thrift spans from an io.Reader, and
// converts that list to a slice of Spans.
// The implementation is based on jaeger internals, but not exported there.
func DecodeThrift(r io.Reader) ([]*types.Span, error) {
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
	var spans []*types.Span
	for i := 0; i < size; i++ {
		zs := &zipkincore.Span{}
		if err = zs.Read(transport); err != nil {
			return nil, err
		}
		spans = append(spans, convertThriftSpan(zs))
	}

	return spans, nil
}
