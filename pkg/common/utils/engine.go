package utils

import (
	"fireboom-server/pkg/common/consts"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
)

var (
	buildAndStartFuncWatchers []func(func())
	RandomIdentifyCode        = RandStr(32)

	// BuildAndStart 引擎编译函数
	BuildAndStart func()
	// InvokeFunctionLimit license触发限制函数
	InvokeFunctionLimit func(string, ...int) bool
	// ReloadPrismaCache 刷新prisma缓存函数
	ReloadPrismaCache func(string) error
	// HookReportFunc hook报告函数
	HookReportFunc func()
)

// EngineStarted 引擎是否启动
func EngineStarted() bool {
	return GetStringWithLockViper(consts.EngineStatusField) == consts.EngineStartSucceed
}

// AddBuildAndStartFuncWatcher 添加对引擎编译函数的监听
func AddBuildAndStartFuncWatcher(watcher func(func())) {
	buildAndStartFuncWatchers = append(buildAndStartFuncWatchers, watcher)
}

// CallBuildAndStartFuncWatchers 处理所有对编译函数的监听
func CallBuildAndStartFuncWatchers() {
	for _, watcher := range buildAndStartFuncWatchers {
		watcher(BuildAndStart)
	}
}

// GetVariableString 根据kind不同读取不同变量
func GetVariableString(variable *wgpb.ConfigurationVariable, lazyLogger ...func() *zap.Logger) string {
	if variable == nil {
		return ""
	}

	switch variable.Kind {
	case wgpb.ConfigurationVariableKind_ENV_CONFIGURATION_VARIABLE:
		envName := variable.EnvironmentVariableName
		value := GetStringWithLockViper(envName)
		if value != "" {
			return value
		}

		if value = variable.EnvironmentVariableDefaultValue; value == "" {
			// 此日志打印会通过日志收集器发送问题到websocket
			printFunc := func(logger *zap.Logger) {
				logger.Warn("missing env, please set default or real value", zap.String("env", envName))
			}
			if len(lazyLogger) > 0 {
				go func() { printFunc(lazyLogger[0]()) }()
			} else {
				printFunc(zap.L())
			}
		}
		return value
	case wgpb.ConfigurationVariableKind_STATIC_CONFIGURATION_VARIABLE:
		return ReplacePlaceholderFromEnv(variable.StaticVariableContent)
	case wgpb.ConfigurationVariableKind_PLACEHOLDER_CONFIGURATION_VARIABLE:
		return variable.PlaceholderVariableName
	default:
		return ""
	}
}

// MakeStaticVariable 构建静态变量
func MakeStaticVariable(val string) *wgpb.ConfigurationVariable {
	return &wgpb.ConfigurationVariable{StaticVariableContent: val}
}

// MakeEnvironmentVariable 构建环境变量
func MakeEnvironmentVariable(name, defaultVal string) *wgpb.ConfigurationVariable {
	return &wgpb.ConfigurationVariable{
		Kind:                            wgpb.ConfigurationVariableKind_ENV_CONFIGURATION_VARIABLE,
		EnvironmentVariableName:         name,
		EnvironmentVariableDefaultValue: defaultVal,
	}
}

// MakePlaceHolderVariable 构建占位变量
func MakePlaceHolderVariable(val string) *wgpb.ConfigurationVariable {
	return &wgpb.ConfigurationVariable{PlaceholderVariableName: val, Kind: wgpb.ConfigurationVariableKind_PLACEHOLDER_CONFIGURATION_VARIABLE}
}

// LastIndex 最后一个元素的索引
func LastIndex[E comparable](s []E, v E) (index int) {
	index = -1
	for i, item := range s {
		if item == v {
			index = i
		}
	}
	return
}
