// Package configs
/*
 通过lumberjack.Logger和globalSetting来提供对日志配置的支持
 收到pull事件会搜索引擎准备时间（dev为编译开始，start为start开始）后的日志发送到websocket
*/
package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"gopkg.in/natefinch/lumberjack.v2"
	"time"
)

const logChannel WsChannel = "log"

type lumberjackLogger lumberjack.Logger

func (l *lumberjackLogger) Write(p []byte) (n int, err error) {
	return (*lumberjack.Logger)(l).Write(removeColorText(p))
}

func (l *lumberjackLogger) Sync() error {
	return nil
}

func (l *lumberjackLogger) initConsoleLogger() {
	consoleLoggerText := &fileloader.ModelText[any]{
		Root:             utils.StringDot,
		ExtensionIgnored: true,
		TextRW:           &fileloader.SingleTextRW[any]{Name: l.Filename},
	}
	consoleLoggerText.Init()

	WsMsgHandlerMap[logChannel] = func(msg *WsMsgBody) any {
		switch msg.Event {
		case PullEvent:
			prepareTime := utils.GetTimeWithLockViper(consts.EnginePrepareTime).Format(time.RFC3339Nano)
			content, _ := consoleLoggerText.ReadWithSearch(consoleLoggerText.Title, prepareTime)
			return content
		}
		return nil
	}
}
