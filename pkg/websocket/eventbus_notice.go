// Package websocket
/*
 数据变更websocket实现
*/
package websocket

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
)

func init() {
	utils.RegisterInitMethod(40, func() {
		if utils.InvokeFunctionLimit(consts.LicenseIncrementBuild) {
			return
		}

		eventbus.Notice(func(channel eventbus.Channel, event eventbus.Event, data any) {
			configs.WebsocketInstance.WriteWsMsgBodyForAll(&configs.WsMsgBody{
				Channel: configs.WsChannel(channel),
				Event:   configs.WsEvent(event),
				Data:    data,
			})
		}, eventbus.EventInsert, eventbus.EventBatchInsert, eventbus.EventUpdate, eventbus.EventBatchUpdate, eventbus.EventDelete, eventbus.EventBatchDelete)
	})
}
