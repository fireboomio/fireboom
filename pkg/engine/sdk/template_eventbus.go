package sdk

import (
	"fireboom-server/pkg/common/models"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
)

func (t *templateContext) Subscribe() {
	eventbus.Subscribe(eventbus.ChannelSdk, eventbus.EventInsert, func(data any) any {
		go t.generateTemplate(data.(*models.Sdk))
		return nil
	})
	eventbus.Subscribe(eventbus.ChannelSdk, eventbus.EventUpdate, func(data any) any {
		go t.generateTemplate(data.(*models.Sdk))
		return nil
	})
}
