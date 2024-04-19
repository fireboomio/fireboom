// Package models
/*
 使用fileloader.Model管理GlobalOperation配置
 读取store/config/global_setting.json文件，变更后会触发引擎编译
 全局的operation配置，用来作为operation的部分配置的默认值
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

type GlobalOperation struct {
	CacheConfig              *wgpb.OperationCacheConfig                                 `json:"cacheConfig"`
	LiveQueryConfig          *wgpb.OperationLiveQueryConfig                             `json:"liveQueryConfig"`
	AuthenticationConfigs    map[wgpb.OperationType]*wgpb.OperationAuthenticationConfig `json:"authenticationConfigs"`
	ApiAuthenticationHooks   map[consts.MiddlewareHook]bool                             `json:"apiAuthenticationHooks"`
	GlobalHttpTransportHooks map[consts.MiddlewareHook]bool                             `json:"globalHttpTransportHooks"`
}

var (
	globalOperationDefaultText *fileloader.ModelText[GlobalOperation]
	GlobalOperationRoot        *fileloader.Model[GlobalOperation]
)

func init() {
	globalOperationDefaultText = &fileloader.ModelText[GlobalOperation]{
		Root:      embed.DefaultRoot,
		Extension: fileloader.ExtJson,
		TextRW:    &fileloader.EmbedTextRW{EmbedFiles: &embed.DefaultFs, Name: consts.GlobalOperation},
	}

	utils.RegisterInitMethod(20, func() {
		globalOperationDefaultText.Init()

		GlobalOperationRoot = &fileloader.Model[GlobalOperation]{
			Root:      utils.NormalizePath(consts.RootStore, consts.StoreConfigParent),
			Extension: fileloader.ExtJson,
			DataHook:  &fileloader.DataHook[GlobalOperation]{},
			DataRW: &fileloader.SingleDataRW[GlobalOperation]{
				InitDataBytes: globalOperationDefaultText.GetFirstCache(),
				DataName:      consts.GlobalOperation,
			},
		}
		GlobalOperationRoot.Init()
		utils.AddBuildAndStartFuncWatcher(func(f func()) { GlobalOperationRoot.DataHook.AfterMutate = f })
	})
}
