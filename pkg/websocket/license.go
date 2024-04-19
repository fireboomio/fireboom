// Package websocket
/*
 licences的websocket实现
*/
package websocket

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const licenseChannel configs.WsChannel = "license"

type licenseWarn struct {
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func init() {
	configs.AddCollector(&configs.LogCollector{
		MatchLevel:    []zapcore.Level{zap.ErrorLevel, zap.WarnLevel},
		IdentifyField: consts.LicenseStatusField,
		Handle: func(entry zapcore.Entry, value *zapcore.Field, fieldMap map[string]*zapcore.Field) *configs.WsMsgBody {
			return &configs.WsMsgBody{
				Channel: licenseChannel,
				Event:   configs.PushEvent,
				Data: &licenseWarn{
					Msg:  entry.Message,
					Data: value.Interface,
				},
			}
		},
	})

	configs.WsMsgHandlerMap[licenseChannel] = func(msg *configs.WsMsgBody) any {
		switch msg.Event {
		case configs.PullEvent:
			return configs.LicenseInfoData
		}
		return nil
	}
}
