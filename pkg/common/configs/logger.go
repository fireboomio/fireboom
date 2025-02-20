// Package configs
/*
 通过添加writeSyncer，日志会输出到控制台、websocket及日志文件
 控制台输出的日志会保留颜色，其他输出会去除
 不同的启动模式，日志有不同的默认配置
 通过levelEnabler函数实现日志级别的运行时切换
 替换全局日志记录器，使得可以在全局引用同时去依赖
*/
package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"github.com/wundergraph/wundergraph/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"regexp"
	"sync"
	"time"
)

var (
	loggerLevel        zapcore.Level
	loggerWriteSyncers []zapcore.WriteSyncer
	colorRegexp        = regexp.MustCompile(`\x1b\[[^m]+m([^ ]+)\x1b\[0m`)
	lazyLogger         func() *zap.Logger
)

func removeColorText(p []byte) []byte {
	return []byte(colorRegexp.ReplaceAllString(string(p), "$1"))
}

func addLoggerWriteSyncer(writer zapcore.WriteSyncer) {
	loggerWriteSyncers = append(loggerWriteSyncers, writer)
}

func init() {
	lazyLoggerMutex := sync.Mutex{}
	lazyLoggerMutex.Lock()
	lazyLogger = func() *zap.Logger {
		lazyLoggerMutex.Lock()
		defer lazyLoggerMutex.Unlock()
		return zap.L()
	}
	utils.RegisterInitMethod(13, func() {
		var zapOptions []zap.Option
		var defaultEncoderConfig zapcore.EncoderConfig
		if utils.GetBoolWithLockViper(consts.DevMode) {
			defaultEncoderConfig = zap.NewDevelopmentEncoderConfig()
			loggerLevel = zapcore.DebugLevel
			host, _ := os.Hostname()
			zapOptions = append(zapOptions,
				zap.AddCaller(),
				zap.AddStacktrace(zap.ErrorLevel),
				zap.Fields(zap.String("hostname", host), zap.Int("pid", os.Getpid())))
		} else {
			defaultEncoderConfig = zap.NewProductionEncoderConfig()
		}
		if utils.GetBoolWithLockViper(consts.EnableWebConsole) {
			zapOptions = append(zapOptions, zap.WrapCore(func(core zapcore.Core) zapcore.Core { return registerHooks(core, analysis) }))
		}

		defaultEncoderConfig.ConsoleSeparator = " "
		defaultEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		defaultEncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)

		loggerWriteSyncers = append(loggerWriteSyncers, zapcore.AddSync(os.Stdout))
		var levelEnablerFunc zap.LevelEnablerFunc
		levelEnablerFunc = func(level zapcore.Level) bool {
			return loggerLevel.Enabled(level)
		}
		// 创建一个新的核心，将日志输出到控制台和 WebSocket
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(defaultEncoderConfig),
			zapcore.NewMultiWriteSyncer(loggerWriteSyncers...),
			levelEnablerFunc,
		)

		// 替换全局的日志记录器
		zap.ReplaceGlobals(zap.New(consoleCore, zapOptions...))
		lazyLoggerMutex.Unlock()
	})
}

func setLoggerLevel(level string) {
	parseLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return
	}

	loggerLevel = parseLevel
	logging.SetLogLevel(loggerLevel)
}
