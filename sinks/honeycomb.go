package sinks

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/zipkinproxy/types"
)

// HoneycombSink implements the Sink interface. It sends spans to the Honeycomb
// API.
type HoneycombSink struct {
	Writekey string
	Dataset  string
	APIHost  string
	// TODO use builder to allow for multiple datasets?
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
	for _, s := range spans {
		ev := libhoney.NewEvent()
		ev.Timestamp = s.Timestamp
		ev.Add(s.CoreSpanMetadata)
		for k, v := range s.BinaryAnnotations {
			ev.AddField(fmt.Sprintf("%s.%s", s.Name, k), v)
		}
		err := ev.Send()
		if err != nil {
			logrus.WithError(err).Info("Error sending libhoney event")
		}
	}
	return nil
}
