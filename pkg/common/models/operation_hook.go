// Package models
/*
 使用fileloader.ModelText管理钩子工作目录下代码文件
 读取钩子工作目录/operation下的代码文件
 当钩子配置变更时，自动切换父目录和重置路径字典
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"sync"
)

var (
	OperationHookOptionMap HookOptions
	operationHookOnce      sync.Once
)

func GetOperationHookOptions(dataName string) (result HookOptions) {
	return getHookOptionResultMap(OperationHookOptionMap, dataName)
}

func buildOperationHook(hook consts.MiddlewareHook, enabledFunc func(item *wgpb.OperationHooksConfiguration) bool) {
	item := &fileloader.ModelText[Operation]{
		Title: string(hook),
		TextRW: &fileloader.MultipleTextRW[Operation]{
			Name: fileloader.DefaultBasenameFunc(string(hook)),
			Enabled: func(item *Operation, _ ...string) bool {
				return item.Enabled && item.HooksConfiguration != nil && enabledFunc(item.HooksConfiguration)
			},
		},
	}
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = OperationRoot
		item.Init()
	})
	AddSwitchServerSdkWatcher(func(outputPath, codePackage, extension string, upperFirstBasename bool) {
		item.Extension = fileloader.Extension(extension)
		if outputPath != "" {
			if codePackage != "" {
				outputPath = utils.NormalizePath(outputPath, codePackage)
			}
			outputPath = utils.NormalizePath(outputPath, consts.HookOperationParent)
		}
		item.Root = outputPath
		item.UpperFirstBasename = upperFirstBasename
		item.ResetRootDirectory()
		operationHookOnce.Do(func() {
			OperationRoot.AddRenameAction(func(srcDataName, _ string) error {
				return removeSdkEmptyDir(outputPath, srcDataName)
			})
			OperationRoot.AddRemoveAction(func(dataName string) error {
				return removeSdkEmptyDir(outputPath, dataName)
			})
		})
	})
	OperationHookOptionMap[hook] = buildHookOption(item)
}

func init() {
	OperationHookOptionMap = make(HookOptions)

	buildOperationHook(consts.PreResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.PreResolve
	})
	buildOperationHook(consts.MutatingPreResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.MutatingPreResolve
	})
	buildOperationHook(consts.MockResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.MockResolve != nil && item.MockResolve.Enabled
	})
	buildOperationHook(consts.CustomResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.CustomResolve
	})
	buildOperationHook(consts.PostResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.PostResolve
	})
	buildOperationHook(consts.MutatingPostResolve, func(item *wgpb.OperationHooksConfiguration) bool {
		return item.MutatingPostResolve
	})

	operationHttpTransportMap := map[consts.MiddlewareHook]func(item *wgpb.OperationHooksConfiguration) bool{
		consts.HttpTransportBeforeRequest: func(item *wgpb.OperationHooksConfiguration) bool { return item.HttpTransportBeforeRequest },
		consts.HttpTransportAfterResponse: func(item *wgpb.OperationHooksConfiguration) bool { return item.HttpTransportAfterResponse },
		consts.HttpTransportOnRequest:     func(item *wgpb.OperationHooksConfiguration) bool { return item.HttpTransportOnRequest },
		consts.HttpTransportOnResponse:    func(item *wgpb.OperationHooksConfiguration) bool { return item.HttpTransportOnResponse },
	}

	for hook, option := range HttpTransportHookOptionMap {
		hookName := hook
		optionCopy := *option
		OperationHookOptionMap[hookName] = &hookOption{
			pathFunc: func(string, ...string) string {
				return optionCopy.pathFunc(string(hookName))
			},
			enabledFunc: func(dataName string, _ ...string) bool {
				data, _ := OperationRoot.GetByDataName(dataName)
				return data != nil && data.HooksConfiguration != nil && operationHttpTransportMap[hookName](data.HooksConfiguration)
			},
		}
	}
}
