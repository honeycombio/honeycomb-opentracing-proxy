package sinks

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/zipkinproxy/types"
)

// StdoutSink implements the Sink interface. It writes span data to stdout.
type StdoutSink struct{}

func (s *StdoutSink) Start() error {
	return nil
}

func (s *StdoutSink) Stop() error {
	return nil
}

func (s *StdoutSink) Send(spans []*types.Span) error {
	for _, span := range spans {
		marshalled, err := json.Marshal(span)
		if err != nil {
			logrus.WithError(err).Error("Error marshalling spans!")
			continue
		}
		fmt.Println(string(marshalled))
	}
	return nil
}
