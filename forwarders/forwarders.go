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
