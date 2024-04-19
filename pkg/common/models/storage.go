// Package models
/*
 使用fileloader.Model管理存储配置
 读取store/storage下的文件，支持逻辑删除，变更后会触发引擎编译
 每个storage可以添加多个profile来实现上传的验证和自定义逻辑处理
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

type Storage struct {
	Enabled    bool   `json:"enabled"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
	DeleteTime string `json:"deleteTime"`

	wgpb.S3UploadConfiguration
}

var StorageRoot *fileloader.Model[Storage]

func init() {
	StorageRoot = &fileloader.Model[Storage]{
		Root:      utils.NormalizePath(consts.RootStore, consts.StoreStorageParent),
		Extension: fileloader.ExtJson,
		DataHook: &fileloader.DataHook[Storage]{
			OnInsert: func(item *Storage) error {
				item.CreateTime = utils.TimeFormatNow()
				return nil
			},
			OnUpdate: func(_, dst *Storage, user string) error {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				return nil
			},
			AfterInsert: func(item *Storage, _ string) bool { return item.Enabled },
			AfterRename: func(src, _ *Storage, _ string) {
				ClientCache.removeClient(src.Name)
			},
			AfterDelete: func(dataName string, _ string) {
				ClientCache.removeClient(dataName)
			},
		},
		DataRW: &fileloader.MultipleDataRW[Storage]{
			GetDataName: func(item *Storage) string { return item.Name },
			SetDataName: func(item *Storage, name string) { item.Name = name },
			Filter:      func(item *Storage) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Storage) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}

	utils.RegisterInitMethod(20, func() {
		StorageRoot.Init()
		configs.AddFileLoaderQuestionCollector(StorageRoot.GetModelName(), func(dataName string) map[string]any {
			data, _ := StorageRoot.GetByDataName(dataName)
			if data == nil {
				return nil
			}

			return map[string]any{fieldEnabled: data.Enabled}
		})
		utils.AddBuildAndStartFuncWatcher(func(f func()) { StorageRoot.DataHook.AfterMutate = f })
	})
}
