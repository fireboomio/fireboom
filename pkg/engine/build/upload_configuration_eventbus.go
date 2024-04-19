package build

import (
	"fireboom-server/pkg/common/models"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
)

func (u *uploadConfiguration) Subscribe() {
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventInsert, func(data any) any {
		return u.buildStorageItem(data.(*models.Storage))
	})
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventUpdate, func(data any) any {
		return u.buildStorageItem(data.(*models.Storage))
	})
}
