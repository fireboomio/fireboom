// Package models
/*
 使用fileloader.Model管理身份验证配置
 读取store/authentication下的文件，变更后会触发引擎编译，支持逻辑删除
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

type (
	Authentication struct {
		Name       string `json:"name"`
		Enabled    bool   `json:"enabled"`
		CreateTime string `json:"createTime"`
		UpdateTime string `json:"updateTime"`
		DeleteTime string `json:"deleteTime"`

		Issuer              *wgpb.ConfigurationVariable `json:"issuer"`
		OidcConfigEnabled   bool                        `json:"oidcConfigEnabled"`
		OidcConfig          *AuthenticationOidcConfig   `json:"oidcConfig"`
		JwksProviderEnabled bool                        `json:"jwksProviderEnabled"`
		JwksProvider        *AuthenticationJwksProvider `json:"jwksProvider"`
	}
	AuthenticationOidcConfig struct {
		ClientId        *wgpb.ConfigurationVariable         `json:"clientId"`
		ClientSecret    *wgpb.ConfigurationVariable         `json:"clientSecret"`
		QueryParameters []*wgpb.OpenIDConnectQueryParameter `json:"queryParameters"`
	}
	AuthenticationJwksProvider struct {
		JwksJson                *wgpb.ConfigurationVariable `json:"jwksJson"`
		UserInfoCacheTtlSeconds int64                       `json:"userInfoCacheTtlSeconds"`
	}
)

var AuthenticationRoot *fileloader.Model[Authentication]

func init() {
	AuthenticationRoot = &fileloader.Model[Authentication]{
		Root:      utils.NormalizePath(consts.RootStore, consts.StoreAuthenticationParent),
		Extension: fileloader.ExtJson,
		DataHook: &fileloader.DataHook[Authentication]{
			OnInsert: func(item *Authentication) error {
				item.CreateTime = utils.TimeFormatNow()
				return nil
			},
			OnUpdate: func(_, dst *Authentication, user string) error {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				return nil
			},
		},
		DataRW: &fileloader.MultipleDataRW[Authentication]{
			GetDataName: func(item *Authentication) string { return item.Name },
			SetDataName: func(item *Authentication, name string) { item.Name = name },
			Filter:      func(item *Authentication) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Authentication) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}

	utils.RegisterInitMethod(20, func() {
		AuthenticationRoot.Init()
		configs.AddFileLoaderQuestionCollector(AuthenticationRoot.GetModelName(), func(dataName string) map[string]any {
			data, _ := AuthenticationRoot.GetByDataName(dataName)
			if data == nil {
				return nil
			}

			return map[string]any{fieldEnabled: data.Enabled}
		})
		utils.AddBuildAndStartFuncWatcher(func(f func()) { AuthenticationRoot.DataHook.AfterMutate = f })
	})
}
