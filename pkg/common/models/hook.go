// Package models
/*
 提供快捷构建钩子代码文件路径的管理
 初始化时设置pathFunc和enabledFunc并在运行时返回真实的结果
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

type (
	HookOptions map[consts.MiddlewareHook]*hookOption
	hookOption  struct {
		Path        string `json:"path"`
		Existed     bool   `json:"existed"`
		Enabled     bool   `json:"enabled"`
		pathFunc    func(string, ...string) string
		enabledFunc func(string, ...string) bool
	}
)

func buildHookOption[T any](modelText *fileloader.ModelText[T]) *hookOption {
	return &hookOption{pathFunc: modelText.GetPath, enabledFunc: modelText.Enabled}
}

func getHookOptionResultMap(optionMap HookOptions, dataName string, optional ...string) (result HookOptions) {
	result = make(HookOptions)
	for hook, option := range optionMap {
		result[hook] = getHookOptionResultItem(option, dataName, optional...)
	}
	return
}

func getHookOptionResultItem(option *hookOption, dataName string, optional ...string) *hookOption {
	hookPath := option.pathFunc(dataName, optional...)
	return &hookOption{
		Path:    hookPath,
		Existed: !utils.NotExistFile(hookPath),
		Enabled: option.enabledFunc(dataName, optional...),
	}
}

func GetHookServerUrl() string {
	return utils.GetVariableString(configs.GlobalSettingRoot.FirstData().ServerOptions.ServerUrl)
}
