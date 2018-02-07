package sinks

import (
	"github.com/facebookgo/startstop"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

// Sink is the interface for doing something with Spans, e.g. sending them to
// Honeycomb or writing them to stdout.
type Sink interface {
	Send([]*types.Span) error
	startstop.Starter
	startstop.Stopper
}

// CompositeSink is an implementation of Sink that sends spans to each provided
// individual sink.
type CompositeSink struct {
	sinks []Sink
}

func (cs *CompositeSink) Add(s Sink) {
	cs.sinks = append(cs.sinks, s)
}

func (cs *CompositeSink) Send(spans []*types.Span) error {
	for _, s := range cs.sinks {
		s.Send(spans)
	}
	return nil
}

func (cs *CompositeSink) Start() error {
	for _, s := range cs.sinks {
		if err := s.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (cs *CompositeSink) Stop() error {
	for _, s := range cs.sinks {
		if err := s.Stop(); err != nil {
			return err
		}
	}
	return nil
}
