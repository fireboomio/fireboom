// Package models
/*
 使用fileloader.Model管理角色配置
 读取store/role下的文件，支持逻辑删除，变更后会触发引擎编译
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

type Role struct {
	Code       string `json:"code"`
	Remark     string `json:"remark"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
	DeleteTime string `json:"deleteTime"`
}

var RoleRoot *fileloader.Model[Role]

func init() {
	RoleRoot = &fileloader.Model[Role]{
		Root:      utils.NormalizePath(consts.RootStore, consts.StoreRoleParent),
		Extension: fileloader.ExtJson,
		DataHook: &fileloader.DataHook[Role]{
			OnInsert: func(item *Role) error {
				item.CreateTime = utils.TimeFormatNow()
				return nil
			},
			OnUpdate: func(_, dst *Role, user string) error {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				return nil
			},
		},
		DataRW: &fileloader.MultipleDataRW[Role]{
			GetDataName: func(item *Role) string { return item.Code },
			SetDataName: func(item *Role, name string) { item.Code = name },
			Filter:      func(item *Role) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Role) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}

	utils.RegisterInitMethod(20, func() {
		RoleRoot.Init()
		utils.AddBuildAndStartFuncWatcher(func(f func()) { RoleRoot.DataHook.AfterMutate = f })
	})
}
