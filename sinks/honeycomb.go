package sinks

import (
	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	libhoney "github.com/honeycombio/libhoney-go"
)

const datasetKey = "honeycomb.dataset"
const sampleRateKey = "honeycomb.samplerate"

// HoneycombSink implements the Sink interface. It sends spans to the Honeycomb
// API.
type HoneycombSink struct {
	Writekey   string
	Dataset    string
	APIHost    string
	SampleRate uint
	DropFields []string

	dropFieldsMap map[string]struct{}
}

func (hs *HoneycombSink) Start() error {
	hs.dropFieldsMap = make(map[string]struct{})
	for _, v := range hs.DropFields {
		hs.dropFieldsMap[v] = struct{}{}
	}
	libhoney.Init(libhoney.Config{
		WriteKey: hs.Writekey,
		Dataset:  hs.Dataset,
		APIHost:  hs.APIHost,
	})

	go func() {
		for resp := range libhoney.Responses() {
			if resp.Err != nil || resp.StatusCode != 202 {
				logrus.WithFields(logrus.Fields{
					"error":  resp.Err,
					"status": resp.StatusCode,
					"body":   string(resp.Body),
				}).Error("Error sending span to Honeycomb")
			} else {
				spanId, _ := resp.Metadata.(string)
				logrus.WithField("spanId", spanId).Debug("Successfully sent span to Honeycomb")
			}
		}
	}()

	return nil
}

func (hs *HoneycombSink) Stop() error {
	return nil
}

func (hs *HoneycombSink) Send(spans []*types.Span) error {
spanLoop:
	for _, s := range spans {
		if hs.SampleRate > 1 && s.TraceIDAsInt%int64(hs.SampleRate) != 0 {
			continue
		}
		ev := libhoney.NewEvent()
		ev.Timestamp = s.Timestamp
		ev.Add(s.CoreSpanMetadata)
		ev.Metadata = s.ID
		for k, v := range s.BinaryAnnotations {
			if _, ok := hs.dropFieldsMap[k]; ok {
				// drop this tag instead of sending its data to Honeycomb
				continue
			}

			switch k {
			case datasetKey:
				// Let clients route spans to different datasets using the
				// `honeycomb.dataset` tag.
				if ds, ok := v.(string); ok {
					ev.Dataset = ds
				} else {
					logrus.WithField("honeycomb.dataset", v).Error(
						"unexpected type for honeycomb.dataset tag value")
					continue spanLoop
				}
			case sampleRateKey:
				if sampleRate, ok := extractUint(v); ok {
					ev.SampleRate = sampleRate
				} else {
					logrus.WithField(sampleRateKey, v).Error(
						"unexpected value for honeycomb.samplerate tag")
					// Let's not drop on invalid sample rate though
				}
			default:
				ev.AddField(k, v)
			}
		}
		err := ev.SendPresampled()
		if err != nil {
			logrus.WithError(err).Info("Error sending libhoney event")
		}
	}
	return nil
}

// Extract an unsigned int from an interface{} type if possible, so that we can
// get a samplerate value from a span tag.
// This implementation relies on us having converted annotation values of string
// type to int64 or float64 values in guessAnnotationType().
func extractUint(v interface{}) (uint, bool) {
	switch val := v.(type) {
	case int64:
		if val < 0 {
			return 0, false
		}
		return uint(val), true
	case float64:
		if val < 0 {
			return 0, false
		}
		return uint(val), true
	default:
		return 0, false
	}
}
