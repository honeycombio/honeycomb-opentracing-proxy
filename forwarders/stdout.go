package forwarders

import (
	"encoding/json"
	"fmt"

	"github.com/honeycombio/zipkinproxy/types"
)

type StdoutForwarder struct{}

func (sf *StdoutForwarder) Start() error {
	return nil
}

func (sf *StdoutForwarder) Stop() error {
	return nil
}

func (sf *StdoutForwarder) Forward(spans []*types.Span) error {
	for _, s := range spans {
		marshalled, _ := json.Marshal(s)
		fmt.Println(string(marshalled))
	}
	return nil
}
