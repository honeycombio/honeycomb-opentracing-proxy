package forwarders

import (
	"github.com/facebookgo/startstop"
	"github.com/honeycombio/zipkinproxy/types"
)

type Forwarder interface {
	Forward([]*types.Span) error
	startstop.Starter
	startstop.Stopper
}

type CompositeForwarder struct {
	forwarders []Forwarder
}

func (cf *CompositeForwarder) Add(f Forwarder) {
	cf.forwarders = append(cf.forwarders, f)
}

func (cf *CompositeForwarder) Forward(spans []*types.Span) error {
	for _, f := range cf.forwarders {
		f.Forward(spans)
	}
	return nil
}

func (cf *CompositeForwarder) Start() error {
	for _, f := range cf.forwarders {
		if err := f.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (cf *CompositeForwarder) Stop() error {
	for _, f := range cf.forwarders {
		if err := f.Stop(); err != nil {
			return err
		}
	}
	return nil
}
