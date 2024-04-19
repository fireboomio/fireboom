package build

import (
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

func init() {
	addResolve(5, func() Resolve { return &webhooks{} })
}

type webhooks struct{}

func (w *webhooks) Resolve(builder *Builder) (err error) {
	builder.DefinedApi.Webhooks = make([]*wgpb.WebhookConfiguration, 0)
	return
}
