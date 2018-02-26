package sinks

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	libhoney "github.com/honeycombio/libhoney-go"
)

const datasetKey = "honeycomb.dataset"

// HoneycombSink implements the Sink interface. It sends spans to the Honeycomb
// API.
type HoneycombSink struct {
	Writekey string
	Dataset  string
	APIHost  string
}

func (hs *HoneycombSink) Start() error {
	libhoney.Init(libhoney.Config{
		WriteKey: hs.Writekey,
		Dataset:  hs.Dataset,
		APIHost:  hs.APIHost,
	})

	go func() {
		for resp := range libhoney.Responses() {
			fmt.Println("got libhoney response!", resp)
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
		ev := libhoney.NewEvent()
		ev.Timestamp = s.Timestamp
		ev.Add(s.CoreSpanMetadata)
		for k, v := range s.BinaryAnnotations {
			if k == datasetKey {
				// Let clients route spans to different datasets using the
				// `honeycomb.dataset` tag.
				ds, ok := v.(string)
				if !ok {
					logrus.WithField("honeycomb.dataset", v).Error("unexpected type for honeycomb.dataset tag value")
					continue spanLoop
				}
				ev.Dataset = ds
			} else {
				ev.AddField(k, v)
			}
		}
		err := ev.Send()
		if err != nil {
			logrus.WithError(err).Info("Error sending libhoney event")
		}
	}
	return nil
}
