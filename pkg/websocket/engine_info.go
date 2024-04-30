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
	"golang.org/x/exp/slices"
	"sync"
	"time"
)

const engineChannel configs.WsChannel = "engine"

type onStartedHook struct {
	hook      func()
	order     int
	goroutine bool
}

var (
	onFirstStartedOnce  = sync.Once{}
	onEveryStartedMutex = sync.Mutex{}
	onFirstStartedHooks []onStartedHook
	onEveryStartedHooks []onStartedHook
)

func executeHooks(hooks []onStartedHook) {
	for _, hook := range hooks {
		if hook.goroutine {
			go hook.hook()
		} else {
			hook.hook()
		}
	}
}

func AddOnFirstStartedHook(hook func(), order int, goroutine ...bool) {
	onFirstStartedHooks = append(onFirstStartedHooks, onStartedHook{
		hook:      hook,
		order:     order,
		goroutine: len(goroutine) > 0 && goroutine[0],
	})
}

func AddOnEveryStartedHook(hook func(), goroutine ...bool) {
	onEveryStartedMutex.Lock()
	defer onEveryStartedMutex.Unlock()
	onEveryStartedHooks = append(onEveryStartedHooks, onStartedHook{
		hook:      hook,
		goroutine: len(goroutine) > 0 && goroutine[0],
	})
}

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
				onFirstStartedOnce.Do(func() {
					slices.SortFunc(onFirstStartedHooks, func(a, b onStartedHook) bool { return a.order < b.order })
					executeHooks(onFirstStartedHooks)
				})
				onEveryStartedMutex.Lock()
				defer onEveryStartedMutex.Unlock()
				executeHooks(onEveryStartedHooks)
				onEveryStartedHooks = onEveryStartedHooks[:0]
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
