// Package websocket
/*
 问题收集的websocket实现
 结合日志收集和hooked中间件实现
*/
package websocket

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/utils"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	questionChannel configs.WsChannel = "question"
	questionField                     = "error"
)

var questions utils.SyncMap[*question, bool]

type question struct {
	Level string         `json:"level"`
	Model string         `json:"model"`
	Name  string         `json:"name"`
	Msg   string         `json:"msg"`
	Extra map[string]any `json:"extra"`
}

func init() {
	configs.WsMsgHandlerMap[questionChannel] = func(msg *configs.WsMsgBody) any {
		switch msg.Event {
		case configs.PullEvent:
			return questions.Keys()
		}
		return nil
	}

	configs.AddFileLoaderQuestionCollector = func(modelName string, extraFunc func(string) map[string]any) {
		configs.AddCollector(&configs.LogCollector{
			MatchLevel:    []zapcore.Level{zap.ErrorLevel, zap.WarnLevel},
			IdentifyField: modelName,
			Handle: func(entry zapcore.Entry, value *zapcore.Field, fieldMap map[string]*zapcore.Field) *configs.WsMsgBody {
				msg := entry.Message
				if errorField, ok := fieldMap[questionField]; ok {
					msg += ": " + cast.ToString(errorField.Interface)
				}

				qs := &question{Level: entry.Level.String(), Model: modelName, Name: value.String, Msg: msg}
				if extraFunc != nil {
					qs.Extra = extraFunc(value.String)
				}
				appendQuestion(qs)
				return &configs.WsMsgBody{
					Channel: questionChannel,
					Event:   configs.PushEvent,
					Data:    qs,
				}
			},
		})
	}
}

func appendQuestion(qs *question) {
	questions.Store(qs, true)
}

func clearQuestion() {
	questions.Range(func(k *question, _ bool) bool {
		questions.Delete(k)
		return true
	})
}
