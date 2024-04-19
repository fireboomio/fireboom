// Package models
/*
 使用fileloader.ModelText管理钩子生成的自定义数据源内省文本
 读取钩子工作目录/customize下的文件
 依赖与父model Datasource实现多文件管理
 当钩子配置变更时，自动切换父目录和重置路径字典
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

var DatasourceCustomize *fileloader.ModelText[Datasource]

func init() {
	DatasourceCustomize = &fileloader.ModelText[Datasource]{
		Title:     consts.HookCustomizeParent,
		Extension: fileloader.ExtJson,
		TextRW: &fileloader.MultipleTextRW[Datasource]{
			Enabled: func(item *Datasource, _ ...string) bool { return item.Enabled },
			Name:    fileloader.DefaultBasenameFunc(),
		},
	}

	utils.RegisterInitMethod(20, func() {
		DatasourceCustomize.RelyModel = DatasourceRoot
		DatasourceCustomize.Init()
	})

	AddSwitchServerSdkWatcher(func(outputPath, _, _ string, upperFirstBasename bool) {
		if outputPath != "" {
			outputPath = utils.NormalizePath(outputPath, consts.HookCustomizeParent)
		}
		DatasourceCustomize.Root = outputPath
		DatasourceCustomize.UpperFirstBasename = upperFirstBasename
		DatasourceCustomize.ResetRootDirectory()
	})
}
