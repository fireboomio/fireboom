// Package configs
/*
 添加日志自定义处理的支持
 通过AddCollector函数，设置匹配等级、标识字段、处理函数实现自定义日志处理
 处理函数需要按照约定格式返回，并最终统一发送到websocket
 通过添加自定义处理函数，对日志的中间处理，零依赖实现的问题、运行状态等信息收集
*/
package configs

import (
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
	"sync"
)

var (
	logCollectors                  []*LogCollector
	collectorMutex                 = &sync.Mutex{}
	AddFileLoaderQuestionCollector func(string, func(string) map[string]any)
)

type LogCollector struct {
	MatchLevel    []zapcore.Level // 匹配日志等级
	IdentifyField string          // 关键词字段
	Handle        func(zapcore.Entry, *zapcore.Field, map[string]*zapcore.Field) *WsMsgBody
}

func AddCollector(c *LogCollector) {
	collectorMutex.Lock()
	defer collectorMutex.Unlock()

	logCollectors = append(logCollectors, c)
}

func analysis(entry zapcore.Entry, fields []zapcore.Field) error {
	if len(fields) == 0 {
		return nil
	}

	fieldMap := make(map[string]*zapcore.Field)
	for _, field := range fields {
		itemField := field
		fieldMap[field.Key] = &itemField
	}

	handlerCollectors(entry, logCollectors, fieldMap)
	return nil
}

func handlerCollectors(entry zapcore.Entry, collectors []*LogCollector, fieldMap map[string]*zapcore.Field) {
	for _, collector := range collectors {
		if !slices.Contains(collector.MatchLevel, entry.Level) {
			continue
		}

		value, ok := fieldMap[collector.IdentifyField]
		if !ok {
			continue
		}

		delete(fieldMap, value.Key)
		result := collector.Handle(entry, value, fieldMap)
		WebsocketInstance.WriteWsMsgBodyForAll(result)
	}
}
