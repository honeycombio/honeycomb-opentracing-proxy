package forwarders

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/zipkinproxy/types"
)

type HoneycombForwarder struct {
	Writekey string
	Dataset  string
	// TODO use builder to allow for multiple datasets?
}

func (hf *HoneycombForwarder) Start() error {
	libhoney.Init(libhoney.Config{
		WriteKey: hf.Writekey,
		Dataset:  hf.Dataset,
	})

	go func() {
		for resp := range libhoney.Responses() {
			fmt.Println("got libhoney response!", resp)
		}
	}()

	return nil
}

func (hf *HoneycombForwarder) Stop() error {
	return nil
}

func (hf *HoneycombForwarder) Forward(spans []*types.Span) error {
	for _, s := range spans {
		ev := libhoney.NewEvent()
		ev.AddField("id", s.ID)
		ev.AddField("name", s.Name)
		ev.AddField("traceId", s.TraceID)
		ev.AddField("parentId", s.ParentID)
		ev.AddField("timestamp", s.Timestamp)
		ev.AddField("debug", s.Debug)
		ev.AddField("durationMs", s.DurationMs)
		for k, v := range s.BinaryAnnotations {
			ev.AddField(fmt.Sprintf("ba.%s", k), v)
		}
		err := ev.Send()
		if err != nil {
			logrus.WithError(err).Info("Error sending libhoney event")
		}
	}
	return nil
}
