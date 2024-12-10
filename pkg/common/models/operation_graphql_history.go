// Package models
/*
 使用fileloader.ModelText管理graphql文本
 依赖与父model graphql实现多文件管理
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"strings"
)

var OperationGraphqlHistory *fileloader.ModelText[Operation]

func init() {
	OperationGraphqlHistory = &fileloader.ModelText[Operation]{
		Title:               "operation.graphql.history",
		Root:                utils.NormalizePath(consts.RootStore, consts.StoreOperationParent),
		Extension:           fileloader.ExtGraphql,
		SkipRelyModelUpdate: true,
		TextRW: &fileloader.MultipleTextRW[Operation]{
			Enabled: func(item *Operation, _ ...string) bool { return true },
			Name: func(dataName string, _ int, elem ...string) (string, bool) {
				path := make([]string, len(elem)+1)
				if lastSplitIndex := strings.LastIndexByte(dataName, '/'); lastSplitIndex != -1 {
					dataName = dataName[:lastSplitIndex+1] + "." + dataName[lastSplitIndex+1:]
				} else {
					dataName = "." + dataName
				}
				path[0] = dataName
				copy(path[1:], elem)
				return utils.NormalizePath(path...), true
			},
		},
	}

	utils.RegisterInitMethod(20, func() {
		OperationGraphqlHistory.RelyModel = OperationRoot
		OperationGraphqlHistory.Init()
	})
}
