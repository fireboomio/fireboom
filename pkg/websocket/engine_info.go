// Package websocket
/*
 引擎信息websocket实现
*/
package websocket

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
	"time"
)

const engineChannel configs.WsChannel = "engine"

type engineInfo struct {
	EngineStatus    string    `json:"engineStatus"`              // 引擎状态
	EngineStartTime time.Time `json:"engineStartTime"`           // 启动时间
	GlobalStartTime time.Time `json:"globalStartTime,omitempty"` // 全局启动时间
	FbVersion       string    `json:"fbVersion,omitempty"`       // 版本号
	FbCommit        string    `json:"fbCommit,omitempty"`        // 版本hash值
}

func init() {
	configs.WsMsgHandlerMap[engineChannel] = func(msg *configs.WsMsgBody) any {
		switch msg.Event {
		case configs.PullEvent:
			return &engineInfo{
				GlobalStartTime: utils.GetTimeWithLockViper(consts.GlobalStartTime),
				EngineStartTime: utils.GetTimeWithLockViper(consts.EngineStartTime),
				EngineStatus:    utils.GetStringWithLockViper(consts.EngineStatusField),
				FbCommit:        utils.GetStringWithLockViper(consts.FbCommit),
				FbVersion:       utils.GetStringWithLockViper(consts.FbVersion),
			}
		}
		return nil
	}

	onceReport := sync.Once{}
	configs.AddCollector(&configs.LogCollector{
		MatchLevel:    []zapcore.Level{zap.ErrorLevel, zap.InfoLevel},
		IdentifyField: consts.EngineStatusField,
		Handle: func(entry zapcore.Entry, value *zapcore.Field, fieldMap map[string]*zapcore.Field) *configs.WsMsgBody {
			utils.SetWithLockViper(consts.EngineStatusField, value.String)
			switch value.String {
			case utils.GetStringWithLockViper(consts.EngineFirstStatus):
				utils.SetWithLockViper(consts.EnginePrepareTime, entry.Time)
				clearQuestion()
			case consts.EngineStartSucceed:
				utils.SetWithLockViper(consts.EngineStartTime, entry.Time)
				onceReport.Do(func() { go utils.HookReportFunc() })
			}

			return &configs.WsMsgBody{
				Channel: engineChannel,
				Event:   configs.PushEvent,
				Data: &engineInfo{
					EngineStatus:    value.String,
					EngineStartTime: utils.GetTimeWithLockViper(consts.EngineStartTime),
				},
			}
		},
	})
}
