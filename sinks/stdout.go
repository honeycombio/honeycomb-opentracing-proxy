package sinks

import (
	"encoding/json"
	"fmt"

	"github.com/honeycombio/zipkinproxy/types"
)

type StdoutSink struct{}

func (s *StdoutSink) Start() error {
	return nil
}

func (s *StdoutSink) Stop() error {
	return nil
}

func (s *StdoutSink) Send(spans []*types.Span) error {
	for _, span := range spans {
		marshalled, _ := json.Marshal(span)
		fmt.Println(string(marshalled))
	}
	return nil
}
