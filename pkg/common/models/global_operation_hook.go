// Package models
/*
 使用fileloader.ModelText管理钩子工作目录下的代码文件
 读取钩子工作目录/global下的代码文件
 提供全局钩子代码文件存在、开启、路径等属性的返回
 当钩子配置变更时，自动切换父目录和重置路径字典
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

var (
	HttpTransportHookAliasMap = map[consts.MiddlewareHook]string{
		consts.HttpTransportBeforeRequest: "httpTransportBeforeRequest",
		consts.HttpTransportAfterResponse: "httpTransportAfterResponse",
		consts.HttpTransportOnRequest:     "httpTransportOnRequest",
		consts.HttpTransportOnResponse:    "httpTransportOnResponse",
	}
	HttpTransportHookOptionMap  HookOptions
	AuthenticationHookOptionMap HookOptions
)

func GetHttpTransportHookOptions() HookOptions {
	return getHookOptionResultMap(HttpTransportHookOptionMap, consts.GlobalOperation)
}

func GetAuthenticationHookOptions() HookOptions {
	return getHookOptionResultMap(AuthenticationHookOptionMap, consts.GlobalOperation)
}

func buildHttpTransportHook(hook consts.MiddlewareHook) {
	item := &fileloader.ModelText[GlobalOperation]{
		TextRW: &fileloader.SingleTextRW[GlobalOperation]{
			Name: string(hook),
			Enabled: func(item *GlobalOperation, _ ...string) bool {
				enabled, ok := item.GlobalHttpTransportHooks[hook]
				return ok && enabled
			},
		},
	}
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = GlobalOperationRoot
		item.Init()
	})
	AddSwitchServerSdkWatcher(func(outputPath, codePackage, extension string, upperFirstBasename bool) {
		item.Extension = fileloader.Extension(extension)
		if outputPath != "" {
			if codePackage != "" {
				outputPath = utils.NormalizePath(outputPath, codePackage)
			}
			outputPath = utils.NormalizePath(outputPath, consts.HookGlobalParent)
		}
		item.Root = outputPath
		item.UpperFirstBasename = upperFirstBasename
		item.ResetRootDirectory()
	})
	if hook != consts.WsTransportOnConnectionInit {
		HttpTransportHookOptionMap[hook] = buildHookOption(item)
	}
}

func buildAuthenticationHook(hook consts.MiddlewareHook) {
	item := &fileloader.ModelText[GlobalOperation]{
		TextRW: &fileloader.SingleTextRW[GlobalOperation]{
			Name: string(hook),
			Enabled: func(item *GlobalOperation, _ ...string) bool {
				enabled, ok := item.ApiAuthenticationHooks[hook]
				return ok && enabled
			},
		},
	}
	AuthenticationHookOptionMap[hook] = buildHookOption(item)
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = GlobalOperationRoot
		item.Init()
	})
	AddSwitchServerSdkWatcher(func(outputPath, codePackage, extension string, upperFirstBasename bool) {
		item.Extension = fileloader.Extension(extension)
		if outputPath != "" {
			if codePackage != "" {
				outputPath = utils.NormalizePath(outputPath, codePackage)
			}
			outputPath = utils.NormalizePath(outputPath, consts.HookAuthenticationParent)
		}
		item.Root = outputPath
		item.UpperFirstBasename = upperFirstBasename
		item.ResetRootDirectory()
	})
}

func init() {
	HttpTransportHookOptionMap = make(HookOptions)
	AuthenticationHookOptionMap = make(HookOptions)

	buildHttpTransportHook(consts.HttpTransportBeforeRequest)
	buildHttpTransportHook(consts.HttpTransportAfterResponse)
	buildHttpTransportHook(consts.HttpTransportOnRequest)
	buildHttpTransportHook(consts.HttpTransportOnResponse)
	buildHttpTransportHook(consts.WsTransportOnConnectionInit)

	buildAuthenticationHook(consts.PostAuthentication)
	buildAuthenticationHook(consts.MutatingPostAuthentication)
	buildAuthenticationHook(consts.RevalidateAuthentication)
	buildAuthenticationHook(consts.PostLogout)
}
