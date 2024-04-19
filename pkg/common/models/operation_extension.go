// Package models
/*
 使用fileloader.ModelText管理function, proxy接口配置
 当钩子配置变更时，自动切换父目录和重置路径字典
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"strings"
)

var (
	OperationFunction *fileloader.ModelText[Operation]
	OperationProxy    *fileloader.ModelText[Operation]
)

func buildOperationExtension(parent string) *fileloader.ModelText[Operation] {
	item := &fileloader.ModelText[Operation]{
		Title:     parent,
		Extension: fileloader.ExtJson,
		TextRW: &fileloader.MultipleTextRW[Operation]{
			Enabled: func(item *Operation, _ ...string) bool { return item.Enabled },
			Name: func(dataName string, _ int, _ ...string) (string, bool) {
				return strings.TrimPrefix(dataName, parent+"/"), true
			},
		},
	}
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = OperationRoot
		item.Init()
	})
	AddSwitchServerSdkWatcher(func(outputPath, _, _ string, upperFirstBasename bool) {
		if outputPath != "" {
			outputPath = utils.NormalizePath(outputPath, parent)
		}
		item.Root = outputPath
		item.UpperFirstBasename = upperFirstBasename
		item.ResetRootDirectory()
	})
	return item
}

func init() {
	OperationFunction = buildOperationExtension(consts.HookFunctionParent)
	OperationProxy = buildOperationExtension(consts.HookProxyParent)
}
