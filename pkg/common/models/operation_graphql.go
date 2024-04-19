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
)

var OperationGraphql *fileloader.ModelText[Operation]

func init() {
	OperationGraphql = &fileloader.ModelText[Operation]{
		Title:             "operation.graphql",
		Root:              utils.NormalizePath(consts.RootStore, consts.StoreOperationParent),
		Extension:         fileloader.ExtGraphql,
		ReadCacheRequired: true,
		TextRW: &fileloader.MultipleTextRW[Operation]{
			Enabled: func(item *Operation, _ ...string) bool { return item.Enabled },
			Name:    fileloader.DefaultBasenameFunc(),
		},
	}

	utils.RegisterInitMethod(20, func() {
		OperationGraphql.RelyModel = OperationRoot
		OperationGraphql.Init()
	})
}
