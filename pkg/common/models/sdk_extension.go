// Package models
/*
 使用fileloader.ModelText管理钩子的模板/输出目录
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

func init() {
	sdkOutput := &fileloader.ModelText[Sdk]{
		Title:                  "sdk.output",
		ExtensionIgnored:       true,
		RelyModelActionIgnored: true,
		TextRW: &fileloader.MultipleTextRW[Sdk]{
			Enabled: func(item *Sdk, _ ...string) bool { return item.Enabled },
			Name: func(dataName string, _ int, _ ...string) (path string, extension bool) {
				data, err := SdkRoot.GetByDataName(dataName)
				if err != nil {
					return
				}

				path = data.OutputPath
				return
			},
		},
	}

	sdkTemplate := &fileloader.ModelText[Sdk]{
		Title:            "sdk.template",
		Root:             consts.RootTemplate,
		ExtensionIgnored: true,
		TextRW: &fileloader.MultipleTextRW[Sdk]{
			Enabled: func(item *Sdk, _ ...string) bool { return item.Enabled },
			Name: func(dataName string, _ int, _ ...string) (string, bool) {
				return dataName, false
			},
		},
	}

	utils.RegisterInitMethod(20, func() {
		sdkOutput.RelyModel = SdkRoot
		sdkTemplate.RelyModel = SdkRoot
		sdkOutput.Init()
		sdkTemplate.Init()
	})
}
